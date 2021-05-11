package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	log "github.com/Sirupsen/logrus"
	"github.com/jmmcatee/cracklord/common"
	"github.com/jmmcatee/cracklord/common/log"
	"github.com/jmmcatee/cracklord/common/resource"
	"github.com/jmmcatee/cracklord/plugins/tools/hashcat"
	"github.com/jmmcatee/cracklord/plugins/tools/hashcat3"
	"github.com/jmmcatee/cracklord/plugins/tools/johndict"
	"github.com/jmmcatee/cracklord/plugins/tools/nmap"
	"github.com/jmmcatee/cracklord/plugins/tools/testtimercpu"
	"github.com/jmmcatee/cracklord/plugins/tools/testtimergpu"
	"github.com/vaughan0/go-ini"
	"io/ioutil"
	"net/rpc"
	"os"
)

func main() {
	//Set our logger to STDERR and level
	log.SetOutput(os.Stderr)

	// Define the flags
	var confPath = flag.String("conf", "", "Configuration file to use")

	// Parse the flags
	flag.Parse()

	// Read the configuration file
	var confFile ini.File
	var confError error
	if *confPath == "" {
		log.Error("A configuration file was not included on the command line.")
		flag.PrintDefaults()
		return
	}
	confFile, confError = ini.LoadFile(*confPath)

	if confError != nil {
		log.Error("Unable to load configuration file:" + confError.Error())
		log.Error("See https://github.com/jmmcatee/cracklord/src/wiki/Configuration-Files.")
		return
	}

	//  Check for auth token
	resConf := confFile.Section("General")
	if len(resConf) == 0 {
		// We do not have configuration data to quit
		log.Error("There was a problem parsing the 'General' section of the configuration file.")
		log.Error("See https://github.com/jmmcatee/cracklord/src/wiki/Configuration-Files.")
		return
	}

	// Load the CA Certificate, Resource Key, and Resource Certificate from the config
	caCertPath, ok := resConf["CACertFile"]
	if !ok {
		log.Error("The CACertFile directive was not included in the 'General' section of the configuration file.")
		log.Error("See https://github.com/jmmcatee/cracklord/src/wiki/Configuration-Files.")
	}
	caCertPath = common.StripQuotes(caCertPath)

	resKeyPath, ok := resConf["KeyFile"]
	if !ok {
		log.Error("The KeyFile directive was not included in the 'General' section of the configuration file.")
		log.Error("See https://github.com/jmmcatee/cracklord/src/wiki/Configuration-Files.")
	}
	resKeyPath = common.StripQuotes(resKeyPath)

	resCertPath, ok := resConf["CertFile"]
	if !ok {
		log.Error("The CertFile directive was not included in the 'General' section of the configuration file.")
		log.Error("See https://github.com/jmmcatee/cracklord/src/wiki/Configuration-Files.")
	}
	resCertPath = common.StripQuotes(resCertPath)

	runIP, ok := resConf["BindIP"]
	if !ok {
		runIP = "0.0.0.0"
	} else {
		runIP = common.StripQuotes(runIP)
	}

	runPort, ok := resConf["BindPort"]
	if !ok {
		runPort = "9443"
	} else {
		runPort = common.StripQuotes(runPort)
	}

	switch common.StripQuotes(resConf["LogLevel"]) {
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

	lf := common.StripQuotes(resConf["LogFile"])
	if lf != "" {
		hook, err := cracklog.NewFileHook(lf)
		if err != nil {
			log.Error("Unable to open log file: " + err.Error())
		} else {
			log.AddHook(hook)
		}
	}

	log.WithFields(log.Fields{
		"conf": *confPath,
		"ip":   runIP,
		"port": runPort,
	}).Debug("Config file setup")

	// Create a resource queue
	resQueue := resource.NewResourceQueue()

	//Get the configuration section for plugins
	pluginConf := confFile.Section("Plugins")
	if len(pluginConf) == 0 {
		log.Error("No plugin section in the resource server config file.")
		log.Error("See https://github.com/jmmcatee/cracklord/src/wiki/Configuration-Files.")
		return
	}
	if common.StripQuotes(pluginConf["hashcat"]) != "" {
		hashcat.Setup(common.StripQuotes(pluginConf["hashcat"]))
		resQueue.AddTool(hashcat.NewTooler())
	}
	if common.StripQuotes(pluginConf["nmap"]) != "" {
		nmap.Setup(common.StripQuotes(pluginConf["nmap"]))
		resQueue.AddTool(nmap.NewTooler())
	}
	if common.StripQuotes(pluginConf["johndict"]) != "" {
		johndict.Setup(common.StripQuotes(pluginConf["johndict"]))
		resQueue.AddTool(johndict.NewTooler())
	}
	if common.StripQuotes(pluginConf["hashcat3"]) != "" {
		hashcat3.Setup(common.StripQuotes(pluginConf["hashcat3"]))
		resQueue.AddTool(hashcat3.NewTooler())
	}
	if common.StripQuotes(pluginConf["testtimer"]) == "true" {
		testtimergpu.Setup()
		testtimercpu.Setup()
		resQueue.AddTool(testtimergpu.NewTooler())
		resQueue.AddTool(testtimercpu.NewTooler())
	}

	// Get an RPC server
	res := rpc.NewServer()

	// Register the RPC endpoints
	res.Register(&resQueue)

	caBytes, err := ioutil.ReadFile(caCertPath)
	if err != nil {
		log.Error("Unable to read CA certificate: " + err.Error())
		return
	}

	// Load the CA file
	caPool := x509.NewCertPool()
	caPool.AppendCertsFromPEM(caBytes)

	// Load the cert and key files
	tlscert, err := tls.LoadX509KeyPair(resCertPath, resKeyPath)
	if err != nil {
		log.Error("There was an error loading the resource key or certificate files: " + err.Error())
		return
	}

	// Setup TLS connection
	tlsconfig := &tls.Config{}
	tlsconfig.Certificates = make([]tls.Certificate, 1)
	tlsconfig.Certificates[0] = tlscert
	tlsconfig.RootCAs = caPool
	tlsconfig.ClientCAs = caPool
	tlsconfig.ClientAuth = tls.RequireAndVerifyClientCert
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

	listen, err := tls.Listen("tcp", runIP+":"+runPort, tlsconfig)
	if err != nil {
		log.Error("Unable to bind to '" + runIP + ":" + runPort + "':" + err.Error())
		return
	}

	log.WithFields(log.Fields{
		"ip":   runIP,
		"port": runPort,
	}).Info("Listening for queueserver connection.")

	// Accept only one connection at a time
	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Error("Failed to accept connection: " + err.Error())
			return
		}

		res.ServeConn(conn)
	}

	log.Info("Connection closed, stopping resource server.")

	listen.Close()
}
