package ldap

import (
	"errors"
	"fmt"
	"github.com/jmckaskill/asn1"
	"reflect"
)

type message struct {
	Id   uint32
	Data asn1.RawValue
}

type result struct {
	Code            asn1.Enumerated
	DN              []byte
	Msg             []byte
	Referrals       [][]byte `asn1:"optional,tag:3"`
	SaslCredentials []byte   `asn1:"optional,tag:7"`
	ExtendedName    []byte   `asn1:"optional,tag:10"`
	ExtendedValue   []byte   `asn1:"optional,tag:11"`
}

type bindRequest struct {
	Version int
	BindDN  []byte
	Auth    asn1.RawValue
}

type saslCredentials struct {
	Mechanism   []byte
	Credentials []byte `asn1:"optional"`
}

type searchRequest struct {
	BaseObject   []byte
	Scope        asn1.Enumerated
	DerefAliases asn1.Enumerated
	SizeLimit    int
	TimeLimit    int
	TypesOnly    bool
	Filter       asn1.RawValue
	Attrs        [][]byte
}

const (
	baseObjectScope asn1.Enumerated = iota
	singleLevelScope
	wholeSubtreeScope
)

const (
	NeverDeferAliases asn1.Enumerated = iota
	DerefInSearching
	DerefFindingBaseObj
	DerefAlways
)

type filterSet struct {
	Filters []asn1.RawValue `"asn1:set"`
}

type assertion struct {
	Attr  []byte
	Value []byte
}

type substringFilter struct {
	Desc       []byte
	Substrings []asn1.RawValue
}

type attribute struct {
	Desc []byte
	Vals [][]byte `asn1:"set"`
}

type searchEntry struct {
	DN    []byte
	Attrs []attribute
}

type searchReference [][]byte

type modifyRequest struct {
	DN  []byte
	Ops []change
}

type change struct {
	Op   asn1.Enumerated
	Attr []attribute
}

const (
	addOp asn1.Enumerated = iota
	deleteOp
	replaceOp
)

type addRequest struct {
	DN   []byte
	Attr []attribute
}

type deleteRequest []byte

type modifyDNRequest struct {
	OldDN       []byte
	NewDN       []byte // note this can be relative
	DeleteOld   bool
	NewSuperior []byte `asn1:"explicit,tag:0,optional"`
}

type compareRequest struct {
	DN  []byte
	Ava AttributeValueAssertion
}

type abandonRequest int

type extendedMessage struct {
	Name  []byte `asn1:"explicit,optional,tag:0"`
	Value []byte `asn1:"explicit,optional,tag:1"`
}

const (
	ldapVersion = 3

	bindRequestTag       = 0 // bindRequest
	bindRequestParam     = "application,tag:0"
	bindResultTag        = 1 // result
	bindResultParam      = "application,tag:1"
	unbindRequestTag     = 2 // nil
	searchRequestTag     = 3 // searchRequest
	searchRequestParam   = "application,tag:3"
	searchEntryTag       = 4 // searchEntry
	searchEntryParam     = "application,tag:4"
	searchReferenceTag   = 19 // searchReference
	searchDoneTag        = 5  // result
	searchDoneParam      = "application,tag:5"
	modifyRequestTag     = 6  // modifyRequest
	modifyResultTag      = 7  // result
	addRequestTag        = 8  // addRequest
	addResultTag         = 9  // result
	deleteRequestTag     = 10 // deleteRequest
	deleteResultTag      = 11 // result
	modifyDNRequestTag   = 12 // modifyDNRequest
	modifyDNResultTag    = 13 // result
	compareRequestTag    = 14 // compareRequest
	compareResultTag     = 15 // result
	abandonRequestTag    = 16 // abandonRequest
	abandonRequestParam  = "application,tag:16"
	extendedRequestTag   = 23 // extendedMessage
	extendedRequestParam = "application,tag:23"
	extendedResultTag    = 24 // result
	extendedResultParam  = "application,tag:24"
	intermediatTag       = 25 // extendedMessage

	simpleBindParam = "tag:0" // []byte
	saslBindParam   = "tag:3" // saslCredentials

	andParam             = "tag:0" // filterSet
	orParam              = "tag:1" // filterSet
	notParam             = "tag:2" // filter (asn1.RawValue)
	equalParam           = "tag:3" // AttributeValueAssertion
	substringParam       = "tag:4" // substringFilter
	greaterOrEqualParam  = "tag:5" // AttributeValueAssertion
	lessOrEqualParam     = "tag:6" // AttributeValueAssertion
	presentParam         = "tag:7" // []byte
	approxEqualParam     = "tag:8" // AttributeValueAssertion
	extensibleMatchParam = "tag:9" // matchingRuleAssertion

	substringInitialParam = "tag:0" // []byte
	substringAnyParam     = "tag:1"
	substringFinalParam   = "tag:2"

	classApplication     = 1
	universalSequenceTag = 0x30
	maxMessageSize       = 16 * 1024 * 1024
)

var (
	noAttributes = []byte("1.1")
	startTLS     = "1.3.6.1.4.1.1466.20037"
)

type LdapResultCode int

const (
	SuccessError LdapResultCode = iota
	OperationsError
	ProtocolError
	TimeLimitExceeded
	SizeLimitExceeded
	CompareFalse
	CompareTrue
	AuthMethodNotSupported
	StrongerAuthRequired
	_
	ReferralError
	AdminLimitExceeded
	UnavailableCriticalExtension
	ConfidentialityRequired
	SaslBindInProgress
	_
	NoSuchAttribute
	UndefinedAttributeType
	InappropriateMatching
	ConstraintViolation
	AttributeOrValueExists
	InvalidAttributeSyntax
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	NoSuchObject
	AliasProblem
	InvalidDNSyntax
	_
	AliasDereferncingProblem
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	InappropriateAuthentication
	InvalidCredentials
	InsufficientAccessRights
	Busy
	Unavailable
	UnwillingToPerform
	LoopDetect
	_
	_
	_
	_
	_
	_
	_
	_
	_
	NamingViolation
	ObjectClassViolation
	NotAllowedOnNonLeaf
	NotAllowedOnRDN
	EntryAlreadyExists
	ObjectClassModeProhibited
	_
	AffectsMultipleDSAs
	_
	_
	_
	_
	_
	_
	_
	_
	Other
)

func (s LdapResultCode) String() string {
	switch s {
	case SuccessError:
		return "success"
	case OperationsError:
		return "operations error"
	case ProtocolError:
		return "protocol error"
	case TimeLimitExceeded:
		return "time limit exceeded"
	case SizeLimitExceeded:
		return "size limit exceeded"
	case CompareFalse:
		return "compare false"
	case CompareTrue:
		return "compare true"
	case AuthMethodNotSupported:
		return "auth method not supported"
	case StrongerAuthRequired:
		return "stronger auth required"
	case ReferralError:
		return "referral"
	case AdminLimitExceeded:
		return "admin limit exceeded"
	case UnavailableCriticalExtension:
		return "unavailble critical extension"
	case ConfidentialityRequired:
		return "confidentiality required"
	case SaslBindInProgress:
		return "sasl bind in progress"
	case NoSuchAttribute:
		return "no such attribute"
	case UndefinedAttributeType:
		return "undefined attribute type"
	case InappropriateMatching:
		return "inappropriate matching"
	case ConstraintViolation:
		return "constraint violation"
	case AttributeOrValueExists:
		return "attribute or value exists"
	case InvalidAttributeSyntax:
		return "invalid attribute syntax"
	case NoSuchObject:
		return "no such Object"
	case AliasProblem:
		return "alias problem"
	case InvalidDNSyntax:
		return "invalid DN syntax"
	case AliasDereferncingProblem:
		return "alias dereferencing problem"
	case InappropriateAuthentication:
		return "inappropriate authentication"
	case InvalidCredentials:
		return "invalid credentials"
	case InsufficientAccessRights:
		return "insufficient access rights"
	case Busy:
		return "busy"
	case Unavailable:
		return "unavailable"
	case UnwillingToPerform:
		return "unwilling to perform"
	case LoopDetect:
		return "loop detect"
	case NamingViolation:
		return "naming violation"
	case ObjectClassViolation:
		return "object class violation"
	case NotAllowedOnNonLeaf:
		return "not allowed on non-leaf"
	case NotAllowedOnRDN:
		return "not allowed on RDN"
	case EntryAlreadyExists:
		return "entry already exists"
	case ObjectClassModeProhibited:
		return "object class mode prohibited"
	case AffectsMultipleDSAs:
		return "affects multiple DSAs"
	case Other:
		return "other"
	}

	return fmt.Sprintf("unknown error %d", int(s))
}

var (
	ErrAuthNotSupported = errors.New("ldap: auth not supported")
	ErrProtocol         = errors.New("ldap: protocol error")
	ErrInvalidSID       = errors.New("ldap: unable to parse SID")
	ErrIncompleteAuth   = errors.New("ldap: incomplete auth")
	ErrClosed           = errors.New("ldap: db closed")
	ErrNotFound         = errors.New("ldap: not found")
	ErrShortRead        = errors.New("ldap: short SASL read")
)

type ErrLdap struct {
	res *result
}

func (e ErrLdap) Error() string {
	return fmt.Sprintf("ldap: remote error (%s) %s", LdapResultCode(e.res.Code).String(), string(e.res.Msg))
}

type ErrUnsupportedType struct {
	typ reflect.Type
}

func (e ErrUnsupportedType) Error() string {
	return fmt.Sprintf("ldap: unsupported type %s", e.typ.String())
}

type Filter interface {
	marshal() ([]byte, error)
}

type AttributeValueAssertion struct {
	// attribute can be followed by zero or more options using ;
	// seperators
	Attr  string
	Value []byte
}

type And []Filter
type Or []Filter
type Not struct {
	Filter Filter
}
type Equal AttributeValueAssertion
type ApproxEqual AttributeValueAssertion
type GreaterOrEqual AttributeValueAssertion
type LessOrEqual AttributeValueAssertion
type Present string

func (a And) marshal() (data []byte, err error) {
	filters := make([]asn1.RawValue, len(a))
	for i, f := range a {
		if filters[i].FullBytes, err = f.marshal(); err != nil {
			return nil, err
		}
	}
	return asn1.MarshalWithParams(filterSet{filters}, andParam)
}

func (o Or) marshal() (data []byte, err error) {
	filters := make([]asn1.RawValue, len(o))
	for i, f := range o {
		if filters[i].FullBytes, err = f.marshal(); err != nil {
			return nil, err
		}
	}
	return asn1.MarshalWithParams(filterSet{filters}, orParam)
}

func (n Not) marshal() ([]byte, error) {
	f, err := n.Filter.marshal()
	if err != nil {
		return nil, err
	}
	return asn1.MarshalWithParams(asn1.RawValue{FullBytes: f}, notParam)
}

func (e Equal) marshal() ([]byte, error) {
	return asn1.MarshalWithParams(assertion{[]byte(e.Attr), e.Value}, equalParam)
}

func (e ApproxEqual) marshal() ([]byte, error) {
	return asn1.MarshalWithParams(assertion{[]byte(e.Attr), e.Value}, approxEqualParam)
}

func (e GreaterOrEqual) marshal() ([]byte, error) {
	return asn1.MarshalWithParams(assertion{[]byte(e.Attr), e.Value}, greaterOrEqualParam)
}

func (e LessOrEqual) marshal() ([]byte, error) {
	return asn1.MarshalWithParams(assertion{[]byte(e.Attr), e.Value}, lessOrEqualParam)
}

func (p Present) marshal() ([]byte, error) {
	return asn1.MarshalWithParams([]byte(p), presentParam)
}
