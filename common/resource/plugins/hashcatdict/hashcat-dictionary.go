package hashcatdict

import (
	"errors"
	log "github.com/Sirupsen/logrus"
	"github.com/jmmcatee/cracklord/common"
	"github.com/vaughan0/go-ini"
	"sort"
)

type hcConfig struct {
	BinPath      string
	WorkDir      string
	Arguments    string
	Dictionaries map[string]string
	DictOrder    []string
	Rules        map[string]string
	RulesOrder   []string
	HashTypes    map[string]string
	HashOrder    []string
}

var config = hcConfig{
	BinPath:      "",
	WorkDir:      "",
	Arguments:    "",
	Dictionaries: map[string]string{},
	DictOrder:    []string{},
	Rules:        map[string]string{},
	RulesOrder:   []string{},
	HashTypes:    map[string]string{},
	HashOrder:    []string{},
}

var supportedHash = map[string]string{
	"MD5":                       "0",
	"md5($pass.$salt)":          "10",
	"md5($salt.$pass)":          "20",
	"md5(unicode($pass).$salt)": "30",
	"md5($salt.unicode($pass))": "40",
	"HMAC-MD5 (key = $pass)":    "50",
	"HMAC-MD5 (key = $salt)":    "60",
	"SHA1":                                             "100",
	"sha1($pass.$salt)":                                "110",
	"sha1($salt.$pass)":                                "120",
	"sha1(unicode($pass).$salt)":                       "130",
	"sha1($salt.unicode($pass))":                       "140",
	"HMAC-SHA1 (key = $pass)":                          "150",
	"HMAC-SHA1 (key = $salt)":                          "160",
	"sha1(LinkedIn)":                                   "190",
	"MySQL323":                                         "200",
	"MySQL4.1/MySQL5":                                  "300",
	"phpass, MD5(Wordpress), MD5(phpBB3), MD5(Joomla)": "400",
	"md5crypt, MD5(Unix), FreeBSD MD5, Cisco-IOS MD5":  "500",
	"Juniper IVE":                                      "501",
	"MD4":                                              "900",
	"NTLM":                                             "1000",
	"Domain Cached Credentials, mscash": "1100",
	"SHA256":                               "1400",
	"sha256($pass.$salt)":                  "1410",
	"sha256($salt.$pass)":                  "1420",
	"sha256(unicode($pass).$salt)":         "1430",
	"sha256($salt.unicode($pass))":         "1440",
	"HMAC-SHA256 (key = $pass)":            "1450",
	"HMAC-SHA256 (key = $salt)":            "1460",
	"descrypt, DES(Unix), Traditional DES": "1500",
	"md5apr1, MD5(APR), Apache MD5":        "1600",
	"SHA512":                               "1700",
	"sha512($pass.$salt)":                  "1710",
	"sha512($salt.$pass)":                  "1720",
	"sha512(unicode($pass).$salt)":         "1730",
	"sha512($salt.unicode($pass))":         "1740",
	"HMAC-SHA512 (key = $pass)":            "1750",
	"HMAC-SHA512 (key = $salt)":            "1760",
	"sha512crypt, SHA512(Unix)":            "1800",
	"Domain Cached Credentials2, mscash2":  "2100",
	"Cisco-PIX MD5":                        "2400",
	"Cisco-ASA MD5":                        "2410",
	"WPA/WPA2":                             "2500",
	"Double MD5":                           "2600",
	"LM":                                   "3000",
	"Oracle 7-10g, DES(Oracle)":            "3100",
	"bcrypt, Blowfish(OpenBSD)":            "3200",
	"md5($salt.md5($pass))":                "3710",
	"md5($pass.$salt.$pass)":               "3810",
	"md5(strtoupper(md5($pass)))":          "4300",
	"md5(sha1($pass))":                     "4400",
	"Double SHA1":                          "4500",
	"sha1(md5($pass))":                     "4700",
	"sha1($salt.$pass.$salt)":              "4710",
	"MD5(Chap), iSCSI CHAP authentication": "4800",
	"SHA-3(Keccak)":                        "5000",
	"Half MD5":                             "5100",
	"Password Safe v3":                     "5200",
	"IKE-PSK MD5":                          "5300",
	"IKE-PSK SHA1":                         "5400",
	"NetNTLMv1-VANILLA / NetNTLMv1+ESS":    "5500",
	"NetNTLMv2":                            "5600",
	"Cisco-IOS SHA256":                     "5700",
	"Android PIN":                          "5800",
	"RipeMD160":                            "6000",
	"Whirlpool":                            "6100",
	"AIX {smd5}":                           "6300",
	"AIX {ssha256}":                        "6400",
	"AIX {ssha512}":                        "6500",
	"1Password, agilekeychain":             "6600",
	"AIX {ssha1}":                          "6700",
	"Lastpass":                             "6800",
	"GOST R 34.11-94":                      "6900",
	"OSX v10.8 / v10.9":                    "7100",
	"GRUB 2":                               "7200",
	"IPMI2 RAKP HMAC-SHA1":                 "7300",
	"sha256crypt, SHA256(Unix)":            "7400",
	"Kerberos 5 AS-REQ Pre-Auth etype 23":  "7500",
	"SAP CODVN B (BCODE)":                  "7700",
	"SAP CODVN F/G (PASSCODE)":             "7800",
	"Drupal7":                              "7900",
	"Sybase ASE":                           "8000",
	"Citrix Netscaler":                     "8100",
	"1Password, cloudkeychain":             "8200",
	"DNSSEC (NSEC3)":                       "8300",
	"WBB3, Woltlab Burning Board 3":        "8400",
	"RACF":                 "8500",
	"Lotus Notes/Domino 5": "8600",
	"Lotus Notes/Domino 6": "8700",
	"Android FDE <= 4.3":   "8800",
	"scrypt":               "8900",
	"Password Safe v2":     "9000",
	"Lotus Notes/Domino 8": "9100",
	"Cisco $8$":            "9200",
	"Cisco $9$":            "9300",
	"Office 2007":          "9400",
	"Office 2010":          "9500",
	"Office 2013":          "9600",
	"MS Office <= 2003 MD5 + RC4, oldoffice$0, oldoffice$1":  "9700",
	"MS Office <= 2003 MD5 + RC4, collider-mode #1":          "9710",
	"MS Office <= 2003 MD5 + RC4, collider-mode #2":          "9720",
	"MS Office <= 2003 SHA1 + RC4, oldoffice$3, oldoffice$4": "9800",
	"MS Office <= 2003 SHA1 + RC4, collider-mode #1":         "9810",
	"MS Office <= 2003 SHA1 + RC4, collider-mode #2":         "9820",
	"Radmin2":                                          "9900",
	"Django (PBKDF2-SHA256)":                           "10000",
	"SipHash":                                          "10100",
	"Cram MD5":                                         "10200",
	"SAP CODVN H (PWDSALTEDHASH) iSSHA-1":              "10300",
	"PDF 1.1 - 1.3 (Acrobat 2 - 4)":                    "10400",
	"PDF 1.1 - 1.3 (Acrobat 2 - 4) + collider-mode #1": "10410",
	"PDF 1.1 - 1.3 (Acrobat 2 - 4) + collider-mode #2": "10420",
	"PDF 1.4 - 1.6 (Acrobat 5 - 8)":                    "10500",
	"PDF 1.7 Level 3 (Acrobat 9)":                      "10600",
	"PDF 1.7 Level 8 (Acrobat 10 - 11)":                "10700",
	"SHA384":                           "10800",
	"PBKDF2-HMAC-SHA256":               "10900",
	"Joomla < 2.5.18":                  "11",
	"PostgreSQL":                       "12",
	"osCommerce, xt:Commerce":          "21",
	"Juniper Netscreen/SSG (ScreenOS)": "22",
	"Skype": "23",
	"nsldap, SHA-1(Base64), Netscape LDAP SHA":    "101",
	"nsldaps, SSHA-1(Base64), Netscape LDAP SSHA": "111",
	"Oracle 11g/12c":                              "112",
	"SMF > v1.1":                                  "121",
	"OSX v10.4, v10.5, v10.6":                     "122",
	"Django (SHA-1)":                              "124",
	"MSSQL(2000)":                                 "131",
	"MSSQL(2005)":                                 "132",
	"PeopleSoft":                                  "133",
	"EPiServer 6.x < v4":                          "141",
	"hMailServer":                                 "1421",
	"EPiServer 6.x > v4":                          "1441",
	"SSHA-512(Base64), LDAP {SSHA512}":            "1711",
	"OSX v10.7":                                   "1722",
	"MSSQL(2012), MSSQL(2014)":                    "1731",
	"vBulletin < v3.8.5":                          "2611",
	"PHPS":                                        "2612",
	"vBulletin > v3.8.5":                          "2711",
	"IPB2+, MyBB1.2+":                             "2811",
	"Mediawiki B type":                            "3711",
	"Redmine Project Management Web App":          "7600",
}

var supportedHashInt = map[string]int{
	"MD5":                       0,
	"md5($pass.$salt)":          10,
	"md5($salt.$pass)":          20,
	"md5(unicode($pass).$salt)": 30,
	"md5($salt.unicode($pass))": 40,
	"HMAC-MD5 (key = $pass)":    50,
	"HMAC-MD5 (key = $salt)":    60,
	"SHA1":                                             100,
	"sha1($pass.$salt)":                                110,
	"sha1($salt.$pass)":                                120,
	"sha1(unicode($pass).$salt)":                       130,
	"sha1($salt.unicode($pass))":                       140,
	"HMAC-SHA1 (key = $pass)":                          150,
	"HMAC-SHA1 (key = $salt)":                          160,
	"sha1(LinkedIn)":                                   190,
	"MySQL323":                                         200,
	"MySQL4.1/MySQL5":                                  300,
	"phpass, MD5(Wordpress), MD5(phpBB3), MD5(Joomla)": 400,
	"md5crypt, MD5(Unix), FreeBSD MD5, Cisco-IOS MD5":  500,
	"Juniper IVE":                                      501,
	"MD4":                                              900,
	"NTLM":                                             1000,
	"Domain Cached Credentials, mscash": 1100,
	"SHA256":                               1400,
	"sha256($pass.$salt)":                  1410,
	"sha256($salt.$pass)":                  1420,
	"sha256(unicode($pass).$salt)":         1430,
	"sha256($salt.unicode($pass))":         1440,
	"HMAC-SHA256 (key = $pass)":            1450,
	"HMAC-SHA256 (key = $salt)":            1460,
	"descrypt, DES(Unix), Traditional DES": 1500,
	"md5apr1, MD5(APR), Apache MD5":        1600,
	"SHA512":                               1700,
	"sha512($pass.$salt)":                  1710,
	"sha512($salt.$pass)":                  1720,
	"sha512(unicode($pass).$salt)":         1730,
	"sha512($salt.unicode($pass))":         1740,
	"HMAC-SHA512 (key = $pass)":            1750,
	"HMAC-SHA512 (key = $salt)":            1760,
	"sha512crypt, SHA512(Unix)":            1800,
	"Domain Cached Credentials2, mscash2":  2100,
	"Cisco-PIX MD5":                        2400,
	"Cisco-ASA MD5":                        2410,
	"WPA/WPA2":                             2500,
	"Double MD5":                           2600,
	"LM":                                   3000,
	"Oracle 7-10g, DES(Oracle)":            3100,
	"bcrypt, Blowfish(OpenBSD)":            3200,
	"md5($salt.md5($pass))":                3710,
	"md5($pass.$salt.$pass)":               3810,
	"md5(strtoupper(md5($pass)))":          4300,
	"md5(sha1($pass))":                     4400,
	"Double SHA1":                          4500,
	"sha1(md5($pass))":                     4700,
	"sha1($salt.$pass.$salt)":              4710,
	"MD5(Chap), iSCSI CHAP authentication": 4800,
	"SHA-3(Keccak)":                        5000,
	"Half MD5":                             5100,
	"Password Safe v3":                     5200,
	"IKE-PSK MD5":                          5300,
	"IKE-PSK SHA1":                         5400,
	"NetNTLMv1-VANILLA / NetNTLMv1+ESS":    5500,
	"NetNTLMv2":                            5600,
	"Cisco-IOS SHA256":                     5700,
	"Android PIN":                          5800,
	"RipeMD160":                            6000,
	"Whirlpool":                            6100,
	"AIX {smd5}":                           6300,
	"AIX {ssha256}":                        6400,
	"AIX {ssha512}":                        6500,
	"1Password, agilekeychain":             6600,
	"AIX {ssha1}":                          6700,
	"Lastpass":                             6800,
	"GOST R 34.11-94":                      6900,
	"OSX v10.8 / v10.9":                    7100,
	"GRUB 2":                               7200,
	"IPMI2 RAKP HMAC-SHA1":                 7300,
	"sha256crypt, SHA256(Unix)":            7400,
	"Kerberos 5 AS-REQ Pre-Auth etype 23":  7500,
	"SAP CODVN B (BCODE)":                  7700,
	"SAP CODVN F/G (PASSCODE)":             7800,
	"Drupal7":                              7900,
	"Sybase ASE":                           8000,
	"Citrix Netscaler":                     8100,
	"1Password, cloudkeychain":             8200,
	"DNSSEC (NSEC3)":                       8300,
	"WBB3, Woltlab Burning Board 3":        8400,
	"RACF":                 8500,
	"Lotus Notes/Domino 5": 8600,
	"Lotus Notes/Domino 6": 8700,
	"Android FDE <= 4.3":   8800,
	"scrypt":               8900,
	"Password Safe v2":     9000,
	"Lotus Notes/Domino 8": 9100,
	"Cisco $8$":            9200,
	"Cisco $9$":            9300,
	"Office 2007":          9400,
	"Office 2010":          9500,
	"Office 2013":          9600,
	"MS Office <= 2003 MD5 + RC4, oldoffice$0, oldoffice$1":  9700,
	"MS Office <= 2003 MD5 + RC4, collider-mode #1":          9710,
	"MS Office <= 2003 MD5 + RC4, collider-mode #2":          9720,
	"MS Office <= 2003 SHA1 + RC4, oldoffice$3, oldoffice$4": 9800,
	"MS Office <= 2003 SHA1 + RC4, collider-mode #1":         9810,
	"MS Office <= 2003 SHA1 + RC4, collider-mode #2":         9820,
	"Radmin2":                                          9900,
	"Django (PBKDF2-SHA256)":                           10000,
	"SipHash":                                          10100,
	"Cram MD5":                                         10200,
	"SAP CODVN H (PWDSALTEDHASH) iSSHA-1":              10300,
	"PDF 1.1 - 1.3 (Acrobat 2 - 4)":                    10400,
	"PDF 1.1 - 1.3 (Acrobat 2 - 4) + collider-mode #1": 10410,
	"PDF 1.1 - 1.3 (Acrobat 2 - 4) + collider-mode #2": 10420,
	"PDF 1.4 - 1.6 (Acrobat 5 - 8)":                    10500,
	"PDF 1.7 Level 3 (Acrobat 9)":                      10600,
	"PDF 1.7 Level 8 (Acrobat 10 - 11)":                10700,
	"SHA384":                           10800,
	"PBKDF2-HMAC-SHA256":               10900,
	"Joomla < 2.5.18":                  11,
	"PostgreSQL":                       12,
	"osCommerce, xt:Commerce":          21,
	"Juniper Netscreen/SSG (ScreenOS)": 22,
	"Skype": 23,
	"nsldap, SHA-1(Base64), Netscape LDAP SHA":    101,
	"nsldaps, SSHA-1(Base64), Netscape LDAP SSHA": 111,
	"Oracle 11g/12c":                              112,
	"SMF > v1.1":                                  121,
	"OSX v10.4, v10.5, v10.6":                     122,
	"Django (SHA-1)":                              124,
	"MSSQL(2000)":                                 131,
	"MSSQL(2005)":                                 132,
	"PeopleSoft":                                  133,
	"EPiServer 6.x < v4":                          141,
	"hMailServer":                                 1421,
	"EPiServer 6.x > v4":                          1441,
	"SSHA-512(Base64), LDAP {SSHA512}":            1711,
	"OSX v10.7":                                   1722,
	"MSSQL(2012), MSSQL(2014)":                    1731,
	"vBulletin < v3.8.5":                          2611,
	"PHPS":                                        2612,
	"vBulletin > v3.8.5":                          2711,
	"IPB2+, MyBB1.2+":                             2811,
	"Mediawiki B type":                            3711,
	"Redmine Project Management Web App":          7600,
}

/*
	Read the hascatdict init file to setup hashcatdict
*/
func Setup(path string) error {
	log.Debug("Setting up hashcatdict tool")
	// Join the path provided
	confFile, err := ini.LoadFile(path)
	if err != nil {
		log.WithField("file", path).Error("Unable to load configuration file.")
		return err
	}

	// Get the bin path
	basic := confFile.Section("Basic")
	if len(basic) == 0 {
		// Nothing retrieved, so return error
		return errors.New("No \"Basic\" configuration section.")
	}
	config.BinPath = basic["binPath"]
	config.WorkDir = basic["workingdir"]
	config.Arguments = basic["arguments"]

	log.WithFields(log.Fields{
		"binpath":   config.BinPath,
		"WorkDir":   config.WorkDir,
		"Arguments": config.Arguments,
	}).Debug("Basic configuration complete")

	// Get the dictionary section
	dicts := confFile.Section("Dictionaries")
	if len(dicts) == 0 {
		// Nothing retrieved, so return error
		log.Debug("No 'dictionaries' configuration section.")
		return errors.New("No \"Dictionaries\" configuration section.")
	}
	for key, value := range dicts {
		log.WithFields(log.Fields{
			"name": key,
			"path": value,
		}).Debug("Added dictionary")
		config.Dictionaries[key] = value
	}

	// Get the rule section
	rules := confFile.Section("Rules")
	if len(dicts) == 0 {
		// Nothing retrieved, so return error
		log.Debug("No 'rules' configuration section.")
		return errors.New("No \"Rules\" configuration section.")
	}
	for key, value := range rules {
		log.WithFields(log.Fields{
			"name": key,
			"path": value,
		}).Debug("Added rule")
		config.Rules[key] = value
	}

	// Setup the hashes
	config.HashTypes = supportedHash

	// Setup sorted order for consistency
	for key, _ := range config.Dictionaries {
		config.DictOrder = append(config.DictOrder, key)
	}
	sort.Strings(config.DictOrder)

	for key, _ := range config.Rules {
		config.RulesOrder = append(config.RulesOrder, key)
	}
	sort.Strings(config.RulesOrder)

	for key, _ := range config.HashTypes {
		config.HashOrder = append(config.HashOrder, key)
	}
	sort.Strings(config.HashOrder)

	log.Info("Hashcatdict tool successfully setup")

	return nil
}

type hashcatDictTooler struct {
	toolUUID string
}

func (h *hashcatDictTooler) Name() string {
	return "Hashcat Dictionary Attack"
}

func (h *hashcatDictTooler) Type() string {
	return "Dictionary"
}

func (h *hashcatDictTooler) Version() string {
	return "1.33"
}

func (h *hashcatDictTooler) UUID() string {
	return h.toolUUID
}

func (h *hashcatDictTooler) SetUUID(s string) {
	h.toolUUID = s
}

func (h *hashcatDictTooler) Parameters() string {
	params := `{
		"form": [
		  "algorithm",
		  "dictionaries",
		  "rules",
		  {
		    "key": "hashes",
		    "type": "textarea",
		    "placeholder": "Add in Hashcat required format"
		  }
		],
		"schema": {
		"type": "object",
		  "properties": {
		    "name": {
		      "title": "Name",
		      "type": "string"
		    },
		    "algorithm": {
		      "title": "Select hash type to attack",
		      "type": "string",
		      "enum": [ `
	var first = true
	for _, key := range config.HashOrder {
		if !first {
			params += `,`
		}

		params += `"` + key + `"`

		first = false
	}

	params += `
		]
	   },
	    "dictionaries": {
	      "title": "Select dictionary to use",
	      "type": "string",
	      "enum": [ `

	first = true
	for _, key := range config.DictOrder {
		if !first {
			params += `,`
		}

		params += `"` + key + `"`

		first = false
	}

	params += `
      ]
    },
    "rules": {
      "title": "Select rule file to use",
      "type": "string",
      "enum": [ `

	first = true
	for _, key := range config.RulesOrder {
		if !first {
			params += `,`
		}

		params += `"` + key + `"`

		first = false
	}

	params += ` ]
	    },
	    "customdictadd": {
	      "title": "Custom Dictionary Additions",
	      "type": "string"
	    },
	    "hashes": {
	      "title": "Hashes",
	      "type": "string"
	    }
	  },
	  "required": [
	    "name",
	    "algorithm",
	    "dictionaries",
	    "hashes"
	  ]
	} } `

	return params
}

func (h *hashcatDictTooler) Requirements() string {
	return common.RES_GPU
}

func (h *hashcatDictTooler) NewTask(job common.Job) (common.Tasker, error) {
	return newHashcatTask(job)
}

func NewTooler() common.Tooler {
	return &hashcatDictTooler{}
}
