package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	log "github.com/Sirupsen/logrus"
	"github.com/jmmcatee/cracklord/common/log"
	"github.com/jmmcatee/cracklord/common/resource"
	"github.com/jmmcatee/cracklord/common/resource/plugins/hashcatdict"
	"github.com/vaughan0/go-ini"
	"io/ioutil"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
)

const (
	RESOURCED_INIT_FILE = "/etc/cracklord/resourced.conf"
	CA_CERT_FILE        = "/etc/cracklord/ssl/cracklord_ca.pem"
	RESOURCED_KEY_FILE  = "/etc/cracklord/ssl/resourced.key"
	RESOURCED_CERT_FILE = "/etc/cracklord/ssl/resourced.crt"
)

func main() {
	//Set our logger to STDERR and level
	log.SetOutput(os.Stderr)

	// Define the flags
	var confPath = flag.String("conf", "", "Configuration file to use")
	var runIP = flag.String("host", "0.0.0.0", "IP to bind to")
	var runPort = flag.String("port", "9443", "Port to bind to")
	var caCertPath = flag.String("cacert", "", "CA Certficate to use for validation")
	var resKeyPath = flag.String("key", "", "Private key for the resource to use over TLS")
	var resCertPath = flag.String("cert", "", "Certicate to use with the resource for TLS")

	// Parse the flags
	flag.Parse()

	// Read the configuration file
	var confFile ini.File
	var confError error
	if *confPath == "" {
		confFile, confError = ini.LoadFile(RESOURCED_INIT_FILE)
	} else {
		confFile, confError = ini.LoadFile(*confPath)
	}

	if confError != nil {
		println("ERROR: Unable to " + confError.Error())
		println("See https://github.com/jmmcatee/cracklord/src/wiki/Configuration-Files.")
		return
	}

	//  Check for auth token
	resConf := confFile.Section("General")
	if len(resConf) == 0 {
		// We do not have configuration data to quit
		println("ERROR: There was a problem with your configuration file.")
		println("See https://github.com/jmmcatee/cracklord/src/wiki/Configuration-Files.")
		return
	}
	authToken := resConf["AuthToken"]
	if authToken == "" {
		println("ERROR: No authentication token given in configuration file.")
		println("See https://github.com/jmmcatee/cracklord/src/wiki/Configuration-Files.")
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
	resQueue := resource.NewResourceQueue()

	//Get the configuration section for plugins
	pluginConf := confFile.Section("Plugins")
	if len(pluginConf) == 0 {
		println("ERROR: No plugin section in the resource server config file.")
		println("See https://github.com/jmmcatee/cracklord/src/wiki/Configuration-Files.")
		return
	}
	if pluginConf["hashcatdict"] != "" {
		hashcatdict.Setup(pluginConf["hashcatdict"])
		resQueue.AddTool(hashcatdict.NewTooler())
	}

	// Get an RPC server
	res := rpc.NewServer()

	// Register the RPC endpoints
	res.Register(&resQueue)

	// Get the CA
	if *caCertPath == "" {
		// Use default path to get the CA certificate
		*caCertPath = CA_CERT_FILE
	}

	caBytes, err := ioutil.ReadFile(*caCertPath)
	if err != nil {
		println("ERROR: " + err.Error())
	}

	caPool := x509.NewCertPool()
	caPool.AppendCertsFromPEM(caBytes)

	// Get the certificates and private key
	if *resCertPath == "" || *resKeyPath == "" {
		// The private key and/or the certificate were not given so go with defaults
		*resKeyPath = RESOURCED_KEY_FILE
		*resCertPath = RESOURCED_CERT_FILE
	}

	tlscert, err := tls.LoadX509KeyPair(*resCertPath, *resKeyPath)

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

	listen, err := tls.Listen("tcp", *runIP+":"+*runPort, tlsconfig)
	if err != nil {
		println("ERROR: Unable to bind to '" + *runIP + ":" + *runPort + "':" + err.Error())
		return
	}

	log.WithFields(log.Fields{
		"ip":   *runIP,
		"port": *runPort,
	}).Info("Listening for queueserver connection.")

	// Accept only one connection at a time
	for {
		conn, err := listen.Accept()
		if err != nil {
			println("ERROR: Failed to accept connection: " + err.Error())
			return
		}

		res.ServeCodec(jsonrpc.NewServerCodec(conn))
	}

	log.Info("Connection closed, stopping resource server.")

	listen.Close()
}
