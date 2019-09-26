package queue

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"testing"

	"github.com/jmmcatee/cracklord/common"
	"github.com/jmmcatee/cracklord/common/resource"
)

func TestBoltSave(t *testing.T) {
	bdb, err := NewJobDB("test.db")
	if err != nil {
		t.Fatal(err)
	}

	job := common.Job{
		UUID: "727cb3c9-6ca6-41be-a680-f3d03cfa62a0",
		Name: "test 1",
		Parameters: map[string]string{
			"adv_options_loopback":     "false",
			"brute_increment":          "true",
			"brute_max_length":         "9",
			"brute_min_length":         "7",
			"brute_predefined_charset": "UPPER, lower, Number [9]",
			"brute_use_custom_chars":   "false",
			"dict_rules_use_random":    "false",
			"hashes_use_upload":        "false",
			"hashmode":                 "100",
			"use_adv_options":          "false",
		},
	}

	err = bdb.AddJob(job)
	if err != nil {
		log.Fatal(err)
	}

	err = bdb.boltdb.Sync()
	if err != nil {
		log.Fatal(err)
	}

	job2, err := bdb.GetJob("727cb3c9-6ca6-41be-a680-f3d03cfa62a0")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Job 1: %#v\n", job)
	fmt.Printf("Job 2: %#v\n", job2)
}

func TestRPC001(t *testing.T) {
	// Create a resource queue
	resQueue := resource.NewResourceQueue()

	// Get an RPC server
	res := rpc.NewServer()

	// Register the RPC endpoints
	res.Register(&resQueue)

	listen, err := net.Listen("tcp", "localhost:9443")
	if err != nil {
		t.Fatal(err)
	}

	// Accept only one connection at a time
	go func() {
		for {
			conn, err := listen.Accept()
			if err != nil {
				t.Fatal(err)
				return
			}

			res.ServeConn(conn)
		}
	}()

	conn, err := net.Dial("tcp", "localhost:9443")
	if err != nil {
		t.Fatal(err)
	}

	// Build the RPC client for the resource
	rpcClient := rpc.NewClient(conn)
	if err != nil {
		t.Fatal(err)
	}

	job := common.Job{
		UUID: "727cb3c9-6ca6-41be-a680-f3d03cfa62a0",
		Name: "test 1",
		Parameters: map[string]string{
			"adv_options_loopback":     "false",
			"brute_increment":          "true",
			"brute_max_length":         "9",
			"brute_min_length":         "7",
			"brute_predefined_charset": "UPPER, lower, Number [9]",
			"brute_use_custom_chars":   "false",
			"dict_rules_use_random":    "false",
			"hashes_use_upload":        "false",
			"hashmode":                 "100",
			"use_adv_options":          "false",
		},
	}

	var retJob1 common.Job
	retJob2 := common.Job{}
	retJob3 := new(common.Job)

	fmt.Printf("PRE Job: %#v\n\n", job)

	err = rpcClient.Call("Queue.JobManTest", common.RPCCall{Job: job}, &retJob1)
	if err != nil {
		t.Fatal(err)
	}

	err = rpcClient.Call("Queue.JobManTest", common.RPCCall{Job: job}, &retJob2)
	if err != nil {
		t.Fatal(err)
	}

	err = rpcClient.Call("Queue.JobManTest", common.RPCCall{Job: job}, &retJob3)
	if err != nil {
		t.Fatal(err)
	}

	err = rpcClient.Call("Queue.JobManTest", common.RPCCall{Job: job}, &job)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("POST Job: %#v\n\n", job)
	fmt.Printf("RetJob 1: %#v\n\n", retJob1)
	fmt.Printf("RetJob 2: %#v\n\n", retJob2)
	fmt.Printf("RetJob 3: %#v\n\n", retJob3)
}
