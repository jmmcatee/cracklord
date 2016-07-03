// ad provides a wrapper for querying ldap user/groups out of an active directory database.
package ad

import (
	"errors"
	"fmt"
	"github.com/jmmcatee/gokerb"
	"github.com/jmmcatee/goldap"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
)

type ldapObject struct {
	DN             ldap.ObjectDN
	ObjectClass    []string
	Mail           string
	DisplayName    string
	ObjectSID      ldap.SID
	SAMAccountName string
	Member         []ldap.ObjectDN
	Realm          string `ldap:"-"`
}

type User ldapObject
type Group ldapObject

var ErrInvalidRealm = errors.New("ldap-ad: invalid realm")
var ErrNoRealm = errors.New("ldap-ad: no realm specified")

type ErrSIDNotFound ldap.SID

func (s ErrSIDNotFound) Error() string {
	return fmt.Sprintf("ldap-ad: sid %s not found", ldap.SID(s).String())
}

type ldapMech struct {
	*DB
	addr string
}

func (c *ldapMech) MechanismName() string {
	return "GSSAPI"
}

func (c *ldapMech) dial(network, addr string) (net.Conn, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	// SRV

	_, addrs, _ := net.LookupSRV("ldap", network, host)

	for _, a := range addrs {
		c.addr = a.Target
		sock, err := net.Dial("tcp", net.JoinHostPort(a.Target, strconv.Itoa(int(a.Port))))
		if err == nil {
			return sock, nil
		}
	}

	// Non-SRV

	c.addr = host
	return net.Dial(network, addr)
}

func (c *ldapMech) Connect(rw io.ReadWriter) (io.ReadWriter, error) {
	serv := "ldap/" + c.addr
	if strings.HasSuffix(serv, ".") {
		serv = serv[:len(serv)-1]
	}

	tkt, err := c.cred.GetTicket(serv, nil)
	if err != nil {
		return nil, err
	}

	rwwrap, err := tkt.Connect(rw, kerb.SASLAuth|kerb.NoSecurity)
	if err != nil {
		return nil, err
	}

	return rwwrap, nil
}

type cacheDB struct {
	*ldap.DB
	base ldap.ObjectDN
}

// DB represents a connection to an active directory forest. Allowing you to
// lookup users, groups, etc. DB is thread safe and the lookup functions can
// be called from multiple threads.
type DB struct {
	// This is used from multiple threads

	cred *kerb.Credential // thread safe

	dbs        map[string]*cacheDB // not modified
	sidRealm   map[string]string
	realmAlias map[string]string
	cfg        ldap.ClientConfig

	// the cache maps are locked
	lk       sync.Mutex
	dnusers  map[ldap.ObjectDN]interface{}
	prusers  map[principal]*User
	prgroups map[principal]*Group
}

type principal struct {
	SAMAccountName, Realm string
}

// New creates a new active directory database connection using the specified
// kerberos credential and Windows 2000 style alias for the base realm. The
// trust chain is recursively followed on creation to find all of the domain
// aliases. If you change the trust chain at all, you need to create a new db.
func New(cred *kerb.Credential, baseAlias string) *DB {
	c := &DB{
		cred:       cred,
		dbs:        make(map[string]*cacheDB),
		sidRealm:   make(map[string]string),
		realmAlias: make(map[string]string),
		dnusers:    make(map[ldap.ObjectDN]interface{}),
		prusers:    make(map[principal]*User),
		prgroups:   make(map[principal]*Group),
	}

	m := &ldapMech{c, ""}
	c.cfg.Auth = []ldap.AuthMechanism{m}
	c.cfg.Dial = func(net, addr string) (net.Conn, error) {
		return m.dial(net, addr)
	}

	c.findRealms(baseAlias, cred.Realm())

	// Figure out the SID of the base realm
	parts := strings.Split(cred.Realm(), ".")
	o, _ := c.LookupDN(ldap.ObjectDN("CN=Domain Users,CN=Users,DC=" + strings.Join(parts, ",DC=")))
	if u, _ := o.(*User); u != nil {
		if dsid, err := u.ObjectSID.Domain(); err == nil {
			c.sidRealm[dsid.String()] = cred.Realm()
		}
	}

	return c
}

type trustedDomain struct {
	FlatName           string
	TrustPartner       string
	SecurityIdentifier ldap.SID
}

// findRealms resolves all trusts in a given realm filling out the sidrealms
// cache. Any realms found are followed recursively if not already in the
// cache. This should only be called on creation of a cache as it is unlocked.
// If a realm is found which is not already opened, a db is opened for it.
func (c *DB) findRealms(alias, realm string) {
	filter := ldap.Equal{"ObjectClass", []byte("trustedDomain")}
	trusts := []trustedDomain{}
	c.dbs[realm] = nil

	// Make sure we can get a ticket for this domain before trying to
	// connect. This way we can filter out realms which we have no chance
	// of connecting to.
	if _, err := c.cred.GetTicket("krbtgt/"+realm, nil); err != nil {
		return
	}

	parts := strings.Split(realm, ".")
	base := ldap.ObjectDN("DC=" + strings.Join(parts, ",DC="))
	db := ldap.Open(fmt.Sprintf("ldap://%s", realm), &c.cfg)
	c.dbs[realm] = &cacheDB{db, base}

	c.realmAlias[strings.ToUpper(alias)] = realm

	if err := db.SearchTree(&trusts, base, filter); err != nil {
		return
	}

	for _, trust := range trusts {
		realm := strings.ToUpper(trust.TrustPartner)
		if _, ok := c.dbs[realm]; ok {
			continue
		}

		if dsid, err := trust.SecurityIdentifier.Domain(); err == nil {
			c.sidRealm[dsid.String()] = realm
		}

		c.findRealms(trust.FlatName, realm)
	}
}

// LookupSID finds a SID by searching through all of the found realms,
// returning either a *User or *Group. Lookups are cached and should be
// flushed every so often by calling FlushCache.
func (c *DB) LookupSID(sid ldap.SID) (interface{}, error) {
	filter := ldap.Equal{"ObjectSID", []byte(sid)}

	dsid, err := sid.Domain()
	if err != nil {
		return nil, err
	}

	realm := c.sidRealm[dsid.String()]
	db := c.dbs[realm]
	if db == nil {
		return nil, ErrInvalidRealm
	}

	obj := ldapObject{Realm: realm}
	if err := db.SearchTree(&obj, db.base, filter); err != nil {
		return nil, err
	}

	for _, class := range obj.ObjectClass {
		switch class {
		case "group":
			return (*Group)(&obj), nil
		case "user":
			return (*User)(&obj), nil
		}
	}

	return nil, ErrSIDNotFound(sid)
}

// ResolvePrincipal converts a generic username to a user, realm pair which
// can then be used for kerberos or LookupPrincipal. Valid forms are
// user@REALM or Win 2000 style ALIAS\User or ALIAS/User. This can return
// ErrInvalidRealm if the alias can not be resolved or ErrNoRealm if no realm
// could be parsed from the user (you may want to fall back and use a default
// realm in this case).
func (c *DB) ResolvePrincipal(user string) (string, string, error) {
	// Try and parse as user@REALM
	r := strings.SplitN(user, "@", 2)
	if len(r) == 2 {
		return r[0], strings.ToUpper(r[1]), nil
	}

	// Parse as ALIAS/user
	a := strings.SplitN(user, "/", 2)
	if len(a) < 2 {
		a = strings.SplitN(user, "\\", 2)
	}

	if len(a) == 2 {
		realm, ok := c.realmAlias[strings.ToUpper(a[0])]
		if !ok {
			return "", "", ErrInvalidRealm
		}
		return a[1], realm, nil
	}

	return "", "", ErrNoRealm
}

// LookupGroup lookus up and returns a Group object for a given group and
// kerberose principal. Lookups are cached and should be flushed every so often
// by calling FlushCache.
func (c *DB) LookupGroup(group, realm string) (*Group, error) {
	gr := principal{group, realm}

	c.lk.Lock()
	g := c.prgroups[gr]
	c.lk.Unlock()

	if g != nil {
		return g, nil
	}

	filter := ldap.Equal{"SAMAccountName", []byte(group)}

	db := c.dbs[realm]
	if db == nil {
		return nil, ErrInvalidRealm
	}

	obj := ldapObject{Realm: realm}
	if err := db.SearchTree(&obj, db.base, filter); err != nil {
		return nil, err
	}

	// Verify what we got back is a group
	var isGroup bool
	for _, v := range obj.ObjectClass {
		if v == "group" {
			isGroup = true
		}
	}

	if !isGroup {
		return nil, errors.New("Returned object that is not a group.")
	}

	g = (*Group)(&obj)
	c.lk.Lock()
	c.prgroups[gr] = g
	c.lk.Unlock()

	return g, nil
}

func (c *DB) GetAllGroups(realm string) ([]Group, error) {
	filter := ldap.Equal{"objectClass", []byte("group")}

	db := c.dbs[realm]
	if db == nil {
		return nil, ErrInvalidRealm
	}

	objs := []ldapObject{}
	if err := db.SearchTree(&objs, db.base, filter); err != nil {
		return nil, err
	}

	// Verify what we got back is a group
	var g = []Group{}
	for _, obj := range objs {
		gp := (Group)(obj)
		g = append(g, gp)
	}

	return g, nil
}

// LookupPrincipal looks up and returns a User object for a given user and
// kerberos principal. If you have a Win 2000 style user name (e.g.
// AM/MyAccount) then use ResolvePrincipal first. Lookups are cached and
// should be flushed every so often by calling FlushCache.
func (c *DB) LookupPrincipal(user, realm string) (*User, error) {
	pr := principal{user, realm}

	c.lk.Lock()
	u := c.prusers[pr]
	c.lk.Unlock()

	if u != nil {
		return u, nil
	}

	filter := ldap.Equal{"SAMAccountName", []byte(user)}

	db := c.dbs[realm]
	if db == nil {
		return nil, ErrInvalidRealm
	}

	obj := ldapObject{Realm: realm}
	if err := db.SearchTree(&obj, db.base, filter); err != nil {
		return nil, err
	}

	// Verify what we got back is a group
	var isPerson bool
	for _, v := range obj.ObjectClass {
		if v == "person" {
			isPerson = true
		}
	}

	if !isPerson {
		return nil, errors.New("Returned object that is not a person.")
	}

	u = (*User)(&obj)
	c.lk.Lock()
	c.prusers[pr] = u
	c.lk.Unlock()
	return u, nil
}

func dnToRealm(dn ldap.ObjectDN) string {
	realm := []byte{}
	for _, part := range strings.Split(string(dn), ",") {
		upper := strings.ToUpper(part)
		if strings.HasPrefix(upper, "DC=") {
			if len(realm) > 0 {
				realm = append(realm, byte('.'))
			}
			realm = append(realm, upper[len("DC="):]...)
		}
	}

	return string(realm)
}

// FlushCache flushes the user/group cache and should be called every so
// often. This is thread-safe wrt the Lookup functions.
func (c *DB) FlushCache() {
	c.lk.Lock()
	c.dnusers = make(map[ldap.ObjectDN]interface{})
	c.prusers = make(map[principal]*User)
	c.prgroups = make(map[principal]*Group)
	c.lk.Unlock()
}

// LookupDN will lookup a DN returning either a *User or a *Group. Lookups are
// cached and should be flushed every so often by calling FlushCache.
func (c *DB) LookupDN(dn ldap.ObjectDN) (val interface{}, err error) {
	c.lk.Lock()
	u := c.dnusers[dn]
	c.lk.Unlock()

	if u != nil {
		return u, nil
	}

	if i := strings.Index(string(dn), ",CN=ForeignSecurityPrincipals,"); i >= 0 {
		if sid, err := ldap.ParseSID(string(dn[len("CN="):i])); err == nil {
			val, err := c.LookupSID(sid)
			return val, err
		}
	}

	realm := dnToRealm(dn)
	db := c.dbs[realm]
	if db == nil {
		return nil, ErrInvalidRealm
	}

	obj := ldapObject{Realm: realm}
	if err := db.GetObject(&obj, dn); err != nil {
		return nil, err
	}

	var ret interface{}

	for _, class := range obj.ObjectClass {
		switch class {
		case "group":
			ret = (*Group)(&obj)
		case "user":
			ret = (*User)(&obj)
		}
	}

	if ret == nil {
		return nil, ldap.ErrNotFound
	}

	c.lk.Lock()
	c.dnusers[dn] = ret
	c.lk.Unlock()
	return ret, nil
}
