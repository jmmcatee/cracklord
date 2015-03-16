package main

import (
	"flag"
	"github.com/jmmcatee/cracklord/common"
	"github.com/jmmcatee/cracklord/resource"
	"github.com/jmmcatee/cracklord/resource/plugins/hashcatdict"
	"github.com/vaughan0/go-ini"
	"net"
	"net/rpc"
)

var tools []common.Tooler

func init() {
	// Add tool plugins here
	tools = append(tools, hashcatdict.NewTooler())
}

func main() {
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
		println("Error reading config: " + confError.Error())
		return
	}

	//  Check for auth token
	resConf := confFile.Section("Resource")
	if len(resConf) == 0 {
		// We do not have configuration data to quit
		println("No configuration for resource.")
		return
	}
	authToken := resConf["AuthToken"]
	if authToken == "" {
		println("No authentication token given in configuration file.")
		return
	}

	// Create a resource queue
	resQueue := resource.NewResourceQueue(authToken)

	// Add tools
	for i, _ := range tools {
		resQueue.AddTool(tools[i])
	}

	res := rpc.NewServer()
	res.Register(&resQueue)

	listen, err := net.Listen("tcp", *runIP+":"+*runPort)
	if err != nil {
		println("Failed to bind: " + err.Error())
		return
	}

	conn, err := listen.Accept()
	if err != nil {
		println("Failed to accept connection: " + err.Error())
		return
	}

	res.ServeConn(conn)

	listen.Close()
}
