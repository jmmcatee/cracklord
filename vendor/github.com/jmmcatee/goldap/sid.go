package ldap

import (
	"encoding/binary"
	"strconv"
	"strings"
)

type SID []byte

const sidRevision = 1

func ParseSID(s string) (SID, error) {
	if !strings.HasPrefix(s, "S-1-") {
		return nil, ErrInvalidSID
	}

	parts := strings.Split(s[len("S-1-"):], "-")
	sid := make([]byte, 4+4*len(parts))

	if len(parts) < 1 {
		return nil, ErrInvalidSID
	}

	v, err := strconv.ParseUint(parts[0], 10, 48)
	if err != nil {
		return nil, ErrInvalidSID
	}

	binary.BigEndian.PutUint64(sid, v)
	sid[0] = sidRevision
	sid[1] = byte(len(parts) - 1)

	for i, p := range parts[1:] {
		v, err := strconv.ParseUint(p, 10, 32)
		if err != nil {
			return nil, ErrInvalidSID
		}
		binary.LittleEndian.PutUint32(sid[8+i*4:], uint32(v))
	}

	return sid, nil
}

func (s SID) String() string {
	if len(s) < 8 || s[0] != sidRevision || len(s) != (int(s[1])*4)+8 {
		return ""
	}

	ret := []byte("S-1-")
	ret = strconv.AppendUint(ret, binary.BigEndian.Uint64(s[:8])&0xFFFFFFFFFFFF, 10)

	for i := 0; i < int(s[1]); i++ {
		ret = append(ret, "-"...)
		ret = strconv.AppendUint(ret, uint64(binary.LittleEndian.Uint32(s[8+i*4:])), 10)
	}

	return string(ret)
}

func (s SID) Domain() (SID, error) {
	if len(s) < 24 ||
		s[0] != sidRevision ||
		binary.BigEndian.Uint64(s)&0xFFFFFFFFFFFF != 5 ||
		binary.LittleEndian.Uint32(s[8:]) != 21 {
		return nil, ErrInvalidSID
	}

	if len(s) == 24 && s[1] == 4 {
		return s, nil
	}

	if len(s) != 28 || s[1] != 5 {
		return nil, ErrInvalidSID
	}

	ret := make([]byte, 24)
	copy(ret, s)
	ret[1] = 4 // change the length
	return ret, nil
}

func (s SID) Equal(r SID) bool {
	if len(s) != len(r) {
		return false
	}

	for i, a := range s {
		if a != r[i] {
			return false
		}
	}

	return true
}
