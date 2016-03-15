package kerb

import (
	"bytes"
	"golang.org/x/crypto/md4"
	"crypto/cipher"
	"crypto/des"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/rc4"
	"crypto/subtle"
	"encoding/binary"
	"hash"
	"io"
	"unicode/utf16"
)

type key interface {
	// If algo is -1 then use the default
	Sign(algo, usage int, data ...[]byte) ([]byte, error)
	SignAlgo(usage int) int

	Encrypt(salt []byte, usage int, data ...[]byte) []byte
	Decrypt(salt []byte, algo, usage int, data []byte) ([]byte, error)
	EncryptAlgo(usage int) int

	Key() []byte
}

func mustSign(key key, algo, usage int, data ...[]byte) []byte {
	sign, err := key.Sign(algo, usage, data...)
	if err != nil {
		panic(err)
	}
	return sign
}

func mustDecrypt(key key, salt []byte, algo, usage int, data []byte) []byte {
	dec, err := key.Decrypt(salt, algo, usage, data)
	if err != nil {
		panic(err)
	}
	return dec
}

type rc4hmac struct {
	key []byte
}

// rc4HmacKey converts a UTF8 password into a key suitable for use with the
// rc4hmac.
func rc4HmacKey(password string) []byte {
	// Convert password from UTF8 to UTF16-LE
	s := make([]byte, 0)
	for _, r := range password {
		if r > 0x10000 {
			a, b := utf16.EncodeRune(r)
			s = append(s, byte(a), byte(a>>8), byte(b), byte(b>>8))
		} else {
			s = append(s, byte(r), byte(r>>8))
		}
	}

	h := md4.New()
	h.Write(s)
	return h.Sum(nil)
}

// RC4-HMAC has a few slight differences in the key usage values
func rc4HmacUsage(usage int) uint32 {
	switch usage {
	case asReplyClientKey:
		return 8
	case gssWrapSign:
		return 13
	}

	return uint32(usage)
}

func (c *rc4hmac) EncryptAlgo(usage int) int {
	switch usage {
	case gssWrapSeal, gssSequenceNumber:
		return cryptGssRc4Hmac
	}

	return cryptRc4Hmac
}

func (c *rc4hmac) Key() []byte {
	return c.key
}

func (c *rc4hmac) SignAlgo(usage int) int {
	switch usage {
	case gssWrapSign:
		return signGssRc4Hmac
	}

	// TODO: replace with RC4-HMAC checksum algorithm. For now we are
	// using the unkeyed RSA-MD5 checksum algorithm
	return signMd5
}

func unkeyedSign(algo, usage int, data ...[]byte) ([]byte, error) {
	var h hash.Hash

	switch algo {
	case signMd5:
		h = md5.New()
	case signMd4:
		h = md4.New()
	default:
		return nil, ErrProtocol
	}

	for _, d := range data {
		h.Write(d)
	}
	return h.Sum(nil), nil

}

var signaturekey = []byte("signaturekey\x00")

func (c *rc4hmac) Sign(algo, usage int, data ...[]byte) ([]byte, error) {
	if algo != signGssRc4Hmac && algo != signRc4Hmac {
		return unkeyedSign(algo, usage, data...)
	}

	h := hmac.New(md5.New, c.key)
	h.Write(signaturekey)
	ksign := h.Sum(nil)

	chk := md5.New()
	binary.Write(chk, binary.LittleEndian, rc4HmacUsage(usage))
	for _, d := range data {
		chk.Write(d)
	}

	h = hmac.New(md5.New, ksign)
	h.Write(chk.Sum(nil))
	return h.Sum(nil), nil
}

func (c *rc4hmac) Encrypt(salt []byte, usage int, data ...[]byte) []byte {
	switch usage {
	case gssSequenceNumber:
		// salt is the checksum
		h := hmac.New(md5.New, c.key)
		binary.Write(h, binary.LittleEndian, uint32(0))
		h = hmac.New(md5.New, h.Sum(nil))
		h.Write(salt)
		r, _ := rc4.NewCipher(h.Sum(nil))
		for _, d := range data {
			r.XORKeyStream(d, d)
		}
		return bytes.Join(data, nil)

	case gssWrapSeal:
		// salt is the sequence number in big endian
		seqnum := binary.BigEndian.Uint32(salt)
		kcrypt := make([]byte, len(c.key))
		for i, b := range c.key {
			kcrypt[i] = b ^ 0xF0
		}
		h := hmac.New(md5.New, kcrypt)
		binary.Write(h, binary.LittleEndian, seqnum)
		r, _ := rc4.NewCipher(h.Sum(nil))
		for _, d := range data {
			r.XORKeyStream(d, d)
		}
		return bytes.Join(data, nil)
	}

	// Create the output vector, layout is 0-15 checksum, 16-23 random data, 24- actual data
	outsz := 24
	for _, d := range data {
		outsz += len(d)
	}
	out := make([]byte, outsz)
	io.ReadFull(rand.Reader, out[16:24])

	// Hash the key and usage together to get the HMAC-MD5 key
	h1 := hmac.New(md5.New, c.key)
	binary.Write(h1, binary.LittleEndian, rc4HmacUsage(usage))
	K1 := h1.Sum(nil)

	// Fill in out[:16] with the checksum
	ch := hmac.New(md5.New, K1)
	ch.Write(out[16:24])
	for _, d := range data {
		ch.Write(d)
	}
	ch.Sum(out[:0])

	// Calculate the RC4 key using the checksum
	h3 := hmac.New(md5.New, K1)
	h3.Write(out[:16])
	K3 := h3.Sum(nil)

	// Encrypt out[16:] with 16:24 being random data and 24: being the
	// encrypted data
	r, _ := rc4.NewCipher(K3)
	r.XORKeyStream(out[16:24], out[16:24])

	dst := out[24:]
	for _, d := range data {
		r.XORKeyStream(dst[:len(d)], d)
		dst = dst[len(d):]
	}

	return out
}

func (c *rc4hmac) Decrypt(salt []byte, algo, usage int, data []byte) ([]byte, error) {
	switch usage {
	case gssSequenceNumber:
		if algo != cryptGssRc4Hmac && algo != cryptGssNone {
			return nil, ErrProtocol
		}

		return c.Encrypt(salt, usage, data), nil

	case gssWrapSeal:
		// GSS sealing uses an external checksum for integrity and
		// since RC4 is symettric we can just reencrypt the data
		if algo != cryptGssRc4Hmac {
			return nil, ErrProtocol
		}

		return c.Encrypt(salt, usage, data), nil
	}

	if algo != cryptRc4Hmac || len(data) < 24 {
		return nil, ErrProtocol
	}

	// Hash the key and usage together to get the HMAC-MD5 key
	h1 := hmac.New(md5.New, c.key)
	binary.Write(h1, binary.LittleEndian, rc4HmacUsage(usage))
	K1 := h1.Sum(nil)

	// Calculate the RC4 key using the checksum
	h3 := hmac.New(md5.New, K1)
	h3.Write(data[:16])
	K3 := h3.Sum(nil)

	// Decrypt d.Data[16:] in place with 16:24 being random data and 24:
	// being the encrypted data
	r, _ := rc4.NewCipher(K3)
	r.XORKeyStream(data[16:], data[16:])

	// Recalculate the checksum using the decrypted data
	ch := hmac.New(md5.New, K1)
	ch.Write(data[16:])
	chk := ch.Sum(nil)

	// Check the input checksum
	if subtle.ConstantTimeCompare(chk, data[:16]) != 1 {
		return nil, ErrProtocol
	}

	return data[24:], nil
}

func fixparity(u uint64, expand bool) uint64 {
	for i := 7; i >= 0; i-- {
		// pull out this byte
		var b uint64
		if expand {
			b = (u >> uint(i*7)) & 0x7F
		} else {
			b = (u >> (uint(i*8) + 1)) & 0x7F
		}
		// compute parity
		p := b ^ (b >> 4)
		p &= 0x0F
		p = 0x9669 >> p
		// add in parity as lsb
		b = (b << 1) | (p & 1)
		// set that byte in output
		u &^= 0xFF << uint(i*8)
		u |= b << uint(i*8)
	}

	return u
}

func fixweak(u uint64) uint64 {
	switch u {
	case 0x0101010101010101, 0xFEFEFEFEFEFEFEFE,
		0xE0E0E0E0F1F1F1F1, 0x1F1F1F1F0E0E0E0E,
		0x011F011F010E010E, 0x1F011F010E010E01,
		0x01E001E001F101F1, 0xE001E001F101F101,
		0x01FE01FE01FE01FE, 0xFE01FE01FE01FE01,
		0x1FE01FE00EF10EF1, 0xE01FE01FF10EF10E,
		0x1FFE1FFE0EFE0EFE, 0xFE1FFE1FFE0EFE0E,
		0xE0FEE0FEF1FEF1FE, 0xFEE0FEE0FEF1FEF1:
		u ^= 0xF0
	}

	return u
}

func desStringKey(password, salt string) []byte {
	blk := make([]byte, (len(password)+len(salt)+7)&^7)
	copy(blk, password)
	copy(blk[len(password):], salt)

	var u uint64
	for i := 0; i < len(blk); i += 8 {
		a := binary.BigEndian.Uint64(blk[i:])

		a = (a & 0x7F) |
			((a & 0x7F00) >> 1) |
			((a & 0x7F0000) >> 2) |
			((a & 0x7F000000) >> 3) |
			((a & 0x7F00000000) >> 4) |
			((a & 0x7F0000000000) >> 5) |
			((a & 0x7F000000000000) >> 6) |
			((a & 0x7F00000000000000) >> 7)

		if (i & 8) != 0 {
			a = ((a >> 1) & 0x5555555555555555) | ((a & 0x5555555555555555) << 1)
			a = ((a >> 2) & 0x3333333333333333) | ((a & 0x3333333333333333) << 2)
			a = ((a >> 4) & 0x0F0F0F0F0F0F0F0F) | ((a & 0x0F0F0F0F0F0F0F0F) << 4)
			a = ((a >> 8) & 0x00FF00FF00FF00FF) | ((a & 0x00FF00FF00FF00FF) << 8)
			a = ((a >> 16) & 0x0000FFFF0000FFFF) | ((a & 0x0000FFFF0000FFFF) << 16)
			a = (a >> 32) | (a << 32)
			a >>= 8
		}

		u ^= a
	}

	u = fixweak(fixparity(u, true))
	k := make([]byte, 8)
	binary.BigEndian.PutUint64(k, u)

	b, _ := des.NewCipher(k)
	c := cipher.NewCBCEncrypter(b, k)
	c.CryptBlocks(blk, blk)

	u = binary.BigEndian.Uint64(blk[len(blk)-8:])
	u = fixweak(fixparity(u, false))
	binary.BigEndian.PutUint64(k, u)

	return k
}

type descbc struct {
	key   []byte
	etype int
}

func (s *descbc) Sign(algo, usage int, data ...[]byte) ([]byte, error) {
	var h hash.Hash

	switch algo {
	case signGssDes:
		sz := 0
		for _, d := range data {
			sz += len(d)
		}
		sz = (sz + 7) &^ 7
		u := make([]byte, sz)
		v := u[:0]
		for _, d := range data {
			v = append(v, d...)
		}

		iv := [8]byte{}
		b, _ := des.NewCipher(s.key)
		c := cipher.NewCBCEncrypter(b, iv[:])
		c.CryptBlocks(u, u)
		return u[len(u)-8:], nil

	case signGssMd5Des:
		h = md5.New()
		for _, d := range data {
			h.Write(d)
		}
		return s.Sign(signGssDes, usage, h.Sum(nil))

	case signMd5Des:
		h = md5.New()
	case signMd4Des:
		h = md4.New()
	default:
		return unkeyedSign(algo, usage, data...)
	}

	var key [8]byte
	for i := 0; i < 8; i++ {
		key[i] = s.key[i] ^ 0xF0
	}

	chk := make([]byte, 24)
	io.ReadFull(rand.Reader, chk[:8])

	h.Write(chk[:8])
	for _, d := range data {
		h.Write(d)
	}
	h.Sum(chk[8:])

	iv := [8]byte{}
	b, _ := des.NewCipher(s.key)
	c := cipher.NewCBCEncrypter(b, iv[:])
	c.CryptBlocks(chk, chk)
	return chk, nil
}

func (s *descbc) SignAlgo(usage int) int {
	switch usage {
	case gssWrapSign:
		return signGssMd5Des
	}

	return signMd5Des
}

func (s *descbc) Encrypt(salt []byte, usage int, data ...[]byte) []byte {
	var h hash.Hash

	switch s.etype {
	case cryptDesCbcMd5:
		h = md5.New()
	case cryptDesCbcMd4:
		h = md4.New()
	default:
		panic("")
	}

	outsz := 8 + h.Size()
	for _, d := range data {
		outsz += len(d)
	}
	outsz = (outsz + 7) &^ 7
	out := make([]byte, outsz)

	io.ReadFull(rand.Reader, out[:8])

	v := out[8+h.Size():]
	for _, d := range data {
		n := copy(v, d)
		v = v[n:]
	}

	h.Write(out)
	h.Sum(out[:8])

	iv := [8]byte{}
	b, _ := des.NewCipher(s.key)
	c := cipher.NewCBCEncrypter(b, iv[:])
	c.CryptBlocks(out, out)

	return out
}

func (s *descbc) Decrypt(salt []byte, algo, usage int, data []byte) ([]byte, error) {
	var h hash.Hash

	switch algo {
	case cryptDesCbcMd5:
		h = md5.New()
	case cryptDesCbcMd4:
		h = md4.New()
	default:
		return nil, ErrProtocol
	}

	if (len(data) & 7) != 0 {
		return nil, ErrProtocol
	}

	iv := [8]byte{}
	b, _ := des.NewCipher(s.key)
	c := cipher.NewCBCDecrypter(b, iv[:])
	c.CryptBlocks(data, data)

	chk := make([]byte, h.Size())
	h.Write(data[:8])
	h.Write(chk) // Just need h.Size() zero bytes instead of the checksum
	h.Write(data[8+len(chk):])
	h.Sum(chk[:0])

	if subtle.ConstantTimeCompare(chk, data[8:8+len(chk)]) != 1 {
		return nil, ErrProtocol
	}

	return data[8+len(chk):], nil
}

func (s *descbc) EncryptAlgo(usage int) int {
	switch usage {
	case gssWrapSeal, gssSequenceNumber:
		return cryptGssDes
	}

	return s.etype
}

func (s *descbc) Key() []byte {
	return s.key
}

func generateKey(algo int, rand io.Reader) (key, error) {
	switch algo {
	case cryptRc4Hmac:
		data := [16]byte{}
		if _, err := io.ReadFull(rand, data[:]); err != nil {
			return nil, err
		}

		return loadKey(cryptRc4Hmac, data[:])

	case cryptDesCbcMd4, cryptDesCbcMd5:
		k := make([]byte, 8)
		if _, err := io.ReadFull(rand, k[1:]); err != nil {
			return nil, err
		}
		u := binary.BigEndian.Uint64(k)
		u = fixweak(fixparity(u, true))
		binary.BigEndian.PutUint64(k, u)
		return loadKey(algo, k)
	}

	return nil, ErrProtocol
}

func loadKey(algo int, key []byte) (key, error) {
	switch algo {
	case cryptRc4Hmac:
		return &rc4hmac{key}, nil
	case cryptDesCbcMd4, cryptDesCbcMd5:
		return &descbc{key, algo}, nil
	}
	return nil, ErrProtocol
}

func loadStringKey(algo int, pass, salt string) (key, error) {
	if len(pass) == 0 {
		return nil, ErrProtocol
	}

	switch algo {
	case cryptRc4Hmac:
		if len(salt) > 0 {
			return nil, ErrProtocol
		}
		return &rc4hmac{rc4HmacKey(pass)}, nil

	case cryptDesCbcMd4, cryptDesCbcMd5:
		return &descbc{desStringKey(pass, salt), algo}, nil
	}

	return nil, ErrProtocol
}

func mustGenerateKey(algo int, rand io.Reader) key {
	k, err := generateKey(algo, rand)
	if err != nil {
		panic(err)
	}
	return k
}

func mustLoadKey(algo int, key []byte) key {
	k, err := loadKey(algo, key)
	if err != nil {
		panic(err)
	}
	return k
}

func mustLoadStringKey(algo int, pass, salt string) key {
	k, err := loadStringKey(algo, pass, salt)
	if err != nil {
		panic(err)
	}
	return k
}
