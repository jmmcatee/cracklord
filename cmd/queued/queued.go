package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/negroni"
	"github.com/jmmcatee/cracklord/common"
	"github.com/jmmcatee/cracklord/common/log"
	"github.com/jmmcatee/cracklord/common/queue"
	"github.com/unrolled/secure"
	"github.com/vaughan0/go-ini"
	"io"
	"net/http"
	"os"
	"strconv"
)

const (
	QUEUED_INIT_FILE = "/etc/cracklord/queued.conf"
	CA_CERT_FILE     = "/etc/cracklord/ssl/cracklord_ca.pem"
	QUEUED_KEY_FILE  = "/etc/cracklord/ssl/queued.key"
	QUEUED_CERT_FILE = "/etc/cracklord/ssl/queued.crt"
)

func main() {
	// Define the flags
	var confPath = flag.String("conf", "", "Configuration file to use")
	var webRoot = flag.String("webroot", "./public", "Location of the web server root")
	var runIP = flag.String("host", "0.0.0.0", "IP to bind to")
	var runPort = flag.String("port", "443", "Port to bind to")
	var caCertPath = flag.String("cacert", "", "CA Certficate to use for validation")
	var KeyPath = flag.String("key", "", "Private key for the resource to use over TLS")
	var CertPath = flag.String("cert", "", "Certicate to use with the resource for TLS")

	// Parse the flags
	flag.Parse()

	// Read the configuration file
	var confFile ini.File
	var confErr error
	if *confPath == "" {
		confFile, confErr = ini.LoadFile(QUEUED_INIT_FILE)
	} else {
		confFile, confErr = ini.LoadFile(*confPath)
	}

	// Build the App Controller
	var server AppController

	// Check for errors
	if confErr != nil {
		println("ERROR: Unable to " + confErr.Error())
		println("See https://github.com/jmmcatee/cracklord/src/wiki/Configuration-Files.")
		return
	}

	genConf := confFile.Section("General")
	switch common.StripQuotes(genConf["LogLevel"]) {
	case "Debug":
		log.SetLevel(log.DebugLevel)
		log.Warn("Please note the debug level logs may contain sensitive information!")
	case "Info":
		log.SetLevel(log.InfoLevel)
	case "Warn":
		log.SetLevel(log.WarnLevel)
	case "Error":
		log.SetLevel(log.ErrorLevel)
	case "Fatal":
		log.SetLevel(log.FatalLevel)
	case "Panic":
		log.SetLevel(log.PanicLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}

	lf := common.StripQuotes(genConf["LogFile"])
	if lf != "" {
		hook, err := cracklog.NewFileHook(lf)
		if err != nil {
			println("ERROR: Unable to open log file: " + err.Error())
		} else {
			log.AddHook(hook)
		}
	}

	var statefile string
	statefile = common.StripQuotes(genConf["StateFile"])

	var updatetime int
	var resourcetimeout int
	utconf := common.StripQuotes(genConf["UpdateTime"])
	if ufconf != "" {
		var err error
		updatetime, err = strconv.Atoi(utconf)
		if err != nil {
			log.WithField("error", err.Error()).Error("Unable to parse update time in config file.")
			updatetime = 30
		}
	} else {
		updatetime = 30
	}
	restimeconf = common.StripQuotes(genConf["ResourceTimeout"])
	if restimeconf != "" {
		var err error
		resourcetimeout, err = strconv.Atoi(restimeconf)
		if err != nil {
			log.WithField("error", err.Error()).Error("Unable to parse resource timeout in config file.")
			resourcetimeout = 5
		}
	} else {
		resourcetimeout = 5
	}

	log.WithFields(log.Fields{
		"ip":   *runIP,
		"port": *runPort,
	}).Info("Starting queue server up.")

	// Get the Authentication configuration
	confAuth := confFile.Section("Authentication")
	if confAuth == nil {
		println("Error: Authentication configuration is required.")
		println("See https://github.com/jmmcatee/cracklord/src/wiki/Configuration-Files.")
		return
	}

	_, weberr := os.Stat(*webRoot)
	if weberr != nil {
		println("Error: Public web root '" + *webRoot + "' does not exist.")
		return
	}

	// Check for type of authentication and set conf
	switch confAuth["type"] {
	case "INI":
		var i INIAuth

		// Get the users
		umap := map[string]string{}

		au, ok := common.StripQuotes(confAuth["adminuser"])
		if !ok {
			log.Fatal("An administrative user was not configured. See https://github.com/jmmcatee/cracklord/src/wiki/Configuration-Files#queue-auth")
		}
		ap := common.StripQuotes(confAuth["adminpass"])
		if !ok {
			log.Fatal("An administrative password was not configured. See https://github.com/jmmcatee/cracklord/src/wiki/Configuration-Files#queue-auth")
		}

		su, ok := common.StripQuotes(confAuth["standarduser"])
		if !ok {
			log.Fatal("An standard user was not configured. See https://github.com/jmmcatee/cracklord/src/wiki/Configuration-Files#queue-auth")
		}
		sp := common.StripQuotes(confAuth["standarduser"])
		if !ok {
			log.Fatal("An standard password was not configured. See https://github.com/jmmcatee/cracklord/src/wiki/Configuration-Files#queue-auth")
		}

		ru, ok := common.StripQuotes(confAuth["readonlyuser"])
		if !ok {
			log.Fatal("An read only user was not configured. See https://github.com/jmmcatee/cracklord/src/wiki/Configuration-Files#queue-auth")
		}
		rp := common.StripQuotes(confAuth["readonlypass"])
		if !ok {
			log.Fatal("An read only password was not configured. See https://github.com/jmmcatee/cracklord/src/wiki/Configuration-Files#queue-auth")
		}

		umap[au] = ap
		umap[su] = sp
		umap[ru] = rp

		// Setup group mappings
		gmap := map[string]string{}

		gmap[au] = Administrator
		gmap[su] = StandardUser
		gmap[ru] = ReadOnly

		i.Setup(umap, gmap)

		server.Auth = &i

		log.Info("INI authentication setup complete.")
	case "ActiveDirectory":
		var ad ADAuth

		realm, ok := common.StripQuotes(confAuth["realm"])
		if !ok {
			log.Fatal("No Active Directory realm was configured. See https://github.com/jmmcatee/cracklord/src/wiki/Configuration-Files#queue-auth")
		}
		ad.SetRealm(realm)

		gmap := map[string]string{}
		ro, ok := common.StripQuotes(confAuth["ReadOnlyGroup"])
		if !ok {
			log.Fatal("A read only group was not provided. See https://github.com/jmmcatee/cracklord/src/wiki/Configuration-Files#queue-auth")
		}
		st, ok := common.StripQuotes(confAuth["StandardGroup"])
		if !ok {
			log.Fatal("A group for standard access was not configured. See https://github.com/jmmcatee/cracklord/src/wiki/Configuration-Files#queue-auth")
		}
		admin, ok := common.StripQuotes(confAuth["AdminGroup"])
		if !ok {
			log.Fatal("A group for read only access was not configured. See https://github.com/jmmcatee/cracklord/src/wiki/Configuration-Files#queue-auth")
		}

		gmap[ReadOnly] = ro
		gmap[StandardUser] = st
		gmap[Administrator] = admin

		ad.Setup(gmap)

		server.Auth = &ad
		log.WithFields(log.Fields{
			"readonly": ro,
			"standard": st,
			"admin":    admin,
		}).Info("Active directory authentication configured successfully.")
	}

	// Configure the TokenStore
	server.T = NewTokenStore()

	// Configure the Queue
	server.Q = queue.NewQueue(statefile, updatetime, resourcetimeout)

	// Get the CA
	var caFile *os.File
	if *caCertPath == "" {
		// Use default path to get the CA certificate
		*caCertPath = CA_CERT_FILE
	}

	caFile, err := os.Open(*caCertPath)
	caBytes := []byte{}
	if err != nil {
		println("ERROR: " + err.Error())
	}

	if _, err := io.ReadFull(caFile, caBytes); err != nil {
		println("ERROR: " + err.Error())
	}

	caPool := x509.NewCertPool()
	caPool.AppendCertsFromPEM(caBytes)

	// Get the certificates and private key
	if *CertPath == "" || *KeyPath == "" {
		// The private key and/or the certificate were not given so go with defaults
		*KeyPath = QUEUED_KEY_FILE
		*CertPath = QUEUED_CERT_FILE
	}

	tlscert, err := tls.LoadX509KeyPair(*CertPath, *KeyPath)

	// Setup TLS connection
	tlsconfig := &tls.Config{}
	tlsconfig.Certificates = make([]tls.Certificate, 1)
	tlsconfig.Certificates[0] = tlscert
	tlsconfig.RootCAs = caPool
	tlsconfig.ClientCAs = caPool
	tlsconfig.CipherSuites = []uint16{tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256}
	tlsconfig.MinVersion = tls.VersionTLS12
	tlsconfig.SessionTicketsDisabled = true

	// Set the TLS policy for the Queue to communicate with the resources
	server.TLS = tlsconfig

	// Add some nice security stuff
	secureMiddleware := secure.New(secure.Options{
		SSLRedirect:             true,
		FrameDeny:               true,
		CustomFrameOptionsValue: "SAMEORIGIN",
		BrowserXssFilter:        true,
		IsDevelopment:           true,
	})

	// Build the Negroni handler
	n := negroni.New(negroni.NewRecovery(),
		cracklog.NewNegroniLogger(),
		negroni.NewStatic(http.Dir(*webRoot)))

	n.Use(negroni.HandlerFunc(secureMiddleware.HandlerFuncWithNext))
	n.UseHandler(server.Router())
	log.Debug("Negroni handler started.")

	listen, err := tls.Listen("tcp", *runIP+":"+*runPort, tlsconfig)
	if err != nil {
		println("ERROR: Unable to bind to '" + *runIP + ":" + *runPort + "':" + err.Error())
		return
	}

	err = http.Serve(listen, n)
	if err != nil {
		log.Fatal("Unable to start up web server: " + err.Error())
	}
}
