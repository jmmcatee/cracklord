package main

import (
	"flag"
	"github.com/codegangsta/negroni"
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
		confFile, confErr = ini.LoadFile("./cracklord.ini")
	} else {
		confFile, confErr = ini.LoadFile(*confPath)
	}

	// Build the App Controller
	var server AppController

	// Check for errors
	if confErr != nil {
		println("Error in configuration read:")
		println("\t" + confErr.Error())
		return
	}

	// Get the Authentication configuration
	confAuth := confFile.Section("Authentication")
	if confAuth == nil {
		println("No authentication configuration!")
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
			println("Admin user not present...")
			return
		}
		ap := confAuth["adminpass"]
		if !ok {
			println("Admin password not present...")
			return
		}

		su, ok := confAuth["standarduser"]
		if !ok {
			println("Standard user not present...")
			return
		}
		sp := confAuth["standarduser"]
		if !ok {
			println("Standard password not present...")
			return
		}

		ru, ok := confAuth["readonlyuser"]
		if !ok {
			println("ReadOnly user not present...")
			return
		}
		rp := confAuth["readonlypass"]
		if !ok {
			println("ReadOnly password not present...")
			return
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
	case "ActiveDirectory":
		var ad ADAuth

		realm, ok := confAuth["realm"]
		if !ok {
			println("AD Auth chosen and no Realm given...")
			return
		}
		ad.Realm = realm

		gmap := map[string]string{}
		ro, ok := confAuth["ReadOnlyGroup"]
		if !ok {
			println("No ReadOnly group provided...")
			return
		}
		st, ok := confAuth["StandardGroup"]
		if !ok {
			println("No Standard group provided...")
			return
		}
		admin, ok := confAuth["AdminGroup"]
		if !ok {
			println("No Administrator group provided")
			return
		}

		gmap[ReadOnly] = ro
		gmap[StandardUser] = st
		gmap[Administrator] = admin

		ad.Setup(gmap)

		server.Auth = &ad
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
	n := negroni.Classic()
	n.Use(negroni.HandlerFunc(secureMiddleware.HandlerFuncWithNext))
	n.UseHandler(server.Router())

	// Check for given certs and generate a new one if none exist
	var cFile, kFile string
	if *certPath == "" || *keyPath == "" {
		// We need to create certs
		err := genNewCert("") // Gen file in local directory

		if err != nil {
			println("Error generating TLS certs...")
			return
		}

		cFile = "cert.pem"
		kFile = "cert.key"
	} else {
		cFile = *certPath
		kFile = *keyPath
	}

	http.ListenAndServeTLS(*runIP+":"+*runPort, cFile, kFile, n)
}
