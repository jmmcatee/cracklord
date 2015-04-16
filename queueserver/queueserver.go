package main

import (
	"flag"
	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/negroni"
	"github.com/jmmcatee/cracklord/common/log"
	"github.com/jmmcatee/cracklord/queue"
	"github.com/unrolled/secure"
	"github.com/vaughan0/go-ini"
	"net/http"
)

func main() {
	// Define the flags
	var confPath = flag.String("conf", "", "Configuration file to use")
	var runIP = flag.String("host", "0.0.0.0", "IP to bind to")
	var runPort = flag.String("port", "443", "Port to bind to")
	var certPath = flag.String("cert", "", "Custom certificate file to use")
	var keyPath = flag.String("key", "", "Custom key file to use")

	// Parse the flags
	flag.Parse()

	// Read the configuration file
	var confFile ini.File
	var confErr error
	if *confPath == "" {
		confFile, confErr = ini.LoadFile("./queueserver.ini")
	} else {
		confFile, confErr = ini.LoadFile(*confPath)
	}

	// Build the App Controller
	var server AppController

	// Check for errors
	if confErr != nil {
		println("ERROR: Unable to " + confErr.Error())
		println("See https://github.com/jmmcatee/cracklord/wiki/Configuration-Files.")
		return
	}

	genConf := confFile.Section("General")
	switch genConf["LogLevel"] {
	case "Debug":
		log.SetLevel(log.DebugLevel)
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

	if genConf["LogFile"] != "" {
		hook, err := cracklog.NewFileHook(genConf["LogFile"])
		if err != nil {
			println("ERROR: Unable to open log file: " + err.Error())
		} else {
			log.AddHook(hook)
		}
	}

	log.WithFields(log.Fields{
		"ip":   *runIP,
		"port": *runPort,
	}).Info("Starting queue server up.")

	// Get the Authentication configuration
	confAuth := confFile.Section("Authentication")
	if confAuth == nil {
		println("Error: Authentication configuration is required.")
		println("See https://github.com/jmmcatee/cracklord/wiki/Configuration-Files.")
		return
	}

	// Check for type of authentication and set conf
	switch confAuth["type"] {
	case "INI":
		var i INIAuth

		// Get the users
		umap := map[string]string{}

		au, ok := confAuth["adminuser"]
		if !ok {
			log.Fatal("An administrative user was not configured. See https://github.com/jmmcatee/cracklord/wiki/Configuration-Files#queue-auth")
		}
		ap := confAuth["adminpass"]
		if !ok {
			log.Fatal("An administrative password was not configured. See https://github.com/jmmcatee/cracklord/wiki/Configuration-Files#queue-auth")
		}

		su, ok := confAuth["standarduser"]
		if !ok {
			log.Fatal("An standard user was not configured. See https://github.com/jmmcatee/cracklord/wiki/Configuration-Files#queue-auth")
		}
		sp := confAuth["standarduser"]
		if !ok {
			log.Fatal("An standard password was not configured. See https://github.com/jmmcatee/cracklord/wiki/Configuration-Files#queue-auth")
		}

		ru, ok := confAuth["readonlyuser"]
		if !ok {
			log.Fatal("An read only user was not configured. See https://github.com/jmmcatee/cracklord/wiki/Configuration-Files#queue-auth")
		}
		rp := confAuth["readonlypass"]
		if !ok {
			log.Fatal("An read only password was not configured. See https://github.com/jmmcatee/cracklord/wiki/Configuration-Files#queue-auth")
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

		realm, ok := confAuth["realm"]
		if !ok {
			log.Fatal("No Active Directory realm was configured. See https://github.com/jmmcatee/cracklord/wiki/Configuration-Files#queue-auth")
		}
		ad.Realm = realm

		gmap := map[string]string{}
		ro, ok := confAuth["ReadOnlyGroup"]
		if !ok {
			log.Fatal("A read only group was not provided. See https://github.com/jmmcatee/cracklord/wiki/Configuration-Files#queue-auth")
		}
		st, ok := confAuth["StandardGroup"]
		if !ok {
			log.Fatal("A group for standard access was not configured. See https://github.com/jmmcatee/cracklord/wiki/Configuration-Files#queue-auth")
		}
		admin, ok := confAuth["AdminGroup"]
		if !ok {
			log.Fatal("A group for read only access was not configured. See https://github.com/jmmcatee/cracklord/wiki/Configuration-Files#queue-auth")
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
	server.Q = queue.NewQueue()

	// Add some nice security stuff
	secureMiddleware := secure.New(secure.Options{
		SSLRedirect:             true,
		FrameDeny:               true,
		CustomFrameOptionsValue: "SAMEORIGIN",
		BrowserXssFilter:        true,
		IsDevelopment:           true,
	})

	// Build the Negroni handler
	n := negroni.New(negroni.NewRecovery(), cracklog.NewNegroniLogger(), negroni.NewStatic(http.Dir("public")))
	n.Use(negroni.HandlerFunc(secureMiddleware.HandlerFuncWithNext))
	n.UseHandler(server.Router())
	log.Debug("Negroni handler started.")

	// Check for given certs and generate a new one if none exist
	var cFile, kFile string
	if *certPath == "" || *keyPath == "" {
		log.Info("No certificate provided, generating self-signed certificates")
		// We need to create certs
		err := genNewCert("") // Gen file in local directory

		if err != nil {
			log.Fatal("An error occured while attempting to generate certificates")
		}

		cFile = "cert.pem"
		kFile = "cert.key"
	} else {
		cFile = *certPath
		kFile = *keyPath
		log.WithFields(log.Fields{
			"public-cert": *certPath,
			"private-key": *keyPath,
		}).Info("Utilizing provided certificates")
	}

	err := http.ListenAndServeTLS(*runIP+":"+*runPort, cFile, kFile, n)
	if err != nil {
		log.Fatal("Unable to start up web server: "+err.Error())
	}
}
