package resource

import (
	"errors"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/jmmcatee/cracklord/common"
	"github.com/pborman/uuid"
)

// TODO: Add function for adding tools and assign a UUID

const (
	ERROR_AUTH    = "Call to resource did not have the proper authentication token."
	ERROR_NO_TOOL = "Tool specified does not exit."
)

type Queue struct {
	stack map[string]common.Tasker
	tools []common.Tooler
	sync.RWMutex
	hardware map[string]bool
}

func NewResourceQueue() Queue {
	return Queue{
		stack:    map[string]common.Tasker{},
		tools:    []common.Tooler{},
		hardware: map[string]bool{},
	}
}

func (q *Queue) AddTool(tooler common.Tooler) {
	// Add the hardware used by the tool
	q.hardware[tooler.Requirements()] = true

	tooler.SetUUID(uuid.New())
	q.tools = append(q.tools, tooler)
	log.WithFields(log.Fields{
		"toolid":  tooler.UUID(),
		"name":    tooler.Name(),
		"version": tooler.Version(),
	}).Debug("Tool added")
}

// Task RPC functions
func (q *Queue) Ping(ping int, pong *int) error {
	calc := ping * ping

	*pong = calc

	log.WithFields(log.Fields{
		"ping": ping,
		"pong": calc,
	}).Debug("RPC ping called")

	return nil
}

func (q *Queue) ResourceHardware(rpc common.RPCCall, hw *map[string]bool) error {
	q.RLock()
	defer q.RUnlock()

	*hw = q.hardware

	return nil
}

func (q *Queue) AddTask(rpc common.RPCCall, rj *common.Job) error {
	log.WithFields(common.LogJob(rpc.Job)).Info("Job added")

	// Add a defered catch for panic from within the tools
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("Recovered from Panic in Resource.AddTask: %v", err)
		}
	}()

	// variable to hold the tasker
	var tasker common.Tasker
	var err error
	// loop through common.Toolers for matching tool
	q.Lock()
	defer q.Unlock()
	for i, _ := range q.tools {
		if q.tools[i].UUID() == rpc.Job.ToolUUID {
			tasker, err = q.tools[i].NewTask(rpc.Job)
			if err != nil {
				return err
			}
		}
	}

	// Check if no tool was found and return error
	if tasker == nil {
		log.Warn("An error occured, we could not find the tool requested")
		return errors.New(ERROR_NO_TOOL)
	}
	log.WithFields(log.Fields{
		"task": rpc.Job.UUID,
	}).Debug("Tasker created")

	// Looks good so lets add to the stack
	if q.stack == nil {
		q.stack = make(map[string]common.Tasker)
	}

	q.stack[rpc.Job.UUID] = tasker

	// Everything should be paused by the control queue so start this job
	err = q.stack[rpc.Job.UUID].Run()
	if err != nil {
		log.Debug("Error starting task on resource")

		rpc.Job.Status = common.STATUS_FAILED

		common.CopyJob(rpc.Job, rj)

		return errors.New("Error starting task on the resource: " + err.Error())
	}

	// Grab the status and return that job to the control queue
	common.CopyJob(q.stack[rpc.Job.UUID].Status(), rj)

	return nil
}

func (q *Queue) TaskStatus(rpc common.RPCCall, j *common.Job) error {
	log.WithField("task", rpc.Job.UUID).Debug("Attempting to gather task status")

	// Add a defered catch for panic from within the tools
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("Recovered from Panic in Resource.TaskStatus: %v", err)
		}
	}()

	// Grab the task specified by the UUID and return its status
	q.Lock()
	defer q.Unlock()
	_, ok := q.stack[rpc.Job.UUID]

	// Check for a bad UUID
	if !ok {
		log.WithField("task", rpc.Job.UUID).Error("Task with UUID provided does not exist.")
		return errors.New("Task with UUID provided does not exist.")
	}

	common.CopyJob(q.stack[rpc.Job.UUID].Status(), j)

	return nil
}

func (q *Queue) TaskPause(rpc common.RPCCall, j *common.Job) error {
	log.WithField("task", rpc.Job.UUID).Debug("Attempting to pause task")

	// Add a defered catch for panic from within the tools
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("Recovered from Panic in Resource.TaskPause: %v", err)
		}
	}()

	// Grab the task specified by the UUID
	q.Lock()
	defer q.Unlock()
	_, ok := q.stack[rpc.Job.UUID]

	// Check for a bad UUID
	if !ok {
		log.WithField("task", rpc.Job.UUID).Debug("Task with UUID provided does not exist.")
		return errors.New("Task with UUID provided does not exist.")
	}

	// Pause the task
	err := q.stack[rpc.Job.UUID].Pause()
	if err != nil {
		// return the error but quit the job with status Failed
		// This is a definied behavior that we will not for all tools
		q.stack[rpc.Job.UUID].Quit()
		return err
	}

	common.CopyJob(q.stack[rpc.Job.UUID].Status(), j)

	log.WithField("task", j.UUID).Debug("Task paused successfully")

	return nil
}

func (q *Queue) TaskRun(rpc common.RPCCall, j *common.Job) error {
	log.WithField("task", rpc.Job.UUID).Debug("Attempting to run task")

	// Add a defered catch for panic from within the tools
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("Recovered from Panic in Resource.TaskRun: %v", err)
		}
	}()

	// Grab the task specified by the UUID
	q.Lock()
	defer q.Unlock()
	log.WithField("Stack", q.stack).Debug("Stack")
	_, ok := q.stack[rpc.Job.UUID]

	// Check for a bad UUID
	if !ok {
		log.WithField("task", rpc.Job.UUID).Debug("Task with UUID provided does not exist.")
		return errors.New("Task with UUID provided does not exist.")
	}

	// Start or resume the task
	err := q.stack[rpc.Job.UUID].Run()
	if err != nil {
		return err
	}

	common.CopyJob(q.stack[rpc.Job.UUID].Status(), j)

	log.WithField("task", j.UUID).Debug("Task ran successfully")

	return nil

}

func (q *Queue) TaskQuit(rpc common.RPCCall, j *common.Job) error {
	log.WithField("task", rpc.Job.UUID).Debug("Attempting to quit task")

	// Add a defered catch for panic from within the tools
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("Recovered from Panic in Resource.TaskQuit: %v", err)
		}
	}()

	// Grab a lock and set the unlock on return
	q.Lock()
	defer q.Unlock()

	// Grab the task specified by the UUID
	_, ok := q.stack[rpc.Job.UUID]

	// Check for a bad UUID
	if !ok {
		log.WithField("task", rpc.Job.UUID).Debug("Task with UUID provided does not exist.")
		return errors.New("Task with UUID provided does not exist.")
	}

	// Quit the task and return the final result
	common.CopyJob(q.stack[rpc.Job.UUID].Quit(), j)

	// Remove quit job from stack
	delete(q.stack, rpc.Job.UUID)

	log.WithField("task", rpc.Job.UUID).Debug("Task quit and removed successfully")

	return nil
}

// TaskDone is called by the Queue to tell the Resource that it will no longer be asking about this task and we can
// remove it from our local stack.
func (q *Queue) TaskDone(rpc common.RPCCall, j *common.Job) error {
	log.WithField("task", rpc.Job.UUID).Debug("Queue said it is done with this task.")

	// Get a lock and make sure we unlock the stack on return
	q.Lock()
	defer q.Unlock()

	common.CopyJob(q.stack[rpc.Job.UUID].Status(), j)

	q.stack[rpc.Job.UUID].Done()

	// Delete the specific job
	delete(q.stack, rpc.Job.UUID)

	log.WithField("task", rpc.Job.UUID).Debug("Task has been removed from the local stack.")

	return nil
}

// Queue Tasks

func (q *Queue) ResourceTools(rpc common.RPCCall, tools *[]common.Tool) error {
	log.Debug("Gathering all tools")

	// Add a defered catch for panic from within the tools
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("Recovered from Panic in Resource.ResourceTools: %v", err)
		}
	}()

	q.RLock()
	defer q.RUnlock()

	var ts []common.Tool

	for i, _ := range q.tools {
		var tool common.Tool
		tool.Name = q.tools[i].Name()
		tool.Type = q.tools[i].Type()
		tool.Version = q.tools[i].Version()
		tool.UUID = q.tools[i].UUID()
		tool.Parameters = q.tools[i].Parameters()
		tool.Requirements = q.tools[i].Requirements()

		log.WithFields(log.Fields{
			"UUID": tool.UUID,
			"name": tool.Name,
			"type": tool.Type,
			"ver":  tool.Version,
		}).Debug("Tool added")

		ts = append(ts, tool)
	}

	*tools = ts

	return nil
}

func (q *Queue) AllTaskStatus(rpc common.RPCCall, j *[]common.Job) error {
	log.Debug("Gathering all Task Status")

	// Add a defered catch for panic from within the tools
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("Recovered from Panic in Resource.AllTaskStatus: %v", err)
		}
	}()

	log.Debug("Gathering status on all jobs")

	// Loop through any tasks in the stack and update their status while
	// grabing the Job object output
	var jobs []common.Job

	q.Lock()

	for i := range q.stack {
		jobs = append(jobs, q.stack[i].Status())
	}

	*j = jobs

	q.Unlock()

	return nil
}

func (q *Queue) JobManTest(rpc common.RPCCall, job *common.Job) error {
	common.CopyJob(rpc.Job, job)
	job.Status = "DONE"
	job.PerformanceTitle = "GH/s"
	job.PerformanceData = map[string]string{
		"1": "100",
		"2": "1000",
		"3": "10000",
		"4": "100000",
	}
	job.Parameters = map[string]string{
		"adv_options_loopback":     "false",
		"brute_increment":          "true",
		"brute_max_length":         "9",
		"brute_min_length":         "7",
		"brute_predefined_charset": "UPPER, lower, Number [9]",
		"brute_use_custom_chars":   "false",
		"dict_rules_use_random":    "false",
		"hashes_use_upload":        "false",
		"hashmode":                 "500",
		"use_adv_options":          "false",
		"new_param":                "true",
	}

	return nil
}
