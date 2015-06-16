package queue

import (
	"fmt"
	"github.com/jmmcatee/cracklord/src/common"
	"github.com/jmmcatee/cracklord/src/resource"
	"testing"
	"time"
)

func TestMeetQueueInterface(t *testing.T) {
	queue := NewQueue()

	takeInterface(&queue)
}

func takeInterface(a common.Queue) {
	a.Quit()
}

func dumpQueue(q Queue) {
	fmt.Printf("Queue Dump:\n")
	fmt.Printf("\tStatus:%s\n", q.status)
	fmt.Printf("\tResourcePool:\n")
	for i, v := range q.pool {
		fmt.Printf("\t\tResource(%s): %v\n", i, v)
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
	res.AddTool(tool)

	closed := resource.StartResource("localhost:4444", &res)

	err := queue.AddResource("localhost:4444", "test", "QueueTest")
	if err != nil {
		t.Fatal("TestQueueCreate:" + err.Error())
	}

	// Check that there is a tool called Simple Test Timer
	tools := queue.Tools()
	for _, v := range tools {
		if v.Name != "Simple Timer Tool" {
			t.Fatal("Simple Timer Tool did not return correctly.")
		}
	}

	types := queue.Types()
	for _, v := range types {
		if v != "Timer" {
			t.Fatal("Simple Timer Tool did not give the right type. Given " + v + " but expected Timer.")
		}
	}

	queue.Quit()

	<-closed
}

func TestQueueStop(t *testing.T) {
	// Build the main queue
	queue := NewQueue()

	// Build the resource
	res := resource.NewResourceQueue("QueueTest")
	tool := new(resource.SimpleTimerTooler)
	res.AddTool(tool)

	closed := resource.StartResource("localhost:4444", &res)

	err := queue.AddResource("localhost:4444", "test", "QueueTest")
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
	res.AddTool(tool)

	closed := resource.StartResource("localhost:4444", &res)

	err := queue.AddResource("localhost:4444", "test", "QueueTest")
	if err != nil {
		t.Fatal("TestQueueAddJob:" + err.Error())
	}

	// Get the Queue Tools so we can get the UUID
	var juuid string
	for _, v := range queue.Tools() {
		if v.Name == "Simple Timer Tool" {
			juuid = v.UUID
		}
	}

	// Build a simple jobs to run and add it to the queue
	params := map[string]string{"timer": "1"}
	j := common.NewJob(juuid, "Simple Timer Queue Test", "GoTestSuite", params)

	err = queue.AddJob(j)
	if err != nil {
		t.Fatal("Error adding Job: " + err.Error())
	}

	// Wait for the job to finish
	<-time.After(2 * time.Second)

	println("MADE IT")
	jobs := queue.Quit()
	println("MADE IT 2")

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
	res.AddTool(tool)

	closed := resource.StartResource("localhost:4444", &res)

	err := queue.AddResource("localhost:4444", "test", "QueueTest")
	if err != nil {
		t.Fatal("TestQueueAddJob:" + err.Error())
	}

	// Get the Queue Tools so we can get the UUID
	var juuid string
	for _, v := range queue.Tools() {
		if v.Name == "Simple Timer Tool" {
			juuid = v.UUID
		}
	}

	// Build a simple jobs to run and add it to the queue
	params := map[string]string{"timer": "1"}
	j := common.NewJob(juuid, "Simple Timer Queue Test 1", "GoTestSuite", params)

	err = queue.AddJob(j)
	if err != nil {
		t.Fatal("Error adding Job: " + err.Error())
	}

	// Build another job that runs longer
	params = map[string]string{"timer": "2"}
	j = common.NewJob(juuid, "Simple Timer Queue Test 2", "GoTestSuite", params)

	err = queue.AddJob(j)
	if err != nil {
		t.Fatal("Error adding Job: " + err.Error())
	}

	// Build a third and final job that runs longer
	params = map[string]string{"timer": "3"}
	j = common.NewJob(juuid, "Simple Timer Queue Test 3", "GoTestSuite", params)

	err = queue.AddJob(j)
	if err != nil {
		t.Fatal("Error adding Job: " + err.Error())
	}

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
	res.AddTool(tool)

	closed := resource.StartResource("localhost:4444", &res)

	err := queue.AddResource("localhost:4444", "test", "QueueTest")
	if err != nil {
		t.Fatal("TestQueueAddJob:" + err.Error())
	}

	// Get the Queue Tools so we can get the UUID
	var juuid string
	for _, v := range queue.Tools() {
		if v.Name == "Simple Timer Tool" {
			juuid = v.UUID
		}
	}

	// Build a simple jobs to run and add it to the queue
	params := map[string]string{"timer": "1"}
	j := common.NewJob(juuid, "Simple Timer Queue Test 1", "GoTestSuite", params)

	err = queue.AddJob(j)
	if err != nil {
		t.Fatal("Error adding Job: " + err.Error())
	}

	// Build another job that runs longer
	params = map[string]string{"timer": "2"}
	j = common.NewJob(juuid, "Simple Timer Queue Test 2", "GoTestSuite", params)

	err = queue.AddJob(j)
	if err != nil {
		t.Fatal("Error adding Job: " + err.Error())
	}

	// Build a third that runs longer
	params = map[string]string{"timer": "3"}
	j = common.NewJob(juuid, "Simple Timer Queue Test 3", "GoTestSuite", params)

	err = queue.AddJob(j)
	if err != nil {
		t.Fatal("Error adding Job: " + err.Error())
	}

	// Wait for the job to finish
	<-time.After(8 * time.Second)

	// Build a four and final job much delayed
	params = map[string]string{"timer": "2"}
	j = common.NewJob(juuid, "Simple Timer Queue Test 3", "GoTestSuite", params)

	err = queue.AddJob(j)
	if err != nil {
		t.Fatal("Error adding Job: " + err.Error())
	}

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
	res.AddTool(tool)

	closed := resource.StartResource("localhost:4444", &res)

	err := queue.AddResource("localhost:4444", "test", "QueueTest")
	if err != nil {
		t.Fatal("TestQueueAddJob:" + err.Error())
	}

	// Get the Queue Tools so we can get the UUID
	var juuid string
	for _, v := range queue.Tools() {
		if v.Name == "Simple Timer Tool" {
			juuid = v.UUID
		}
	}

	// Add two jobs with 2 second timers
	params := map[string]string{"timer": "2"}
	j := common.NewJob(juuid, "Simple Timer Queue Test 1", "GoTestSuite", params)

	err = queue.AddJob(j)
	if err != nil {
		t.Fatal("Error adding Job: " + err.Error())
	}

	params = map[string]string{"timer": "2"}
	j = common.NewJob(juuid, "Simple Timer Queue Test 2", "GoTestSuite", params)

	err = queue.AddJob(j)
	if err != nil {
		t.Fatal("Error adding Job: " + err.Error())
	}

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
	j = common.NewJob(juuid, "Simple Timer Queue Test 3", "GoTestSuite", params)

	err = queue.AddJob(j)
	if err != nil {
		t.Fatal("Error adding Job: " + err.Error())
	}

	params = map[string]string{"timer": "2"}
	j = common.NewJob(juuid, "Simple Timer Queue Test 4", "GoTestSuite", params)

	err = queue.AddJob(j)
	if err != nil {
		t.Fatal("Error adding Job: " + err.Error())
	}

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

func TestJobPause(t *testing.T) {
	// Build the main queue
	queue := NewQueue()
	KeeperDuration = 1 * time.Second

	// Build the resource
	res := resource.NewResourceQueue("QueueTest")
	tool := new(resource.SimpleTimerTooler)
	res.AddTool(tool)

	closed := resource.StartResource("localhost:4444", &res)

	err := queue.AddResource("localhost:4444", "test", "QueueTest")
	if err != nil {
		t.Fatal("TestQueueAddJob:" + err.Error())
	}

	// Get the Queue Tools so we can get the UUID
	var juuid string
	for _, v := range queue.Tools() {
		if v.Name == "Simple Timer Tool" {
			juuid = v.UUID
		}
	}

	// Add two jobs with 2 second timers
	params := map[string]string{"timer": "4"}
	j1 := common.NewJob(juuid, "Simple Timer Queue Test 1", "GoTestSuite", params)

	err = queue.AddJob(j1)
	if err != nil {
		t.Fatal("Error adding Job: " + err.Error())
	}

	params = map[string]string{"timer": "2"}
	j2 := common.NewJob(juuid, "Simple Timer Queue Test 2", "GoTestSuite", params)

	err = queue.AddJob(j2)
	if err != nil {
		t.Fatal("Error adding Job: " + err.Error())
	}

	// Now pause the first job we added so the second finished first
	err = queue.PauseJob(j1.UUID)
	if err != nil {
		t.Fatal("Pausing job failed: " + err.Error())
	}

	// The first jobs should now be paused so lets pull all statuses and check
	jobs := queue.AllJobs()
	for _, v := range jobs {
		if v.UUID == j1.UUID {
			if v.Status != common.STATUS_PAUSED {
				t.Fatal("Job was not paused and should have been.")
			}
		}
	}

	// Wait the total runtime to make sure everything eventually finished
	<-time.After(8 * time.Second)

	jobs = queue.Quit()
	for _, v := range jobs {
		if v.Status != common.STATUS_DONE {
			t.Fatal("Not all jobs were finished.")
		}
	}

	<-closed
}

func TestJobQuit(t *testing.T) {
	// Build the main queue
	queue := NewQueue()
	KeeperDuration = 1 * time.Second

	// Build the resource
	res := resource.NewResourceQueue("QueueTest")
	tool := new(resource.SimpleTimerTooler)
	res.AddTool(tool)

	closed := resource.StartResource("localhost:4444", &res)

	err := queue.AddResource("localhost:4444", "test", "QueueTest")
	if err != nil {
		t.Fatal("TestQueueAddJob:" + err.Error())
	}

	// Get the Queue Tools so we can get the UUID
	var juuid string
	for _, v := range queue.Tools() {
		if v.Name == "Simple Timer Tool" {
			juuid = v.UUID
		}
	}

	// Add two jobs with 2 second timers
	params := map[string]string{"timer": "4"}
	j1 := common.NewJob(juuid, "Simple Timer Queue Test 1", "GoTestSuite", params)

	err = queue.AddJob(j1)
	if err != nil {
		t.Fatal("Error adding Job: " + err.Error())
	}

	params = map[string]string{"timer": "2"}
	j2 := common.NewJob(juuid, "Simple Timer Queue Test 2", "GoTestSuite", params)

	err = queue.AddJob(j2)
	if err != nil {
		t.Fatal("Error adding Job: " + err.Error())
	}

	// Now pause the first job we added so the second finished first
	err = queue.QuitJob(j1.UUID)
	if err != nil {
		t.Fatal("Quiting job failed: " + err.Error())
	}

	// The first jobs should now be paused so lets pull all statuses and check
	jobs := queue.AllJobs()
	for _, v := range jobs {
		if v.UUID == j1.UUID {
			if v.Status != common.STATUS_QUIT {
				t.Fatal("Job was not quit and should have been.")
			}
		}
	}

	// Wait the total runtime to make sure everything eventually finished
	<-time.After(4 * time.Second)

	jobs = queue.Quit()
	for _, v := range jobs {
		if v.Status != common.STATUS_DONE && v.Status != common.STATUS_QUIT {
			t.Fatal("Not all jobs were finished. Status:" + v.Status)
		}
	}

	<-closed
}

func TestMultiResourceMultiJobs1(t *testing.T) {
	// Build the main queue
	queue := NewQueue()
	KeeperDuration = 100 * time.Millisecond

	// Build the first resource
	res1 := resource.NewResourceQueue("Resource 1")
	tool := new(resource.SimpleTimerTooler)
	res1.AddTool(tool)

	// Start the first resource
	close1 := resource.StartResource("localhost:4441", &res1)

	// Add the first resource
	err := queue.AddResource("localhost:4441", "name", "Resource 1")
	if err != nil {
		t.Fatal("Error adding resource: " + err.Error())
	}

	// Build the second resource
	res2 := resource.NewResourceQueue("Resource 2")
	tool = new(resource.SimpleTimerTooler)
	res2.AddTool(tool)

	// Start the second resource
	close2 := resource.StartResource("localhost:4442", &res2)

	// Add the second resource
	err = queue.AddResource("localhost:4442", "name", "Resource 2")
	if err != nil {
		t.Fatal("Error adding resource: " + err.Error())
	}

	// Build several jobs to add
	param1 := map[string]string{"timer": "1"}
	param2 := map[string]string{"timer": "2"}
	param3 := map[string]string{"timer": "3"}

	var tu1 string
	for _, v := range queue.Tools() {
		tu1 = v.UUID
	}

	j1 := common.NewJob(tu1, "Simple Timer Queue Test 1", "GoTestSuite", param1)
	j2 := common.NewJob(tu1, "Simple Timer Queue Test 2", "GoTestSuite", param3)
	j3 := common.NewJob(tu1, "Simple Timer Queue Test 3", "GoTestSuite", param3)
	j4 := common.NewJob(tu1, "Simple Timer Queue Test 4", "GoTestSuite", param2)
	j5 := common.NewJob(tu1, "Simple Timer Queue Test 5", "GoTestSuite", param1)

	err = queue.AddJob(j1)
	if err != nil {
		t.Fatal("Job was not added successfully: " + err.Error())
	}
	err = queue.AddJob(j2)
	if err != nil {
		t.Fatal("Job was not added successfully: " + err.Error())
	}
	err = queue.AddJob(j3)
	if err != nil {
		t.Fatal("Job was not added successfully: " + err.Error())
	}
	err = queue.AddJob(j4)
	if err != nil {
		t.Fatal("Job was not added successfully: " + err.Error())
	}
	err = queue.AddJob(j5)
	if err != nil {
		t.Fatal("Job was not added successfully: " + err.Error())
	}

	for {
		running := false

		select {
		case <-time.After(1 * time.Second):
			// Check if all jobs are done
			for _, v := range queue.AllJobs() {
				if v.Status != common.STATUS_DONE {
					running = true
				}
			}

		}

		if !running {
			break
		}
	}

	queue.Quit()

	<-close1
	<-close2
}

func TestMultiResourceMultiJobs2(t *testing.T) {
	// Build the main queue
	queue := NewQueue()
	KeeperDuration = 100 * time.Millisecond

	// Build the first resource
	res1 := resource.NewResourceQueue("Resource 1")
	tool := new(resource.SimpleTimerTooler)
	res1.AddTool(tool)

	// Start the first resource
	close1 := resource.StartResource("localhost:4441", &res1)

	// Add the first resource
	err := queue.AddResource("localhost:4441", "name", "Resource 1")
	if err != nil {
		t.Fatal("Error adding resource: " + err.Error())
	}

	// Build the second resource
	res2 := resource.NewResourceQueue("Resource 2")
	tool = new(resource.SimpleTimerTooler)
	res2.AddTool(tool)

	// Start the second resource
	close2 := resource.StartResource("localhost:4442", &res2)

	// Add the second resource
	err = queue.AddResource("localhost:4442", "name", "Resource 2")
	if err != nil {
		t.Fatal("Error adding resource: " + err.Error())
	}

	// Build the second resource
	res3 := resource.NewResourceQueue("Resource 3")
	tool3 := new(resource.SimpleTimerTooler)
	res3.AddTool(tool3)

	// Start the second resource
	close3 := resource.StartResource("localhost:4443", &res3)

	// Add the second resource
	err = queue.AddResource("localhost:4443", "name", "Resource 3")
	if err != nil {
		t.Fatal("Error adding resource: " + err.Error())
	}

	// Build several jobs to add
	param := map[string]string{"timer": "3"}

	var tu1 string
	for _, v := range queue.Tools() {
		tu1 = v.UUID
	}

	j1 := common.NewJob(tu1, "Simple Timer Queue Test 1", "GoTestSuite", param)
	j2 := common.NewJob(tu1, "Simple Timer Queue Test 2", "GoTestSuite", param)
	j3 := common.NewJob(tu1, "Simple Timer Queue Test 3", "GoTestSuite", param)
	j4 := common.NewJob(tu1, "Simple Timer Queue Test 4", "GoTestSuite", param)
	j5 := common.NewJob(tu1, "Simple Timer Queue Test 5", "GoTestSuite", param)
	j6 := common.NewJob(tu1, "Simple Timer Queue Test 6", "GoTestSuite", param)

	err = queue.AddJob(j1)
	if err != nil {
		t.Fatal("Job was not added successfully: " + err.Error())
	}
	err = queue.AddJob(j2)
	if err != nil {
		t.Fatal("Job was not added successfully: " + err.Error())
	}
	err = queue.AddJob(j3)
	if err != nil {
		t.Fatal("Job was not added successfully: " + err.Error())
	}
	err = queue.AddJob(j4)
	if err != nil {
		t.Fatal("Job was not added successfully: " + err.Error())
	}
	err = queue.AddJob(j5)
	if err != nil {
		t.Fatal("Job was not added successfully: " + err.Error())
	}
	err = queue.AddJob(j6)
	if err != nil {
		t.Fatal("Job was not added successfully: " + err.Error())
	}

	for {
		running := false

		select {
		case <-time.After(1 * time.Second):
			// Check if all jobs are done
			for _, v := range queue.AllJobs() {
				if v.Status != common.STATUS_DONE {
					running = true
				}
			}

		}

		if !running {
			break
		}
	}

	queue.Quit()

	<-close1
	<-close2
	<-close3
}
