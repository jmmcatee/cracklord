package kerb

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/binary"
	"github.com/jmckaskill/asn1"
	"io"
	"time"
)

// GSS requests are a bit screwy in that they are partially asn1 The format
// is:
//
// [APPLICATION 0] IMPLICIT SEQUENCE {
//	mech OBJECT IDENTIFIER
//	data of unknown type and may not be asn1
// }
//
// To decode this we manually unpack the outer header, run the mech through
// the asn1 unmarshaller and then return the rest of the data.

type gssRequest struct {
	Mechanism asn1.ObjectIdentifier
	Data      asn1.RawValue
}

var gssRequestParam = "application,tag:0"

func mustEncodeGSSWrapper(oid asn1.ObjectIdentifier, data []byte) []byte {
	req := gssRequest{
		Mechanism: oid,
		Data:      asn1.RawValue{FullBytes: data},
	}

	return mustMarshal(req, gssRequestParam)
}

func mustDecodeGSSWrapper(data []byte) (asn1.ObjectIdentifier, []byte) {
	must(len(data) >= 2)

	// GSS wrappers are optional, if they are not supplied we assume the data is KRB5
	if data[0] != 0x60 {
		return gssKrb5Oid, data
	}

	isz := int(data[1])
	data = data[2:]

	// Note for the long forms, the data len must be >= 0x80 anyways
	must(isz <= len(data))

	switch {
	case isz == 0x84:
		isz = int(data[0])<<24 + int(data[1])<<16 + int(data[2])<<8 + int(data[3])
		data = data[4:]

	case isz == 0x83:
		isz = int(data[0])<<16 + int(data[1])<<8 + int(data[2])
		data = data[3:]

	case isz == 0x82:
		isz = int(data[0])<<8 + int(data[1])
		data = data[2:]

	case isz == 0x81:
		isz = int(data[0])
		data = data[1:]

	case isz <= 0x7F:
		// short length form

	default:
		panic(ErrProtocol)
	}

	must(0 <= isz && isz <= len(data))
	data = data[:isz]

	oid := asn1.ObjectIdentifier{}
	data, err := asn1.Unmarshal(data, &oid)

	if err != nil {
		panic(err)
	}

	return oid, data
}

type replayKey struct {
	keyType        int
	key            string
	time           time.Time
	microseconds   int
	sequenceNumber uint32
}

// Connect authenticates to a remote service by sending the given ticket.
//
// If SASLAuth is used and a GSS wrapped connection is established, gssrw
// returns a wrapped version of rw that performs the integrity/confidentiality
// wrapping. If no wrapper is negotiated then gssrw is nil.
func (t *Ticket) Connect(rw io.ReadWriter, flags int) (gssrw io.ReadWriter, err error) {
	defer recoverMust(&err)

	appflags := 0
	gssflags := gssIntegrity | gssConfidential

	if (flags & MutualAuth) != 0 {
		appflags |= mutualAuth
		gssflags |= gssMutual
	}

	if (flags & SASLAuth) != 0 {
		// SASL auth requires the AP_REP always
		appflags |= mutualAuth
		gssflags |= gssMutual
		// gssWrapper handles out of order messages but does not keep
		// a replay list
		gssflags |= gssSequence
	}

	if (flags & NoConfidentiality) != 0 {
		gssflags &^= gssConfidential
	}

	if (flags & NoSecurity) == 0 {
		gssflags &^= gssConfidential | gssIntegrity
	}

	// See RFC4121 4.1.1 for the GSS fake auth checksum
	gsschk := [24]byte{}

	// 0..3 Lgth: Number of bytes in Bnd field; Currently contains hex 10
	// 00 00 00 (16, represented in little-endian form)
	binary.LittleEndian.PutUint32(gsschk[0:4], 16)

	// 4..19 Bnd: MD5 hash of channel bindings, taken over all non-null
	// components of bindings, in order of declaration. Integer fields
	// within channel bindings are represented in little-endian order for
	// the purposes of the MD5 calculation; Currently left as 0.

	// 20..23 Flags: Bit vector of context-establishment flags, with
	// values consistent with RFC-1509, p. 41. The resulting bit vector is
	// encoded into bytes 20..23 in little-endian form.
	binary.LittleEndian.PutUint32(gsschk[20:24], uint32(gssflags))

	// 24..25 DlgOpt The Delegation Option identifier (=1) [optional]
	// 26..27 Dlgth: The length of the Deleg field. [optional]
	// 28..(n-1) Deleg: A KRB_CRED message (n = Dlgth + 28) [optional]
	// n..last Exts: Extensions [optional].

	subkey := mustGenerateKey(t.key.EncryptAlgo(appRequestAuthKey), rand.Reader)
	now := t.cfg.now().UTC()
	auth := authenticator{
		ProtoVersion: kerberosVersion,
		ClientRealm:  t.crealm,
		Client:       t.client,
		Time:         time.Unix(now.Unix(), 0).UTC(), // round to the nearest second
		Microseconds: now.Nanosecond() / 1000,
		Checksum:     checksumData{signGssFake, gsschk[:]},
		SubKey: encryptionKey{
			Algo: subkey.EncryptAlgo(appRequestAuthKey),
			Key:  subkey.Key(),
		},
	}

	if err := binary.Read(t.cfg.rand(), binary.BigEndian, &auth.SequenceNumber); err != nil {
		return nil, err
	}

	authdata := mustMarshal(auth, authenticatorParam)
	req := appRequest{
		ProtoVersion: kerberosVersion,
		MsgType:      appRequestType,
		Flags:        flagsToBitString(appflags),
		Ticket:       asn1.RawValue{FullBytes: t.ticket},
		Auth: encryptedData{
			Algo: t.key.EncryptAlgo(appRequestAuthKey),
			Data: t.key.Encrypt(nil, appRequestAuthKey, authdata),
		},
	}

	reqdata := mustMarshal(req, appRequestParam)
	reqdata = append([]byte{(gssAppRequest >> 8) & 0xFF, gssAppRequest & 0xFF}, reqdata...)
	gssdata := mustEncodeGSSWrapper(gssKrb5Oid, reqdata)
	mustWrite(rw, gssdata)

	// Now get the reply

	if (appflags & mutualAuth) == 0 {
		return nil, nil
	}

	brep := [4096]byte{}
	repdata := mustRead(rw, brep[:])
	oid, repdata := mustDecodeGSSWrapper(repdata)
	must(oid.Equal(gssKrb5Oid) && len(repdata) >= 2)

	gsstype := binary.BigEndian.Uint16(repdata[:2])

	switch gsstype {
	case gssAppError:
		errmsg := errorMessage{}
		mustUnmarshal(repdata[2:], &errmsg, errorParam)
		return nil, ErrRemote{&errmsg}
	case gssAppReply:
		// continue below
	default:
		panic(ErrProtocol)
	}

	rep := appReply{}
	mustUnmarshal(repdata[2:], &rep, appReplyParam)
	must(rep.ProtoVersion == kerberosVersion && rep.MsgType == appReplyType)

	erep := encryptedAppReply{}
	edata := mustDecrypt(t.key, nil, rep.Encrypted.Algo, appReplyEncryptedKey, rep.Encrypted.Data)
	mustUnmarshal(edata, &erep, encAppReplyParam)
	must(erep.ClientTime.Equal(auth.Time) && erep.ClientMicroseconds == auth.Microseconds)

	// Now non-SASL requests eg HTTP negotiate are finished.
	if (flags & SASLAuth) == 0 {
		return nil, nil
	}

	key := t.key
	if erep.SubKey.Algo != 0 {
		key = mustLoadKey(erep.SubKey.Algo, erep.SubKey.Key)
	}

	// SASL requests on the otherhand GSS_wrap all messages from now on.
	// We return a read writer for the user to be able to do this. However
	// we first exchange an intial gss_wrap exchange where we each
	// specific the sasl flags as well as the max wrap size. The server
	// starts this exchange. Both of these intial messages are not encrypted.

	g := &gssWrapper{
		// add some extra room for GSS_wrap header and GSS fake ASN1 wrapper
		rxbuf:    make([]byte, maxGSSWrapRead+64),
		rxseqnum: erep.SequenceNumber,
		txseqnum: auth.SequenceNumber,
		checkseq: (gssflags & gssSequence) != 0,
		key:      key,
		client:   true,
		conf:     false,
		rw:       rw,
	}

	repdata = mustRead(g, brep[:])
	must(len(repdata) == 4)

	availsec := int(repdata[0])
	g.maxtxsize = int(binary.BigEndian.Uint32(repdata) & 0xFFFFFF)

	sec := chooseGSSSecurity(availsec, flags)
	must(sec != 0)

	grep := [4]byte{}
	binary.BigEndian.PutUint32(grep[:], maxGSSWrapRead)
	grep[0] = byte(sec)

	mustWrite(g, grep[:])

	if sec == saslNoSecurity {
		return nil, nil
	}

	g.conf = (sec == saslConfidential)
	return g, nil
}

type gssWrapper struct {
	rxbuf              []byte
	rxseqnum, txseqnum uint32
	checkseq           bool
	maxtxsize          int
	key                key
	client, conf       bool
	rw                 io.ReadWriter
}

func direction(senderIsInitiator bool) uint32 {
	if senderIsInitiator {
		return 0
	}

	return 0xFFFFFFFF
}

func (s *gssWrapper) Read(b []byte) (n int, err error) {
	defer recoverMust(&err)

	dir := direction(!s.client)
	data := mustRead(s.rw, s.rxbuf)
	seqnum, gdata := mustGSSUnwrap(data, s.key, dir, s.conf)

	if s.checkseq {
		must(seqnum == s.rxseqnum)
		s.rxseqnum++
	}

	return copy(b, gdata), nil
}

func (s *gssWrapper) Write(b []byte) (n int, err error) {
	defer recoverMust(&err)

	for n = 0; n < len(b); n += s.maxtxsize {
		d := b[n:]
		if len(d) > s.maxtxsize {
			d = d[:s.maxtxsize]
		}

		dir := direction(s.client)
		gdata := mustGSSWrap(s.txseqnum, d, s.key, dir, s.conf)
		mustWrite(s.rw, gdata)

		s.txseqnum++
	}

	return len(b), nil
}

func chooseGSSSecurity(avail, flags int) int {
	rconf := (flags & RequireConfidentiality) != 0
	rint := (flags & RequireIntegrity) != 0
	tconf := (flags & (NoConfidentiality | NoSecurity)) == 0
	tint := (flags & NoSecurity) == 0

	aconf := (avail & saslConfidential) != 0
	aint := (avail & saslIntegrity) != 0
	anone := (avail & saslNoSecurity) != 0

	if (rconf || tconf) && aconf {
		return saslConfidential
	} else if rconf {
		return 0
	}

	if (rint || tint) && aint {
		return saslIntegrity
	} else if rint {
		return 0
	}

	if anone {
		return saslNoSecurity
	}

	return 0
}

// See RFC1964 1.2.2
func mustGSSUnwrap(gdata []byte, key key, dir uint32, conf bool) (seqnum uint32, data []byte) {
	must(len(gdata) >= 2)

	oid, idata := mustDecodeGSSWrapper(gdata)
	must(oid.Equal(gssKrb5Oid) && len(idata) >= 32)

	tok := int(binary.BigEndian.Uint16(idata[0:2]))
	signalg := int(binary.BigEndian.Uint16(idata[2:4]))
	sealalg := int(binary.BigEndian.Uint16(idata[4:6]))
	// filler for 6:8
	seqdata := idata[8:16]
	chk := idata[16:24]
	data = idata[24:]

	must(tok == gssWrap)
	must((sealalg != cryptGssNone) == conf)

	// checksum salt
	seqdata = mustDecrypt(key, chk, sealalg, gssSequenceNumber, seqdata)

	if conf {
		// sequence number salt
		data = mustDecrypt(key, seqdata[:4], sealalg, gssWrapSeal, data)
	}

	chk2 := mustSign(key, signalg, gssWrapSign, idata[:8], data)
	must(subtle.ConstantTimeCompare(chk, chk2[:8]) == 1)
	must(dir == binary.BigEndian.Uint32(seqdata[4:8]))

	// The first 8 bytes of the data is the confounder.
	// The trailing [1:8] pad bytes all have the padding size as the value
	padsz := int(data[len(data)-1])
	must(8+padsz < len(data))
	data = data[8 : len(data)-padsz]

	return binary.BigEndian.Uint32(seqdata), data
}

// See RFC1964 1.2.2
func mustGSSWrap(seqnum uint32, data []byte, key key, dir uint32, conf bool) []byte {
	signalgo := key.SignAlgo(gssWrapSign)
	sealalgo := key.EncryptAlgo(gssWrapSeal)

	if !conf {
		sealalgo = cryptGssNone
	}

	d := make([]byte, 32)
	binary.BigEndian.PutUint16(d[0:2], gssWrap)
	binary.BigEndian.PutUint16(d[2:4], uint16(signalgo))
	binary.BigEndian.PutUint16(d[4:6], uint16(sealalgo))
	binary.BigEndian.PutUint16(d[6:8], 0xFFFF) // filler
	// 8:16 is encrypted sequence number
	// 16:24 is checksum below
	binary.BigEndian.PutUint32(d[8:12], seqnum)
	binary.BigEndian.PutUint32(d[12:16], dir)

	// 24:32 is the confounder
	mustReadFull(rand.Reader, d[24:32])
	d = append(d, data...)

	// 8 byte round padding must be at least one byte
	padsz := ((len(d) + 8) &^ 7) - len(d)

	// RFC4757 (MS RC4-HMAC) violates the standard padding and wants
	// explicitely only one byte
	if key.EncryptAlgo(gssSequenceNumber) == cryptGssRc4Hmac {
		padsz = 1
	}

	for i := 0; i < padsz; i++ {
		d = append(d, byte(padsz))
	}

	// Checksum the 8 byte header, 8 byte confounder, data, and padding
	copy(d[16:24], mustSign(key, signalgo, gssWrapSign, d[:8], d[24:]))

	if conf {
		// encrypt data using sequence number salt
		copy(d[24:], key.Encrypt(d[8:12], gssWrapSeal, d[24:]))
	}

	// encrypt seqnum using checksum salt
	copy(d[8:16], key.Encrypt(d[16:24], gssSequenceNumber, d[8:16]))

	return mustEncodeGSSWrapper(gssKrb5Oid, d)
}

func (c *Credential) isReplay(auth *authenticator, etkt *encryptedTicket) bool {
	now := c.cfg.now().UTC()

	c.lk.Lock()
	defer c.lk.Unlock()

	rkey := replayKey{
		keyType:        etkt.Key.Algo,
		key:            string(etkt.Key.Key),
		time:           auth.Time,
		microseconds:   auth.Microseconds,
		sequenceNumber: auth.SequenceNumber,
	}

	if c.replay == nil {
		c.replay = make(map[replayKey]bool)
		c.lastReplayPurge = now
	}

	if _, ok := c.replay[rkey]; ok {
		return true
	}

	if now.Sub(c.lastReplayPurge) > time.Minute*10 {
		for rkey := range c.replay {
			if now.Sub(rkey.time) > time.Minute*10 {
				delete(c.replay, rkey)
			}
		}

		c.lastReplayPurge = now
	}

	c.replay[rkey] = true
	return false
}

// Accept reads in a connect request checking that it is valid for the given
// credential.
//
// If SASLAuth is requested and a GSS wrapped connection is negotiated for
// integrity/confidentiality then gssrw returns a wrapped version of rw which
// performs the wrapping.
//
// Accept also returns the user and realm that the client authenticated with
// if successful.
func (c *Credential) Accept(rw io.ReadWriter, flags int) (gssrw io.ReadWriter, user, realm string, err error) {
	// TODO send error replies
	defer recoverMust(&err)

	// Get the AP_REQ
	breq := [4096]byte{}
	reqdata := mustRead(rw, breq[:])
	oid, reqdata := mustDecodeGSSWrapper(reqdata)

	spnego := oid.Equal(gssSpnegoOid)
	if spnego {
		neg := negTokenInit{}
		mustUnmarshal(reqdata, &neg, negTokenInitParam)
		oid, reqdata = mustDecodeGSSWrapper(neg.Token)
	}

	must(oid.Equal(gssKrb5Oid) || oid.Equal(gssMsKrb5Oid))
	must(len(reqdata) >= 2 && binary.BigEndian.Uint16(reqdata[:2]) == gssAppRequest)

	req := appRequest{}
	mustUnmarshal(reqdata[2:], &req, appRequestParam)
	must(req.ProtoVersion == kerberosVersion && req.MsgType == appRequestType)

	appflags := bitStringToFlags(req.Flags)

	// Check the ticket - problems with the ticket generate ErrTicket
	// instead of ErrProtocol so that we can log out that the client just
	// sent the wrong or an expired ticket (a corrupt ticket still
	// generates a ErrProtocol though)

	tkt := ticket{}
	mustUnmarshal(req.Ticket.FullBytes, &tkt, ticketParam)
	must(tkt.ProtoVersion == kerberosVersion)

	if tkt.Realm != c.realm || !nameEquals(tkt.Service, c.principal) {
		panic(ErrTicket{"wrong service"})
	}

	if c.kvno != 0 && c.kvno != tkt.Encrypted.KeyVersion {
		panic(ErrTicket{"wrong key version"})
	}

	etkt := encryptedTicket{}
	etktdata := mustDecrypt(c.key, nil, tkt.Encrypted.Algo, ticketKey, tkt.Encrypted.Data)
	mustUnmarshal(etktdata, &etkt, encTicketParam)

	now := c.cfg.now().UTC()
	if etkt.From != *new(time.Time) && now.Before(etkt.From.Add(-5*time.Minute)) {
		panic(ErrTicket{"not valid yet"})
	}
	if now.After(etkt.Till.Add(5 * time.Minute)) {
		panic(ErrTicket{"expired"})
	}
	if bitStringToFlags(etkt.Flags)&TicketInvalid != 0 {
		panic(ErrTicket{"invalid flag"})
	}

	tkey := mustLoadKey(etkt.Key.Algo, etkt.Key.Key)
	user = composePrincipal(etkt.Client)
	realm = etkt.ClientRealm

	// Check the authenticator

	auth := authenticator{}
	authdata := mustDecrypt(tkey, nil, req.Auth.Algo, appRequestAuthKey, req.Auth.Data)
	mustUnmarshal(authdata, &auth, authenticatorParam)

	must(auth.ProtoVersion == kerberosVersion)
	must(auth.ClientRealm == etkt.ClientRealm && nameEquals(auth.Client, etkt.Client))
	must(-5*time.Minute < now.Sub(auth.Time) && now.Sub(auth.Time) < 5*time.Minute)

	// Check the fake checksum.
	// TODO: handle forwarded credentials
	must(auth.Checksum.Algo == signGssFake && len(auth.Checksum.Data) >= 4)

	bndlen := int(binary.LittleEndian.Uint32(auth.Checksum.Data))
	must(0 <= bndlen && bndlen+8 <= len(auth.Checksum.Data))

	gssflags := binary.LittleEndian.Uint32(auth.Checksum.Data[bndlen+4:])
	must(((gssflags & gssMutual) != 0) == ((appflags & mutualAuth) != 0))

	// Now check for replays
	must(!c.isReplay(&auth, &etkt))

	// Now send the reply

	if (appflags & mutualAuth) == 0 {
		return nil, user, realm, nil
	}

	erep := encryptedAppReply{
		ClientTime:         auth.Time,
		ClientMicroseconds: auth.Microseconds,
	}

	if err := binary.Read(c.cfg.rand(), binary.BigEndian, &erep.SequenceNumber); err != nil {
		return nil, "", "", err
	}

	erepdata := mustMarshal(erep, encAppReplyParam)
	rep := appReply{
		ProtoVersion: kerberosVersion,
		MsgType:      appReplyType,
		Encrypted: encryptedData{
			Algo: tkey.EncryptAlgo(appReplyEncryptedKey),
			Data: tkey.Encrypt(nil, appReplyEncryptedKey, erepdata),
		},
	}

	repdata := mustMarshal(rep, appReplyParam)
	repdata = append([]byte{(gssAppReply >> 8) & 0xFF, gssAppReply & 0xFF}, repdata...)
	repdata = mustEncodeGSSWrapper(oid, repdata)

	if spnego {
		srep := negTokenReply{
			State:     spnegoAccepted,
			Mechanism: oid,
			Response:  repdata,
		}

		repdata = mustMarshal(srep, negTokenReplyParam)
		repdata = mustEncodeGSSWrapper(gssSpnegoOid, repdata)
	}

	mustWrite(rw, repdata)

	// Non-SASL accepts eg HTTP negotiate are now finished
	if (flags & SASLAuth) == 0 {
		return nil, user, realm, nil
	}

	// SASL accept continues on with sending a GSS_wrapped request from
	// the server to the client to negotiate the wrapping security mode.

	availsec := saslNoSecurity | saslIntegrity | saslConfidential

	// Remove modes we don't support
	if (flags & NoConfidentiality) != 0 {
		availsec &^= saslConfidential
	}
	if (flags & NoSecurity) != 0 {
		availsec &^= saslIntegrity | saslConfidential
	}

	// Remove modes where we require a higher level
	if (flags & RequireConfidentiality) != 0 {
		availsec &^= saslNoSecurity | saslIntegrity
	} else if (flags & RequireIntegrity) != 0 {
		availsec &^= saslNoSecurity
	}

	// Remove modes the client doesn't support
	if (gssflags & gssIntegrity) == 0 {
		availsec &^= saslIntegrity
	}
	if (gssflags & gssConfidential) == 0 {
		availsec &^= saslConfidential
	}

	if availsec == 0 {
		panic(ErrNoCommonAlgorithm)
	}

	g := &gssWrapper{
		// add some extra room for GSS_wrap header and GSS fake ASN1 wrapper
		rxbuf:     make([]byte, maxGSSWrapRead+64),
		rxseqnum:  auth.SequenceNumber,
		txseqnum:  erep.SequenceNumber,
		checkseq:  (gssflags & gssSequence) != 0,
		maxtxsize: maxGSSWrapRead, // fill in properly later
		key:       tkey,
		client:    false,
		conf:      false,
		rw:        rw,
	}

	gd := [4]byte{}
	binary.BigEndian.PutUint32(gd[:], maxGSSWrapRead)
	gd[0] = byte(availsec)

	mustWrite(g, gd[:])

	repdata = mustRead(g, gd[:])
	must(len(repdata) == 4)

	g.maxtxsize = int(binary.BigEndian.Uint32(repdata) & 0xFFFFFF)

	sec := int(repdata[0])
	must((sec & availsec) != 0)

	switch sec {
	case saslNoSecurity:
		g = nil
	case saslIntegrity:
	case saslConfidential:
		g.conf = true
	default:
		panic(ErrProtocol)
	}

	return g, user, realm, nil
}
