package ldap

import (
	"github.com/jmckaskill/asn1"
	"reflect"
	"strconv"
	"unicode"
	"unsafe"
)

type ObjectDN string

func (s ObjectDN) String() string {return string(s)}

var dnType = reflect.TypeOf(ObjectDN(""))

type appendSlice struct {
	val  reflect.Value
	typ  reflect.Type
	done chan error
}

func (s *appendSlice) onReply(tag int, msg []byte) status {
	v := reflect.Indirect(reflect.New(s.typ))
	sts, err := searchResult(tag, msg, v)

	switch sts {
	case finished:
		s.done <- err
	case data:
		s.val.Set(reflect.Append(s.val, v))
	}

	return sts
}

type setStruct struct {
	val      reflect.Value
	done     chan error
	haveData bool
}

func (s *setStruct) onReply(tag int, msg []byte) status {
	sts, err := searchResult(tag, msg, s.val)

	switch sts {
	case finished:
		if err == nil && !s.haveData {
			err = ErrNotFound
		}
		s.done <- err
	case data:
		s.haveData = true
	}

	return sts
}

type callFunc struct {
	fn   reflect.Value
	typ  reflect.Type
	done chan error
}

func (s *callFunc) onReply(tag int, msg []byte) status {
	args := [1]reflect.Value{reflect.Indirect(reflect.New(s.typ))}
	sts, err := searchResult(tag, msg, args[0])

	switch sts {
	case finished:
		s.done <- err
	case data:
		rets := s.fn.Call(args[:])
		if len(rets) > 0 {
			if err, _ := rets[len(rets)-1].Interface().(error); err != nil {
				s.done <- err
				return abandon
			}
		}
	}

	return sts
}

type sendChan struct {
	ch   reflect.Value
	typ  reflect.Type
	done chan error
}

func (s *sendChan) onReply(tag int, msg []byte) status {
	v := reflect.Indirect(reflect.New(s.typ))
	sts, err := searchResult(tag, msg, v)

	switch sts {
	case finished:
		s.done <- err
	case data:
		s.ch.Send(v)
	}

	return sts
}

func searchResult(tag int, msg []byte, val reflect.Value) (status, error) {
	switch tag {
	case searchEntryTag:
		if err := onSearchEntry(msg, val); err != nil {
			return abandon, err
		}
		return data, nil

	case extendedResultTag:
		err := onSearchDone(msg, extendedResultParam)
		return finished, err

	case searchDoneTag:
		err := onSearchDone(msg, searchDoneParam)
		return finished, err
	}

	return ignored, nil
}

func onSearchDone(data []byte, param string) error {
	res := result{}
	if _, err := asn1.UnmarshalWithParams(data, &res, param); err != nil {
		return err
	}

	if LdapResultCode(res.Code) != SuccessError {
		return ErrLdap{&res}
	}

	return nil
}

func descEquals(desc []byte, name string) bool {
	if len(desc) < len(name) {
		return false
	}

	for i := 0; i < len(name); i++ {
		if unicode.ToUpper(rune(desc[i])) != unicode.ToUpper(rune(name[i])) {
			return false
		}
	}

	return len(desc) == len(name) || desc[len(name)] == ';'
}

func getattr(e *searchEntry, name string) [][]byte {
	for _, a := range e.Attrs {
		if descEquals(a.Desc, name) {
			return a.Vals
		}
	}

	return nil
}

func onSearchEntry(data []byte, val reflect.Value) error {
	e := searchEntry{}
	if _, err := asn1.UnmarshalWithParams(data, &e, searchEntryParam); err != nil {
		panic(err)
		return err
	}

	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		ft := typ.Field(i)
		fv := val.Field(i)

		if ft.PkgPath != "" {
			// private field
			continue
		}

		if ft.Type == dnType {
			fv.SetString(string(e.DN))
			continue
		}

		vals := getattr(&e, ft.Name)
		if len(vals) == 0 {
			continue
		}

		switch ft.Type.Kind() {
		case reflect.Slice, reflect.Array:
			t := ft.Type.Elem()
			switch t.Kind() {
			case reflect.Slice, reflect.Array:
				if t.Elem().Kind() == reflect.Uint8 {
					// [][]byte
					fv.Set(reflect.ValueOf(vals))
				}

			case reflect.Uint8:
				// []byte
				fv.Set(reflect.ValueOf(vals[0]))

			case reflect.String:
				// []string
				strs := reflect.MakeSlice(ft.Type, len(vals), len(vals))
				for i, v := range vals {
					// We need to convert from a string to
					// the users own string type. There
					// doesn't seem to be a way to do this
					// using reflect, so we fall back and
					// use unsafe. This is safe because
					// we've already checked the kind.
					str := string(v)
					str2 := reflect.NewAt(t, unsafe.Pointer(&str))
					strs.Index(i).Set(str2.Elem())
				}
				fv.Set(strs)
			}

		case reflect.String:
			fv.SetString(string(vals[0]))

		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			v, err := strconv.ParseInt(string(vals[0]), 10, ft.Type.Bits())
			if err != nil {
				return err
			}
			fv.SetInt(v)

		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			v, err := strconv.ParseUint(string(vals[0]), 10, ft.Type.Bits())
			if err != nil {
				return err
			}
			fv.SetUint(v)

		case reflect.Float32, reflect.Float64:
			v, err := strconv.ParseFloat(string(vals[0]), ft.Type.Bits())
			if err != nil {
				return err
			}
			fv.SetFloat(v)
		}
	}

	return nil
}

func (db *DB) search(out interface{}, base ObjectDN, filter Filter, scope asn1.Enumerated) error {
	req := searchRequest{
		BaseObject: []byte(base),
		Scope:      scope,
	}

	var err error
	if filter != nil {
		req.Filter.FullBytes, err = filter.marshal()
	} else {
		req.Filter.FullBytes, err = Present("ObjectClass").marshal()
	}

	if err != nil {
		return err
	}

	var typ reflect.Type
	var reply replyHandler
	done := make(chan error)

	ov := reflect.ValueOf(out)
	switch ov.Kind() {
	case reflect.Ptr:
		ov = reflect.Indirect(ov)

		switch ov.Kind() {
		case reflect.Slice:
			typ = ov.Type().Elem()
			reply = &appendSlice{ov, typ, done}
		case reflect.Struct:
			typ = ov.Type()
			reply = &setStruct{ov, done, false}
			req.SizeLimit = 1
		}

	case reflect.Func:
		ft := ov.Type()
		if ft.NumIn() != 1 {
			panic("expected function with 1 argument")
		}

		typ = ft.In(0)
		reply = &callFunc{ov, typ, done}

	case reflect.Chan:
		typ = ov.Type().Elem()
		reply = &sendChan{ov, typ, done}

	default:
		return ErrUnsupportedType{ov.Type()}
	}

	if typ.Kind() != reflect.Struct {
		return ErrUnsupportedType{typ}
	}

	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)

		if f.PkgPath != "" {
			// private field
			continue
		}

		if f.Type == dnType || len(f.Name) == 0 {
			continue
		}

		if f.Tag.Get("ldap") == "-" {
			continue
		}

		req.Attrs = append(req.Attrs, []byte(f.Name))
	}

	if len(req.Attrs) == 0 {
		req.Attrs = [][]byte{noAttributes}
	}

	if err := db.send(searchRequestParam, req, reply); err != nil {
		return err
	}

	return <-done
}

func (db *DB) SearchTree(out interface{}, base ObjectDN, filter Filter) error {
	return db.search(out, base, filter, wholeSubtreeScope)
}

func (db *DB) GetObject(out interface{}, dn ObjectDN) error {
	return db.search(out, dn, nil, baseObjectScope)
}
