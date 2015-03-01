package main

import (
	"flag"
	"github.com/jmmcatee/cracklord/queue"
	"github.com/vaughan0/go-ini"
	"path/filepath"
)

func main() {
	// Define the flags
	var confPath = flag.String("conf", "", "Configuration file to use")
	var runIP = flag.String("host", "0.0.0.0", "IP to bind to")
	var runPort = flag.Int("port", "443", "Port to bind to")
	var certPath = flag.String("cert", "", "Custom certificate file to use")
	var keyPath = flag.String("key", "", "Custom key file to use")

	// Parse the flags
	flag.Parse()

	// Read the configuration file
	var confFile ini.File
	var confErr error
	if confPath == "" {
		confFile, confErr = ini.LoadFile("./cracklord.ini")
	} else {
		confFile, confErr = ini.LoadFile(confPath)
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
	confAuth := confFile.Section("Authenication")
	if confAuth == nil {
		println("No authentication configuration!")
		return
	}

	// Check for type of authentication and set conf
	switch confAuth["type"] {
	case "ActiveDirectory":
		var ad ADAuth

		var realm string
		if realm, ok := confAuth["realm"]; !ok {
			println("AD Auth chosen and no Realm given...")
			return
		}
		ad.Realm = realm

		gmap := map[string]string{}
		var (
			ro    string
			st    string
			admin string
		)
		if ro, ok := confAuth["ReadOnlyGroup"]; !ok {
			println("No ReadOnly group provided...")
			return
		}
		if st, ok := confAuth["StandardGroup"]; !ok {
			println("No Standard group provided...")
			return
		}
		if admin, ok := confAuth["AdminGroup"]; !ok {
			println("No Administrator group provided")
			return
		}

		gmap[ReadOnly] = ro
		gmap[StandardUser] = st
		gmap[Administrator] = admin

		ad.Setup(gmap)

		server.Auth = ad
	}

	// Configure the TokenStore
	server.T = NewTokenStore()

	// Configure the Queue
	server.Q = queue.NewQueue()

}
