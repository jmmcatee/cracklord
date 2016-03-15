package ldap

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"github.com/jmckaskill/asn1"
	"io"
	"net"
	"net/url"
	"strconv"
	"sync"
)

type AuthInfo struct {
	RemoteAddr string
	TLS        *tls.ConnectionState
}

type AuthMechanism interface {
	MechanismName() string
	Connect(rw io.ReadWriter) (io.ReadWriter, error)
}

type ClientConfig struct {
	Dial func(net, addr string) (net.Conn, error)
	Auth []AuthMechanism
	TLS  *tls.Config
}

type status int

const (
	finished status = iota
	abandon
	data
	ignored
)

type replyHandler interface {
	onReply(tag int, data []byte) status
}

type DB struct {
	url string
	cfg *ClientConfig

	lk      sync.Mutex
	conn    net.Conn
	rw      io.ReadWriter
	closed  bool
	replies map[uint32]replyHandler
	nextId  uint32
	buffer  []byte
}

func Open(url string, cfg *ClientConfig) *DB {
	return &DB{
		url:     url,
		cfg:     cfg,
		replies: make(map[uint32]replyHandler),
	}
}

func (db *DB) Close() error {
	db.lk.Lock()
	defer db.lk.Unlock()
	if db.conn != nil {
		db.conn.Close()
		db.conn = nil
		db.rw = nil
	}
	db.closed = true
	return nil
}

func dial(network, addr string) (net.Conn, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	// SRV

	_, addrs, _ := net.LookupSRV("ldap", network, host)

	for _, a := range addrs {
		sock, err := net.Dial("tcp", net.JoinHostPort(a.Target, strconv.Itoa(int(a.Port))))
		if err == nil {
			return sock, nil
		}
	}

	// Non-SRV

	return net.Dial(network, addr)
}

func open(urlstr string, cfg *ClientConfig) (rw io.ReadWriter, sock net.Conn, err error) {
	// URL

	u, err := url.Parse(urlstr)
	if err != nil {
		return nil, nil, err
	}

	addr := u.Host
	if _, _, err := net.SplitHostPort(addr); err != nil {
		addr = net.JoinHostPort(addr, "389")
	}

	// Dial

	if cfg.Dial != nil {
		sock, err = cfg.Dial("tcp", addr)
	} else {
		sock, err = dial("tcp", addr)
	}

	if err != nil {
		return nil, nil, err
	}

	// TLS

	if cfg.TLS != nil {
		if err := sendStartTLS(sock); err != nil {
			sock.Close()
			return nil, nil, err
		}

		sock = tls.Client(sock, cfg.TLS)
	}

	// Bind

	for _, m := range cfg.Auth {
		auth := authSock{Conn: sock, mech: m.MechanismName()}
		rw, err := m.Connect(&auth)

		if err == nil && auth.waitReply {
			_, err = auth.Read(nil)
		}

		if err == nil && !auth.finished {
			err = ErrIncompleteAuth
		}

		if err == ErrAuthNotSupported {
			continue
		} else if err != nil {
			sock.Close()
			return nil, nil, err
		}

		if rw == nil {
			rw = sock
		}

		return rw, sock, nil
	}

	return sock, sock, nil
}

type SimpleAuth struct {
	User, Pass string
}

func (s SimpleAuth) MechanismName() string {
	return "SIMPLE"
}

func (s SimpleAuth) Connect(rw io.ReadWriter) (io.ReadWriter, error) {
	out := []byte(s.User)
	out = append(out, 0)
	out = append(out, s.Pass...)

	if _, err := rw.Write(out); err != nil {
		return nil, err
	}

	return nil, nil
}

type authSock struct {
	net.Conn
	mech string
	// LDAP bind requests need to be a clean request response, even if the
	// SASL mechanism doesn't want to.
	waitReply bool
	// After the bind requests have finished we send/receive SASL PDUs
	// which are a 4 byte big endian length followed by a chunk of data.
	finished bool
}

func (s *authSock) Write(b []byte) (int, error) {
	if s.finished {
		sz := [4]byte{}
		binary.BigEndian.PutUint32(sz[:], uint32(len(b)))
		if _, err := s.Conn.Write(sz[:]); err != nil {
			return 0, err
		}

		return s.Conn.Write(b)
	}

	if s.waitReply {
		if _, err := s.Read(nil); err != nil {
			return 0, err
		}
	}

	var err error

	req := bindRequest{
		Version: ldapVersion,
	}

	if s.mech == "SIMPLE" {
		idx := bytes.IndexByte(b, 0)
		req.BindDN = b[:idx]
		req.Auth.FullBytes, err = asn1.MarshalWithParams(b[idx+1:], simpleBindParam)
	} else {
		req.Auth.FullBytes, err = asn1.MarshalWithParams(saslCredentials{[]byte(s.mech), b}, saslBindParam)
	}

	if err != nil {
		return 0, err
	}

	if err := dosend(s.Conn, bindRequestParam, req); err != nil {
		return 0, err
	}

	s.waitReply = true

	return len(b), nil
}

func (s *authSock) Read(b []byte) (int, error) {
	if s.finished {
		bsz := [4]byte{}
		if _, err := io.ReadFull(s.Conn, bsz[:]); err != nil {
			return 0, err
		}

		sz := binary.BigEndian.Uint32(bsz[:])
		if sz > uint32(len(b)) {
			return 0, ErrShortRead
		}

		if _, err := io.ReadFull(s.Conn, b[:sz]); err != nil {
			return 0, err
		}

		return int(sz), nil
	}

	if !s.waitReply {
		if _, err := s.Write(nil); err != nil {
			return 0, err
		}
	}

	res := result{}

	if err := getresult(s.Conn, bindResultParam, &res); err != nil {
		return 0, err
	}

	s.waitReply = false

	switch LdapResultCode(res.Code) {
	case SaslBindInProgress:
		s.finished = false
	case SuccessError:
		s.finished = true
	case AuthMethodNotSupported:
		return 0, ErrAuthNotSupported
	default:
		return 0, ErrLdap{&res}
	}

	return copy(b, res.SaslCredentials), nil
}

func sendStartTLS(rw io.ReadWriter) error {
	req := extendedMessage{[]byte(startTLS), nil}
	res := result{}

	if err := dosend(rw, extendedRequestParam, req); err != nil {
		return err
	}

	if err := getresult(rw, extendedResultParam, &res); err != nil {
		return err
	}

	if LdapResultCode(res.Code) != SuccessError {
		return ErrLdap{&res}
	}

	if string(res.ExtendedName) != startTLS {
		return ErrProtocol
	}

	return nil
}

func dosend(rw io.ReadWriter, param string, req interface{}) error {
	rdata, err := asn1.MarshalWithParams(req, param)
	if err != nil {
		return err
	}

	msg := message{
		Id: 0,
		Data: asn1.RawValue{
			FullBytes: rdata,
		},
	}

	mdata, err := asn1.Marshal(msg)
	if err != nil {
		return err
	}

	if _, err := rw.Write(mdata); err != nil {
		return err
	}

	return nil
}

func getresult(rw io.ReadWriter, param string, res *result) error {
	// We need to be careful about only reading the response and no more
	// so this can work with start tls
	var b []byte
	var err error
	var de, he int

	if b, err = readExactly(rw, b, 2); err != nil {
		return err
	}

	if he, err = headerEnd(b); err != nil {
		return err
	}

	if b, err = readExactly(rw, b, he); err != nil {
		return err
	}

	if de, err = dataEnd(b); err != nil {
		return err
	}

	if b, err = readExactly(rw, b, de); err != nil {
		return err
	}

	msg := message{}
	if _, err := asn1.Unmarshal(b, &msg); err != nil {
		return err
	}

	if msg.Id != 0 {
		return ErrProtocol
	}

	if _, err := asn1.UnmarshalWithParams(msg.Data.FullBytes, res, param); err != nil {
		return err
	}

	return nil
}

func headerEnd(h []byte) (int, error) {
	if h[0] != universalSequenceTag {
		return 0, ErrProtocol
	}

	if h[1] == 0x80 || h[1] > 0x84 {
		return 0, ErrProtocol
	}

	if h[1] < 0x80 {
		return 2, nil
	}

	return int(h[1]) - 0x80 + 2, nil
}

func dataEnd(h []byte) (int, error) {
	switch len(h) {
	case 6:
		sz := binary.BigEndian.Uint32(h[2:])
		if sz > maxMessageSize {
			return 0, ErrProtocol
		}
		return int(sz) + 6, nil
	case 5:
		h[1] = 0
		sz := binary.BigEndian.Uint32(h[1:])
		if sz > maxMessageSize {
			return 0, ErrProtocol
		}
		return int(sz) + 5, nil
	case 4:
		return int(binary.BigEndian.Uint16(h[2:])) + 4, nil
	case 3:
		return int(h[2]) + 3, nil
	case 2:
		return int(h[1]) + 2, nil
	}

	panic("")
}

func resize(b []byte, sz int) []byte {
	if sz < cap(b) {
		return b
	}

	bn := make([]byte, len(b), sz+4096)
	copy(bn, b)
	return bn
}

func readExactly(r io.Reader, buf []byte, sz int) ([]byte, error) {
	buf = resize(buf, sz)
	n, err := io.ReadFull(r, buf[len(buf):sz])
	return buf[:len(buf)+n], err
}

func readAtLeast(r io.Reader, buf []byte, sz int) ([]byte, error) {
	buf = resize(buf, sz)
	n, err := io.ReadAtLeast(r, buf[len(buf):cap(buf)], sz-len(buf))
	return buf[:len(buf)+n], err
}

func mustMarshal(val interface{}, param string) []byte {
	data, err := asn1.MarshalWithParams(val, param)
	if err != nil {
		panic(err)
	}
	return data
}

// TODO: figure out better routing of read errors
func (db *DB) rxThread(rw io.ReadWriter, conn net.Conn) {
	defer func() {
		conn.Close()
		db.lk.Lock()
		if db.conn == conn {
			db.conn = nil
			db.rw = nil
		}
		db.lk.Unlock()
	}()

	var b []byte

	for {
		var err error
		var he, de int

		if b, err = readAtLeast(rw, b, 2); err != nil {
			return
		}

		if he, err = headerEnd(b); err != nil {
			return
		}

		if b, err = readAtLeast(rw, b, he); err != nil {
			return
		}

		if de, err = dataEnd(b[:he]); err != nil {
			return
		}

		if b, err = readAtLeast(rw, b, de); err != nil {
			return
		}

		msg := message{}
		if _, err := asn1.Unmarshal(b, &msg); err != nil {
			return
		}

		if msg.Data.Class != classApplication {
			return
		}

		db.lk.Lock()
		reply := db.replies[msg.Id]
		db.lk.Unlock()

		if reply != nil {
			switch reply.onReply(msg.Data.Tag, msg.Data.FullBytes) {
			case finished:
				db.lk.Lock()
				delete(db.replies, msg.Id)
				db.lk.Unlock()
			case abandon:
				db.lk.Lock()
				delete(db.replies, msg.Id)
				id := db.nextId
				db.nextId++
				db.lk.Unlock()

				data := mustMarshal(abandonRequest(msg.Id), abandonRequestParam)
				msg := message{
					Id: id,
					Data: asn1.RawValue{
						FullBytes: data,
					},
				}

				if _, err := rw.Write(mustMarshal(msg, "")); err != nil {
					return
				}
			}
		}

		// Note don't overwrite the early part of the buffer as the
		// handler may still be referencing it. EG []byte members in a
		// SearchTree result.
		b = b[de:]
	}
}

func (db *DB) send(param string, req interface{}, reply replyHandler) error {
	data, err := asn1.MarshalWithParams(req, param)
	if err != nil {
		return err
	}

	db.lk.Lock()
	defer db.lk.Unlock()

	if db.closed {
		return ErrClosed
	}

	if db.conn == nil {
		rw, conn, err := open(db.url, db.cfg)
		if err != nil {
			return err
		}

		db.conn = conn
		db.rw = rw
		go db.rxThread(rw, conn)
	}

	id := db.nextId
	db.nextId++

	msg := message{
		Id: id,
		Data: asn1.RawValue{
			FullBytes: data,
		},
	}

	out, err := asn1.Marshal(msg)
	if err != nil {
		return err
	}

	if _, err := db.rw.Write(out); err != nil {
		return err
	}

	if reply != nil {
		db.replies[id] = reply
	}

	return nil
}
