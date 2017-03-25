package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/negroni"
	"github.com/jmmcatee/cracklord/common"
	"github.com/jmmcatee/cracklord/common/log"
	"github.com/jmmcatee/cracklord/common/queue"
	"github.com/jmmcatee/cracklord/plugins/resourcemanagers/aws"
	"github.com/jmmcatee/cracklord/plugins/resourcemanagers/directconnect"
	"github.com/unrolled/secure"
	ini "github.com/vaughan0/go-ini"
)

func processHookSection(section map[string]string) []string {
	var results []string

	for k, v := range section {
		if v == "true" {
			results = append(results, k)
		}
	}

	return results
}

func main() {
	// Define the flags
	var confPath = flag.String("conf", "", "Configuration file to use")

	// Parse the flags
	flag.Parse()

	// Read the configuration file
	var confFile ini.File
	var confErr error
	if *confPath == "" {
		log.Error("A configuration file was not defined.")
		flag.PrintDefaults()
	}
	confFile, confErr = ini.LoadFile(*confPath)

	// Build the App Controller
	var server AppController

	// Check for errors
	if confErr != nil {
		println("ERROR: Unable to " + confErr.Error())
		println("See https://github.com/jmmcatee/cracklord/src/wiki/Configuration-Files.")
		return
	}

	genConf := confFile.Section("General")

	// Load the CA Certificate, Resource Key, and Resource Certificate from the config
	webRoot, ok := genConf["WebRoot"]
	if !ok {
		log.Error("The WebRoot directive was not included in the 'General' section of the configuration file.")
		log.Error("See https://github.com/jmmcatee/cracklord/src/wiki/Configuration-Files.")
	}
	webRoot = common.StripQuotes(webRoot)

	// Load the CA Certificate, Resource Key, and Resource Certificate from the config
	caCertPath, ok := genConf["CACertFile"]
	if !ok {
		log.Error("The CACertFile directive was not included in the 'General' section of the configuration file.")
		log.Error("See https://github.com/jmmcatee/cracklord/src/wiki/Configuration-Files.")
	}
	caCertPath = common.StripQuotes(caCertPath)

	// Load the CA Certificate, Resource Key, and Resource Certificate from the config
	caKeyPath, ok := genConf["CAKeyFile"]
	if !ok {
		log.Error("The CAKeyFile directive was not included in the 'General' section of the configuration file.")
		log.Error("See https://github.com/jmmcatee/cracklord/src/wiki/Configuration-Files.")
	}
	caKeyPath = common.StripQuotes(caKeyPath)

	KeyPath, ok := genConf["KeyFile"]
	if !ok {
		log.Error("The KeyFile directive was not included in the 'General' section of the configuration file.")
		log.Error("See https://github.com/jmmcatee/cracklord/src/wiki/Configuration-Files.")
	}
	KeyPath = common.StripQuotes(KeyPath)

	CertPath, ok := genConf["CertFile"]
	if !ok {
		log.Error("The KeyFile directive was not included in the 'General' section of the configuration file.")
		log.Error("See https://github.com/jmmcatee/cracklord/src/wiki/Configuration-Files.")
	}
	CertPath = common.StripQuotes(CertPath)

	// Check for an API SSL/TLS key/cert pair to use
	var useSepAPITLS bool
	APICertPath, APICertOK := genConf["APICertFile"]
	APIKeyPath, APIKeyOK := genConf["APIKeyFile"]
	if APIKeyOK && APICertOK {
		// We found a key and certificate for the API to use so change our boolean marker
		useSepAPITLS = true
	}

	runIP, ok := genConf["BindIP"]
	if !ok {
		runIP = "0.0.0.0"
	} else {
		runIP = common.StripQuotes(runIP)
	}

	runPort, ok := genConf["BindPort"]
	if !ok {
		runPort = "9443"
	} else {
		runPort = common.StripQuotes(runPort)
	}

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
	if utconf != "" {
		var err error
		updatetime, err = strconv.Atoi(utconf)
		if err != nil {
			log.WithField("error", err.Error()).Error("Unable to parse update time in config file.")
			updatetime = 30
		}
	} else {
		updatetime = 30
	}
	restimeconf := common.StripQuotes(genConf["ResourceTimeout"])
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
		"ip":   runIP,
		"port": runPort,
	}).Info("Starting queue server up.")

	// Get the Authentication configuration
	confAuth := confFile.Section("Authentication")
	if confAuth == nil {
		println("Error: Authentication configuration is required.")
		println("See https://github.com/jmmcatee/cracklord/src/wiki/Configuration-Files.")
		return
	}

	_, weberr := os.Stat(webRoot)
	if weberr != nil {
		println("Error: Public web root '" + webRoot + "' does not exist.")
		return
	}

	// Check for type of authentication and set conf
	switch confAuth["type"] {
	case "INI":
		var i INIAuth

		// Get the users
		umap := map[string]string{}

		au := common.StripQuotes(confAuth["adminuser"])
		if au == "" {
			log.Fatal("An administrative user was not configured. See https://github.com/jmmcatee/cracklord/src/wiki/Configuration-Files#queue-auth")
		}
		ap := common.StripQuotes(confAuth["adminpass"])
		if ap == "" {
			log.Fatal("An administrative password was not configured. See https://github.com/jmmcatee/cracklord/src/wiki/Configuration-Files#queue-auth")
		}

		su := common.StripQuotes(confAuth["standarduser"])
		if su == "" {
			log.Fatal("An standard user was not configured. See https://github.com/jmmcatee/cracklord/src/wiki/Configuration-Files#queue-auth")
		}
		sp := common.StripQuotes(confAuth["standarduser"])
		if sp == "" {
			log.Fatal("An standard password was not configured. See https://github.com/jmmcatee/cracklord/src/wiki/Configuration-Files#queue-auth")
		}

		ru := common.StripQuotes(confAuth["readonlyuser"])
		if ru == "" {
			log.Fatal("An read only user was not configured. See https://github.com/jmmcatee/cracklord/src/wiki/Configuration-Files#queue-auth")
		}
		rp := common.StripQuotes(confAuth["readonlypass"])
		if rp == "" {
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

		realm := common.StripQuotes(confAuth["realm"])
		if realm == "" {
			log.Fatal("No Active Directory realm was configured. See https://github.com/jmmcatee/cracklord/src/wiki/Configuration-Files#queue-auth")
		}
		ad.SetRealm(realm)

		gmap := map[string]string{}
		ro := common.StripQuotes(confAuth["ReadOnlyGroup"])
		if ro == "" {
			log.Fatal("A read only group was not provided. See https://github.com/jmmcatee/cracklord/src/wiki/Configuration-Files#queue-auth")
		}
		st := common.StripQuotes(confAuth["StandardGroup"])
		if st == "" {
			log.Fatal("A group for standard access was not configured. See https://github.com/jmmcatee/cracklord/src/wiki/Configuration-Files#queue-auth")
		}
		admin := common.StripQuotes(confAuth["AdminGroup"])
		if admin == "" {
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

	var hooks queue.HookParameters

	hooksConf := confFile.Section("Hooks")
	if hooksConf == nil {
		hooks.ScriptTimeout = 60
	} else {
		hooks.ScriptTimeout, _ = strconv.Atoi(hooksConf["scripttimeout"])
	}

	hooks.JobCreate = processHookSection(confFile.Section("Hooks.JobCreate"))
	hooks.JobFinish = processHookSection(confFile.Section("Hooks.JobFinish"))
	hooks.JobStart = processHookSection(confFile.Section("Hooks.JobStart"))
	hooks.ResourceConnect = processHookSection(confFile.Section("Hooks.ResourceConnect"))
	hooks.QueueReorder = processHookSection(confFile.Section("Hooks.QueueReorder"))

	// Configure the TokenStore
	server.T = NewTokenStore()

	// Configure the Queue
	server.Q = queue.NewQueue(statefile, updatetime, resourcetimeout, hooks)

	caBytes, err := ioutil.ReadFile(caCertPath)
	if err != nil {
		println("ERROR: " + err.Error())
	}

	caPool := x509.NewCertPool()
	caPool.AppendCertsFromPEM(caBytes)

	tlscert, err := tls.LoadX509KeyPair(CertPath, KeyPath)
	if err != nil {
		log.Fatalf("Failed to load cert pair for Q and R connecton. %s\n", err.Error())
	}

	// Setup TLS connection for the Queue and Resource communication
	qandrTLSConfig := &tls.Config{}
	qandrTLSConfig.Certificates = make([]tls.Certificate, 1)
	qandrTLSConfig.Certificates[0] = tlscert
	qandrTLSConfig.RootCAs = caPool
	qandrTLSConfig.ClientCAs = caPool
	qandrTLSConfig.CipherSuites = []uint16{tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256}
	qandrTLSConfig.MinVersion = tls.VersionTLS12
	qandrTLSConfig.SessionTicketsDisabled = true

	// Check if we are using a different TLS configuration for the API portion of the Queue
	if useSepAPITLS {
		apiCert, err := tls.LoadX509KeyPair(APICertPath, APIKeyPath)
		if err != nil {
			log.Fatalf("API Cert and Key set, but could not be loaded. %s\n", err.Error())
		}
		apiTLSConfig := &tls.Config{
			Certificates: []tls.Certificate{apiCert},
			CipherSuites: []uint16{tls.TLS_RSA_WITH_AES_128_CBC_SHA,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
				tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256},
			MinVersion:             tls.VersionTLS12,
			SessionTicketsDisabled: true,
		}

		server.TLS = apiTLSConfig
	} else {
		// No separate API key was set so use the same we did for the internal communication
		server.TLS = qandrTLSConfig
	}

	// Add some nice security stuff
	secureMiddleware := secure.New(secure.Options{
		SSLRedirect:             true,
		FrameDeny:               true,
		CustomFrameOptionsValue: "SAMEORIGIN",
		BrowserXssFilter:        true,
		IsDevelopment:           true,
	})

	// SETUP RESOURCE MANAGERS
	// Get the Authentication configuration
	confResMgr := confFile.Section("ResourceManagers")
	if confResMgr == nil {
		println("Error: Resource manager configuration is required.")
		println("See https://github.com/jmmcatee/cracklord/src/wiki/Configuration-Files.")
		return
	}

	// First, let's setup the direct connect manager if we have anything there
	if _, ok := confResMgr["directconnect"]; ok {
		resmgr_dc := directconnectresourcemanager.Setup(&server.Q, qandrTLSConfig)
		server.Q.AddResourceManager(resmgr_dc)
	}

	// Now let's setup the AWS manager if we have a config file
	if resDC, ok := confResMgr["aws"]; ok {
		resmgr_aws, err := awsresourcemanager.Setup(resDC, &server.Q, qandrTLSConfig, caCertPath, caKeyPath)
		if err != nil {
			log.WithField("error", err.Error()).Error("Unable to setup AWS resource manager.")
		} else {
			server.Q.AddResourceManager(resmgr_aws)
		}
	}

	// Build the Negroni handler
	n := negroni.New(negroni.NewRecovery(),
		cracklog.NewNegroniLogger(),
		negroni.NewStatic(http.Dir(webRoot)))

	n.Use(negroni.HandlerFunc(secureMiddleware.HandlerFuncWithNext))
	n.UseHandler(server.Router())
	log.Debug("Negroni handler started.")

	listen, err := tls.Listen("tcp", runIP+":"+runPort, server.TLS)
	if err != nil {
		println("ERROR: Unable to bind to '" + runIP + ":" + runPort + "':" + err.Error())
		return
	}

	err = http.Serve(listen, n)
	if err != nil {
		log.Fatal("Unable to start up web server: " + err.Error())
	}
}
