package resource

import (
	"github.com/pborman/uuid"
	"github.com/jmmcatee/cracklord/common"
	"net/rpc"
	"testing"
)

func TestRunFailure(t *testing.T) {
	// Create a queue & start the resource
	q := Queue{authToken: "FailureTest"}
	l := startRPCOnce("tcp", addr, &q)
	defer l.Close()

	// Add the failure tool
	tool := new(simpleFailerTooler)
	tool.SetUUID(uuid.New())
	q.tools = append(q.tools, tool)

	// Build the RPC client
	client, err := rpc.Dial("tcp", addr)
	if err != nil {
		t.Fatal("Error dailing RPC servers.", err)
	}

	// Build the job to send to the simpleFailureTask
	params := map[string]string{"failFunc": "Run"}
	job := common.NewJob(tool.UUID(), "Failure Test", "GoTestSuite", params)

	// Try and create the job... we should get a failure
	call := common.RPCCall{Auth: "FailureTest", Job: job}
	err = client.Call("Queue.AddTask", call, nil)
	if err == nil {
		t.Fatal("Failure task's error was not returned.")
	}
}

func TestPauseFailure(t *testing.T) {
	// Create a queue & start the resource
	q := Queue{authToken: "FailureTest"}
	l := startRPCOnce("tcp", addr, &q)
	defer l.Close()

	// Add the failure tool
	tool := new(simpleFailerTooler)
	tool.SetUUID(uuid.New())
	q.tools = append(q.tools, tool)

	// Build the RPC client
	client, err := rpc.Dial("tcp", addr)
	if err != nil {
		t.Fatal("Error dailing RPC servers.", err)
	}

	// Build the job to send to the simpleFailureTask
	params := map[string]string{"failFunc": "Pause"}
	job := common.NewJob(tool.UUID(), "Failure Test", "GoTestSuite", params)

	// Create the failure job
	call := common.RPCCall{Auth: "FailureTest", Job: job}
	err = client.Call("Queue.AddTask", call, nil)
	if err != nil {
		t.Fatal("Failure task failed on the wrong call.")
	}

	// Try to pause the job... we should get an error
	call = common.RPCCall{Auth: "FailureTest", Job: job}
	err = client.Call("Queue.TaskPause", call, nil)
	if err == nil {
		t.Fatal("Failure task's error was not returned.")
	}
}

func TestRunAfterPauseFailure(t *testing.T) {
	// Create a queue & start the resource
	q := Queue{authToken: "FailureTest"}
	l := startRPCOnce("tcp", addr, &q)
	defer l.Close()

	// Add the failure tool
	tool := new(simpleFailerTooler)
	tool.SetUUID(uuid.New())
	q.tools = append(q.tools, tool)

	// Build the RPC client
	client, err := rpc.Dial("tcp", addr)
	if err != nil {
		t.Fatal("Error dailing RPC servers.", err)
	}

	// Build the job to send to the simpleFailureTask
	params := map[string]string{"failFunc": "RunAfterPause"}
	job := common.NewJob(tool.UUID(), "Failure Test", "GoTestSuite", params)

	// Create the failure job
	call := common.RPCCall{Auth: "FailureTest", Job: job}
	err = client.Call("Queue.AddTask", call, nil)
	if err != nil {
		t.Fatal("Failure task failed on the wrong call.")
	}

	// Try to pause the job... we should get an error
	call = common.RPCCall{Auth: "FailureTest", Job: job}
	err = client.Call("Queue.TaskPause", call, nil)
	if err != nil {
		println("TEST::" + err.Error())
		t.Fatal("Failure task failed on the wrong call.")
	}

	call = common.RPCCall{Auth: "FailureTest", Job: job}
	err = client.Call("Queue.TaskRun", call, nil)
	if err == nil {
		t.Fatal("Failure task did not fail on Resume.")
	}
}

func TestQuitFailure(t *testing.T) {
	// Create a queue & start the resource
	q := Queue{authToken: "FailureTest"}
	l := startRPCOnce("tcp", addr, &q)
	defer l.Close()

	// Add the failure tool
	tool := new(simpleFailerTooler)
	tool.SetUUID(uuid.New())
	q.tools = append(q.tools, tool)

	// Build the RPC client
	client, err := rpc.Dial("tcp", addr)
	if err != nil {
		t.Fatal("Error dailing RPC servers.", err)
	}

	// Build the job to send to the simpleFailureTask
	params := map[string]string{"failFunc": "Quit"}
	job := common.NewJob(tool.UUID(), "Failure Test", "GoTestSuite", params)

	// Create the failure job
	call := common.RPCCall{Auth: "FailureTest", Job: job}
	err = client.Call("Queue.AddTask", call, nil)
	if err != nil {
		t.Fatal("Failure task failed on the wrong call.")
	}

	// Try to quit the job... we should get an error
	call = common.RPCCall{Auth: "FailureTest", Job: job}
	err = client.Call("Queue.TaskQuit", call, &job)
	if job.Error == "" {
		t.Fatal("Failure task's error was not returned.")
	}
}
