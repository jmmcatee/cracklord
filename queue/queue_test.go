package queue

import (
	"code.google.com/p/go-uuid/uuid"
	"fmt"
	"github.com/jmmcatee/cracklord/common"
	"github.com/jmmcatee/cracklord/resource"
	"testing"
	"time"
)

func dumpQueue(q Queue) {
	fmt.Printf("Queue Dump:\n")
	fmt.Printf("\tStatus:%s\n", q.status)
	fmt.Printf("\tResourcePool:\n")
	for _, v := range q.pool {
		fmt.Printf("\t\tResource: %v\n", v)
	}
	fmt.Printf("\tStack:\n")
	for _, j := range q.stack {
		fmt.Printf("\t\t%v\n", j)
	}
	fmt.Printf("\tStats: %v\n", q.stats)
	fmt.Printf("\tLock: %v\n", q.RWMutex)
	fmt.Printf("\tKeeperChannel: %v\n", q.qk)
}

func TestQueueCreate(t *testing.T) {
	// Build the main queue
	queue := NewQueue()

	// Build the resource
	res := resource.NewResourceQueue("QueueTest")
	tool := new(resource.SimpleTimerTooler)
	tool.SetUUID(uuid.New())
	res.AddTool(tool)

	closed := resource.StartResource("tcp", "localhost:4444", &res)

	err := queue.AddResource("tcp", "localhost:4444", "QueueTest")
	if err != nil {
		t.Fatal("TestQueueCreate:" + err.Error())
	}

	// Check that there is a tool called Simple Test Timer
	fail := false
	tools := queue.Tools()
	for _, v := range tools {
		if v.Name == "Simple Timer Tool" {
			fail = true
		}
	}

	if !fail {
		t.Fatal("Simple Timer Tool did not return correctly.")
	}

	for i, _ := range queue.pool {
		queue.pool[i].Client.Close()
	}

	<-closed
}

func TestQueueStop(t *testing.T) {
	// Build the main queue
	queue := NewQueue()

	// Build the resource
	res := resource.NewResourceQueue("QueueTest")
	tool := new(resource.SimpleTimerTooler)
	tool.SetUUID(uuid.New())
	res.AddTool(tool)

	closed := resource.StartResource("tcp", "localhost:4444", &res)

	err := queue.AddResource("tcp", "localhost:4444", "QueueTest")
	if err != nil {
		t.Fatal("TestQueueStop:" + err.Error())
	}

	jobs := queue.Quit()

	// Jobs should be empty
	if len(jobs) != 0 {
		t.Fatal("Queue returned jobs that shouldn't exist.")
	}

	<-closed
}

func TestQueueAddJob(t *testing.T) {
	// Build the main queue
	queue := NewQueue()

	// Build the resource
	res := resource.NewResourceQueue("QueueTest")
	tool := new(resource.SimpleTimerTooler)
	tool.SetUUID(uuid.New())
	res.AddTool(tool)

	closed := resource.StartResource("tcp", "localhost:4444", &res)

	err := queue.AddResource("tcp", "localhost:4444", "QueueTest")
	if err != nil {
		t.Fatal("TestQueueAddJob:" + err.Error())
	}

	// Build a simple jobs to run and add it to the queue
	params := map[string]string{"timer": "1"}
	j := common.NewJob(tool.UUID(), "Simple Timer Queue Test", "GoTestSuite", params)

	queue.AddJob(j)

	// Wait for the job to finish
	<-time.After(1 * time.Second)

	jobs := queue.Quit()

	// Check for done status
	for _, v := range jobs {
		if v.Status != common.STATUS_DONE {
			t.Fatal("Job was not finished and should have been.")
		}
	}

	<-closed
}

func TestQueueAddMultipleJob(t *testing.T) {
	// Build the main queue
	queue := NewQueue()
	KeeperDuration = 1 * time.Second

	// Build the resource
	res := resource.NewResourceQueue("QueueTest")
	tool := new(resource.SimpleTimerTooler)
	tool.SetUUID(uuid.New())
	res.AddTool(tool)

	closed := resource.StartResource("tcp", "localhost:4444", &res)

	err := queue.AddResource("tcp", "localhost:4444", "QueueTest")
	if err != nil {
		t.Fatal("TestQueueAddJob:" + err.Error())
	}

	// Build a simple jobs to run and add it to the queue
	params := map[string]string{"timer": "1"}
	j := common.NewJob(tool.UUID(), "Simple Timer Queue Test 1", "GoTestSuite", params)

	queue.AddJob(j)

	// Build another job that runs longer
	params = map[string]string{"timer": "2"}
	j = common.NewJob(tool.UUID(), "Simple Timer Queue Test 2", "GoTestSuite", params)

	queue.AddJob(j)

	// Build a third and final job that runs longer
	params = map[string]string{"timer": "3"}
	j = common.NewJob(tool.UUID(), "Simple Timer Queue Test 3", "GoTestSuite", params)

	queue.AddJob(j)

	// Wait for the job to finish
	<-time.After(10 * time.Second)

	jobs := queue.Quit()

	// Check for done status
	for _, v := range jobs {
		if v.Status != common.STATUS_DONE {
			t.Fatal("Job was not finished and should have been.")
		}
	}

	<-closed
}

func TestQueueDelayAddMultipleJob(t *testing.T) {
	// Build the main queue
	queue := NewQueue()
	KeeperDuration = 1 * time.Second

	// Build the resource
	res := resource.NewResourceQueue("QueueTest")
	tool := new(resource.SimpleTimerTooler)
	tool.SetUUID(uuid.New())
	res.AddTool(tool)

	closed := resource.StartResource("tcp", "localhost:4444", &res)

	err := queue.AddResource("tcp", "localhost:4444", "QueueTest")
	if err != nil {
		t.Fatal("TestQueueAddJob:" + err.Error())
	}

	// Build a simple jobs to run and add it to the queue
	params := map[string]string{"timer": "1"}
	j := common.NewJob(tool.UUID(), "Simple Timer Queue Test 1", "GoTestSuite", params)

	queue.AddJob(j)

	// Build another job that runs longer
	params = map[string]string{"timer": "2"}
	j = common.NewJob(tool.UUID(), "Simple Timer Queue Test 2", "GoTestSuite", params)

	queue.AddJob(j)

	// Build a third that runs longer
	params = map[string]string{"timer": "3"}
	j = common.NewJob(tool.UUID(), "Simple Timer Queue Test 3", "GoTestSuite", params)

	queue.AddJob(j)

	// Wait for the job to finish
	<-time.After(8 * time.Second)

	// Build a four and final job much delayed
	params = map[string]string{"timer": "2"}
	j = common.NewJob(tool.UUID(), "Simple Timer Queue Test 3", "GoTestSuite", params)

	queue.AddJob(j)

	<-time.After(3 * time.Second)

	jobs := queue.Quit()

	// Check for done status
	for _, v := range jobs {
		if v.Status != common.STATUS_DONE {
			t.Fatal("Job was not finished and should have been.")
		}
	}

	<-closed
}

func TestQueuePause(t *testing.T) {
	// Build the main queue
	queue := NewQueue()
	KeeperDuration = 1 * time.Second

	// Build the resource
	res := resource.NewResourceQueue("QueueTest")
	tool := new(resource.SimpleTimerTooler)
	tool.SetUUID(uuid.New())
	res.AddTool(tool)

	closed := resource.StartResource("tcp", "localhost:4444", &res)

	err := queue.AddResource("tcp", "localhost:4444", "QueueTest")
	if err != nil {
		t.Fatal("TestQueueAddJob:" + err.Error())
	}

	// Add two jobs with 2 second timers
	params := map[string]string{"timer": "2"}
	j := common.NewJob(tool.UUID(), "Simple Timer Queue Test 1", "GoTestSuite", params)

	queue.AddJob(j)

	params = map[string]string{"timer": "2"}
	j = common.NewJob(tool.UUID(), "Simple Timer Queue Test 2", "GoTestSuite", params)

	queue.AddJob(j)

	// Wait 1 second then pause
	<-time.After(1 * time.Second)

	// Pause the queue
	errs := queue.PauseQueue()
	if len(errs) != 0 {
		t.Fatal(errs)
	}
	if queue.status != STATUS_PAUSED {
		t.Fatal("Queue was not paused.")
	}

	// Add two more jobs
	params = map[string]string{"timer": "2"}
	j = common.NewJob(tool.UUID(), "Simple Timer Queue Test 3", "GoTestSuite", params)

	queue.AddJob(j)

	params = map[string]string{"timer": "2"}
	j = common.NewJob(tool.UUID(), "Simple Timer Queue Test 4", "GoTestSuite", params)

	queue.AddJob(j)

	// Restart the Queue
	queue.ResumeQueue()

	// Wait enough time for jobs to finish
	<-time.After(10 * time.Second)

	jobs := queue.Quit()

	// Check for done status
	for _, v := range jobs {
		if v.Status != common.STATUS_DONE {
			t.Fatal("Job was not finished and should have been.")
		}
	}

	<-closed
}
