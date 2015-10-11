package resource

import (
	"github.com/pborman/uuid"
	"github.com/jmmcatee/cracklord/common"
	"log"
	"net/rpc"
	"testing"
	"time"
)

var addr = "localhost:4000"

func TestCreateResource(t *testing.T) {
	// Create the RPC resource
	q := Queue{authToken: "ResourceTest"}
	l := startRPCOnce("tcp", addr, &q)
	defer l.Close()

	// Just connect to the RPC service and check for errors
	client, err := rpc.Dial("tcp", addr)
	if err != nil {
		t.Fatal("dialing:", err)
	}
	defer client.Close()

	// Do a call but ignore the out so that the RPC connection closes properly
	rpccall := common.RPCCall{Auth: "ResourceTest"}
	j := []common.Job{}
	err = client.Call("Queue.AllTaskStatus", rpccall, &j)
	if err != nil {
		log.Fatal("Failed call: ", err)
	}
}

func TestResourceAuth(t *testing.T) {
	// Create the RPC Server
	q := Queue{authToken: "ResourceTest"}
	l := startRPCOnce("tcp", addr, &q)
	defer l.Close()

	// Just connect to the RPC service and check for errors
	client, err := rpc.Dial("tcp", addr)
	if err != nil {
		t.Fatal("dialing:", err)
	}
	defer client.Close()

	// Do a call to get tools
	rpccall := common.RPCCall{Auth: "ResourceTest"}
	err = client.Call("Queue.AllTaskStatus", rpccall, nil)
	if err != nil {
		log.Fatal("Failed call: ", err)
	}

	// Now make a call and look for the error
	badCall := common.RPCCall{Auth: "WRONG"}

	// Resource hardware
	err = client.Call("Queue.ResourceHardware", badCall, nil)
	if err == nil || err.Error() != ERROR_AUTH {
		t.Fatal("Error was not returned for bad authentication.")
	}

	// AddTask
	err = client.Call("Queue.AddTask", badCall, nil)
	if err == nil || err.Error() != ERROR_AUTH {
		t.Fatal("Error was not returned for bad authentication.")
	}

	// Task Status
	err = client.Call("Queue.TaskStatus", badCall, nil)
	if err == nil || err.Error() != ERROR_AUTH {
		t.Fatal("Error was not returned for bad authentication.")
	}

	// Task Pause
	err = client.Call("Queue.TaskPause", badCall, nil)
	if err == nil || err.Error() != ERROR_AUTH {
		t.Fatal("Error was not returned for bad authentication.")
	}

	// Task Run
	err = client.Call("Queue.TaskRun", badCall, nil)
	if err == nil || err.Error() != ERROR_AUTH {
		t.Fatal("Error was not returned for bad authentication.")
	}

	// Task Quit
	err = client.Call("Queue.TaskQuit", badCall, nil)
	if err == nil || err.Error() != ERROR_AUTH {
		t.Fatal("Error was not returned for bad authentication.")
	}

	// AllTaskStatus
	err = client.Call("Queue.AllTaskStatus", badCall, nil)
	if err == nil || err.Error() != ERROR_AUTH {
		t.Fatal("Error was not returned for bad authentication.")
	}
}

func TestListResourceTools(t *testing.T) {
	// Build the queue to pass to the RPC server
	q := Queue{authToken: "ResourceTest"}
	b := new(SimpleTimerTooler)
	b.SetUUID(uuid.New())
	q.tools = append(q.tools, b)

	// Create the RPC resource
	l := startRPCOnce("tcp", addr, &q)
	defer l.Close()

	// Just connect to the RPC service and check for errors
	client, err := rpc.Dial("tcp", addr)
	if err != nil {
		t.Fatal("dialing:", err)
	}
	defer client.Close()

	rpccall := common.RPCCall{Auth: "ResourceTest"}

	var tools []common.Tool
	client.Call("Queue.ResourceTools", rpccall, &tools)

	for _, v := range tools {
		if v.Name != b.Name() {
			t.Errorf("Returned Name did not match provided name: %s vs %s", v.Name, b.Name())
		}
		if v.Type != b.Type() {
			t.Errorf("Returned Type did not match provided name: %s vs %s", v.Type, b.Type())
		}
		if v.Version != b.Version() {
			t.Errorf("Returned name did not match provided name: %s vs %s", v.Version, b.Version())
		}
		if v.Parameters != b.Parameters() {
			t.Errorf("Returned name did not match provided name: %s vs %s", v.Parameters, b.Parameters())
		}
		if v.Requirements != b.Requirements() {
			t.Errorf("Returned name did not match provided name: %s vs %s", v.Requirements, b.Requirements())
		}
	}
}

func TestSimpleStartTask(t *testing.T) {
	// Build the Queue with the SimpleTool timer
	q := Queue{authToken: "ResourceTest"}

	// Add the tool
	st := new(SimpleTimerTooler)
	st.SetUUID(uuid.New())
	q.tools = append(q.tools, st)

	// Create the RPC server and bind the Queue
	listen := startRPCOnce("tcp", addr, &q)
	defer listen.Close()

	// Connect to the RPC server
	client, err := rpc.Dial("tcp", addr)
	if err != nil {
		t.Fatal("Error dailing RPC server.", err)
	}
	defer client.Close()

	// Setup the Job information to start the service
	params := map[string]string{"timer": "1"}
	j := common.NewJob(st.UUID(), "Testing Job", "GoTestSuite", params)

	// Create RPC call for starting a job
	startJob := common.RPCCall{Auth: "ResourceTest", Job: j}

	// Make call
	err = client.Call("Queue.AddTask", startJob, &j)
	if err != nil {
		t.Fatal("Error starting SimpleTimer task.", err)
	}

	// Wait 2 seconds so the timer finishes
	<-time.After(2 * time.Second)

	// Get the job status and check for finished status
	err = client.Call("Queue.TaskStatus", startJob, &j)
	if err != nil {
		t.Fatal("Error getting simpleTimer status.", err)
	}

	if j.Status != common.STATUS_DONE {
		t.Errorf("Status was not done, which was expected. Status:%s", j.Status)
	}
}

func TestSimplePauseTask(t *testing.T) {
	// Build the Queue with the simpleTool timer
	q := Queue{authToken: "ResourceTest"}

	// Add the tool
	st := new(SimpleTimerTooler)
	st.SetUUID(uuid.New())
	q.tools = append(q.tools, st)

	// Create the RPC server and bind the queue
	listen := startRPCOnce("tcp", addr, &q)
	defer listen.Close()

	// Connect client to RPC server
	client, err := rpc.Dial("tcp", addr)
	if err != nil {
		t.Fatal("Error dailing RPC server.", err)
	}
	defer client.Close()

	// Setup the Job information to start the service
	params := map[string]string{"timer": "5"}
	j := common.NewJob(st.UUID(), "Testing Job", "GoTestSuite", params)

	// Create the RPC call for starting a job
	startJob := common.RPCCall{Auth: "ResourceTest", Job: j}

	// Make the call to create the task
	err = client.Call("Queue.AddTask", startJob, &j)
	if err != nil {
		t.Fatal("Error starting simpleTimer task.", err)
	}

	// Wait 2 seconds before pausing
	<-time.After(2 * time.Second)

	// Pause the job
	pauseJob := common.RPCCall{Auth: "ResourceTest", Job: j}
	err = client.Call("Queue.TaskPause", pauseJob, &j)
	if err != nil {
		t.Fatal("Error pausing simpleTimer status.", err)
	}

	// Get the status of the job
	statusJob := common.RPCCall{Auth: "ResourceTest", Job: j}
	err = client.Call("Queue.TaskStatus", statusJob, &j)
	if err != nil {
		t.Fatal("Error getting simpleTimer status.", err)
	}

	if j.Status != common.STATUS_PAUSED {
		t.Errorf("Status was not paused, which was expected. Status:%s", j.Status)
	}
}

func TestSimpleRunTask(t *testing.T) {
	// Build the Queue with the simpleTool timer
	q := Queue{authToken: "ResourceTest"}

	// Add the tool
	st := new(SimpleTimerTooler)
	q.tools = append(q.tools, st)

	// Create the RPC server and bind the queue
	listen := startRPCOnce("tcp", addr, &q)
	defer listen.Close()

	// Connect client to RPC server
	client, err := rpc.Dial("tcp", addr)
	if err != nil {
		t.Fatal("Error dailing RPC server.", err)
	}
	defer client.Close()

	// Setup the Job information to start the service
	params := map[string]string{"timer": "5"}
	j := common.NewJob(st.UUID(), "Testing Job", "GoTestSuite", params)

	// Create the RPC call for starting a job
	startJob := common.RPCCall{Auth: "ResourceTest", Job: j}

	// Make the call to create the task
	err = client.Call("Queue.AddTask", startJob, &j)
	if err != nil {
		t.Fatal("Error starting simpleTimer task.", err)
	}

	// Wait 2 seconds before pausing
	<-time.After(2 * time.Second)

	// Pause the job
	pauseJob := common.RPCCall{Auth: "ResourceTest", Job: j}
	err = client.Call("Queue.TaskPause", pauseJob, &j)
	if err != nil {
		t.Fatal("Error pausing simpleTimer status.", err)
	}

	// Get the status of the job
	statusJob := common.RPCCall{Auth: "ResourceTest", Job: j}
	err = client.Call("Queue.TaskStatus", statusJob, &j)
	if err != nil {
		t.Fatal("Error getting simpleTimer status.", err)
	}

	if j.Status != common.STATUS_PAUSED {
		t.Errorf("Status was not paused, which was expected. Status:%s", j.Status)
	}

	// Restart the job and wait to see if it finishes
	runJob := common.RPCCall{Auth: "ResourceTest", Job: j}
	err = client.Call("Queue.TaskRun", runJob, &j)
	if err != nil {
		t.Fatal("Error resuming simpleTimer task.", err)
	}

	if j.Status != common.STATUS_RUNNING {
		t.Errorf("Status was not running, which was expected. Status%s", j.Status)
	}

	// Wait plenty of time for the task to finish
	<-time.After(5 * time.Second)

	// Get the final status
	finalJob := common.RPCCall{Auth: "ResourceTest", Job: j}
	err = client.Call("Queue.TaskStatus", finalJob, &j)
	if err != nil {
		t.Fatal("Error getting simpleTimer status.", err)
	}

	if j.Status != common.STATUS_DONE {
		t.Errorf("Status was not done, which was expected. Status:%s", j.Status)
	}
}

func TestSimpleQuitTask(t *testing.T) {
	// Build the Queue with the SimpleTool timer
	q := Queue{authToken: "ResourceTest"}

	// Add the tool
	st := new(SimpleTimerTooler)
	q.tools = append(q.tools, st)

	// Create the RPC server and bind the Queue
	listen := startRPCOnce("tcp", addr, &q)
	defer listen.Close()

	// Connect to the RPC server
	client, err := rpc.Dial("tcp", addr)
	if err != nil {
		t.Fatal("Error dailing RPC server.", err)
	}
	defer client.Close()

	// Setup the Job information to start the service
	params := map[string]string{"timer": "10"}
	j := common.NewJob(st.UUID(), "Testing Job", "GoTestSuite", params)

	// Create RPC call for starting a job
	startJob := common.RPCCall{Auth: "ResourceTest", Job: j}

	// Make call
	err = client.Call("Queue.AddTask", startJob, &j)
	if err != nil {
		t.Fatal("Error starting SimpleTimer task.", err)
	}

	// Wait 2 seconds so the timer finishes
	<-time.After(2 * time.Second)

	// Get the job status and check for finished status
	err = client.Call("Queue.TaskQuit", startJob, &j)
	if err != nil {
		t.Fatal("Error getting simpleTimer status.", err)
	}

	if j.Status != common.STATUS_QUIT {
		t.Errorf("Status was not quit, which was expected. Status:%s", j.Status)
	}
}

func TestResourceHardware(t *testing.T) {
	// Build the Queue with the SimpleTool timer
	hw := map[string]bool{common.RES_CPU: true, common.RES_GPU: true}
	q := Queue{authToken: "ResourceTest", hardware: hw}

	// Create the RPC server and bind the Queue
	listen := startRPCOnce("tcp", addr, &q)
	defer listen.Close()

	// Connect to the RPC server
	client, err := rpc.Dial("tcp", addr)
	if err != nil {
		t.Fatal("Error dailing RPC server.", err)
	}
	defer client.Close()

	// Create RPC call for starting a job
	startJob := common.RPCCall{Auth: "ResourceTest"}

	// Make call
	ahw := make(map[string]bool)
	err = client.Call("Queue.ResourceHardware", startJob, &ahw)
	if err != nil {
		t.Fatal("Error getting resource hardware.", err)
	}

	if ahw["cpu"] != true || ahw["gpu"] != true {
		t.Fatal("Hardware returned did not match what was put in.", ahw)
	}
}

func TestToolDoesNotExist(t *testing.T) {
	// Build the Queue with the SimpleTool timer
	q := Queue{authToken: "ResourceTest"}

	// Add the tool
	st := new(SimpleTimerTooler)
	st.SetUUID(uuid.New())
	q.tools = append(q.tools, st)

	// Create the RPC server and bind the Queue
	listen := startRPCOnce("tcp", addr, &q)
	defer listen.Close()

	// Connect to the RPC server
	client, err := rpc.Dial("tcp", addr)
	if err != nil {
		t.Fatal("Error dailing RPC server.", err)
	}
	defer client.Close()

	// Setup the Job information to start the service
	params := map[string]string{"timer": "1"}
	j := common.NewJob(uuid.New(), "Testing Job", "GoTestSuite", params)

	// Create RPC call for starting a job
	startJob := common.RPCCall{Auth: "ResourceTest", Job: j}

	// Make call and expect an error of bad job
	err = client.Call("Queue.AddTask", startJob, &j)
	if err == nil || err.Error() != ERROR_NO_TOOL {
		t.Fatal("No tool error was not returned.", err)
	}
}

func TestMultiToolStatus(t *testing.T) {
	// Build the Queue with the SimpleTool timer
	q := Queue{authToken: "ResourceTest"}

	// Add the tool
	st := new(SimpleTimerTooler)
	st.SetUUID(uuid.New())
	q.tools = append(q.tools, st)

	// Create the RPC server and bind the Queue
	listen := startRPCOnce("tcp", addr, &q)
	defer listen.Close()

	// Connect to the RPC server
	client, err := rpc.Dial("tcp", addr)
	if err != nil {
		t.Fatal("Error dailing RPC server.", err)
	}
	defer client.Close()

	// Setup the Job information to start the service
	params := map[string]string{"timer": "20"}
	j := common.NewJob(st.UUID(), "Testing Job", "GoTestSuite", params)

	// Create RPC call for starting a job
	startJob := common.RPCCall{Auth: "ResourceTest", Job: j}

	// Add our first job
	err = client.Call("Queue.AddTask", startJob, &j)
	if err != nil {
		t.Fatal("Error starting SimpleTimer task.", err)
	}

	// Add our second job
	j = common.NewJob(st.UUID(), "Testing Job", "GoTestSuite", params)
	err = client.Call("Queue.AddTask", startJob, &j)
	if err != nil {
		t.Fatal("Error starting SimpleTimer task.", err)
	}

	// Get the status of both and check that the Job UUIDs aren't the same
	getStatus := common.RPCCall{Auth: "ResourceTest"}
	jobs := []common.Job{}
	err = client.Call("Queue.AllTaskStatus", getStatus, &jobs)
	if err != nil {
		t.Fatal("Failure getting multiple jobs status.", err)
	}

	var firstUUID string
	for i, v := range jobs {
		if i == 0 {
			firstUUID = v.UUID
		} else if firstUUID == v.UUID {
			t.Fatal("Both jobs have the same UUID.")
		}
	}

}
