package main

import (
	"flag"
	log "github.com/Sirupsen/logrus"
	"github.com/jmmcatee/cracklord/common/log"
	"github.com/jmmcatee/cracklord/resource"
	"github.com/jmmcatee/cracklord/resource/plugins/hashcatdict"
	"github.com/vaughan0/go-ini"
	"net"
	"net/rpc"
	"os"
)

func main() {
	//Set our logger to STDERR and level
	log.SetOutput(os.Stderr)

	// Define the flags
	var confPath = flag.String("conf", "", "Configuration file to use")
	var runIP = flag.String("host", "0.0.0.0", "IP to bind to")
	var runPort = flag.String("port", "9443", "Port to bind to")
	// var certPath = flag.String("cert", "", "Custom certificate file to use")
	// var keyPath = flag.String("key", "", "Custom key file to use")

	// Parse the flags
	flag.Parse()

	// Read the configuration file
	var confFile ini.File
	var confError error
	if *confPath == "" {
		confFile, confError = ini.LoadFile("./resource.ini")
	} else {
		confFile, confError = ini.LoadFile(*confPath)
	}

	if confError != nil {
		println("ERROR: Unable to " + confError.Error())
		println("See https://github.com/jmmcatee/cracklord/wiki/Configuration-Files.")
		return
	}

	//  Check for auth token
	resConf := confFile.Section("Resource")
	if len(resConf) == 0 {
		// We do not have configuration data to quit
		println("ERROR: There was a problem with your configuration file.")
		println("See https://github.com/jmmcatee/cracklord/wiki/Configuration-Files.")
		return
	}
	authToken := resConf["AuthToken"]
	if authToken == "" {
		println("ERROR: No authentication token given in configuration file.")
		println("See https://github.com/jmmcatee/cracklord/wiki/Configuration-Files.")
		return
	}

	switch resConf["LogLevel"] {
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

	if resConf["LogFile"] != "" {
		hook, err := cracklog.NewFileHook(resConf["LogFile"])
		if err != nil {
			println("ERROR: Unable to open log file: " + err.Error())
		} else {
			log.AddHook(hook)
		}
	}

	log.WithFields(log.Fields{
		"conf": *confPath,
		"ip":   *runIP,
		"port": *runPort,
	}).Debug("Config file setup")

	// Create a resource queue
	resQueue := resource.NewResourceQueue(authToken)

	// Setup and Add tools
	if *confPath == "" {
		hashcatdict.Setup("./")
	} else {
		hashcatdict.Setup(*confPath)
	}

	resQueue.AddTool(hashcatdict.NewTooler())

	res := rpc.NewServer()
	res.Register(&resQueue)

	listen, err := net.Listen("tcp", *runIP+":"+*runPort)
	if err != nil {
		println("ERROR: Unable to bind to '" + *runIP + ":" + *runPort + "':" + err.Error())
		return
	}

	conn, err := listen.Accept()
	if err != nil {
		println("ERROR: Failed to accept connection: " + err.Error())
		return
	}

	res.ServeConn(conn)

	listen.Close()
}
