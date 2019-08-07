package main

import (
	"crypto/rand"
	"reflect"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	kerb "github.com/jmmcatee/gokerb"
	"github.com/jmmcatee/goldap/ad"
)

// Active Directory structure to implement the basic authenticator
type ADAuth struct {
	GroupMap map[string]string
	realm    string
}

// Function to configure the group mappying. One AD group per server group
// constant is expected.
func (a *ADAuth) Setup(mapping map[string]string) {
	a.GroupMap = mapping
	log.Debug("AD authentication setup complete")
}

// Function to configure the realm of the AD auth
func (a *ADAuth) SetRealm(realm string) {
	a.realm = strings.ToUpper(realm)
	log.WithField("realm", realm).Debug("AD authentication realm set.")
}

// Function to log in a user
func (a *ADAuth) Login(user, pass string) (User, error) {
	// Setup Credential Config
	credConf := kerb.CredConfig{
		Dial: kerb.DefaultDial,
		Now:  time.Now,
		Rand: rand.Reader,
	}

	logger := log.WithFields(log.Fields{
		"user":  user,
		"realm": a.realm,
	})

	// Verify the validity of user and password
	creds, err := kerb.NewCredential(user, a.realm, pass, &credConf)
	if err != nil {
		logger.Error("Error verifying kerberos credentials.")
		logger.Error(err)
		return User{}, err
	}
	logger.Debug("Validated kerberos credentials.")

	// Get a ticket to prove the creds are valid
	_, err = creds.GetTicket("krbtgt/"+a.realm, nil)
	if err != nil {
		logger.WithField("error", err.Error()).Error("Error gathering kerberos ticket.")
		return User{}, err
	}

	// User is valid so get group membership
	db := ad.New(creds, a.realm)

	// Get the user info from AD
	logger.Debug("Attempting to enumerate LDAP user info from AD")
	adUser, err := db.LookupPrincipal(user, a.realm)

	NewUser := User{
		Username: user,
	}

	for l, g := range a.GroupMap {
		// Check if our user is in the group
		users := a.recurseGroup(g, db)

		for _, obj := range users {
			if adUser.SAMAccountName == obj {
				// Our user is in this group so assign the cracklord access level
				NewUser.Groups = append(NewUser.Groups, l)
			}
		}
	}

	// User is logged in now
	NewUser.LogOnTime = time.Now()

	// Expiration timer is handled by the TokenStore

	return NewUser, nil
}

func (a *ADAuth) recurseGroup(group string, db *ad.DB) []string {
	var users []string
	adGroup, err := db.LookupGroup(group, a.realm)
	if err != nil {
		return []string{}
	}

	for _, obj := range adGroup.Member {
		// Get the DN
		dn, err := db.LookupDN(obj)
		if err != nil {
			return []string{}
		}

		if reflect.TypeOf(dn).String() == "*ad.Group" {
			users = append(users, a.recurseGroup(dn.(*ad.Group).SAMAccountName, db)...)
		} else {
			users = append(users, dn.(*ad.User).SAMAccountName)
		}
	}

	return users
}
