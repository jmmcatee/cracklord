package resource

import (
	"code.google.com/p/go-uuid/uuid"
	"errors"
	log "github.com/Sirupsen/logrus"
	"github.com/jmmcatee/cracklord/common"
	"sync"
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
	authToken string
	hardware  map[string]bool
}

func NewResourceQueue(token string) Queue {
	log.WithField("token", token).Debug("New resource queue created.")
	return Queue{
		stack:     map[string]common.Tasker{},
		tools:     []common.Tooler{},
		authToken: token,
		hardware:  map[string]bool{},
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
func (q *Queue) Ping(ping int64, pong *int64) error {
	q.Lock()

	pong = &ping

	q.Unlock()
	return nil
}

func (q *Queue) ResourceHardware(rpc common.RPCCall, hw *map[string]bool) error {
	// Check authentication token
	if rpc.Auth != q.authToken {
		return errors.New(ERROR_AUTH)
	}

	q.RLock()
	defer q.RUnlock()

	*hw = q.hardware

	return nil
}

func (q *Queue) AddTask(rpc common.RPCCall, rj *common.Job) error {
	log.WithFields(log.Fields{
		"name": rpc.Job.Name,
		"uuid": rpc.Job.UUID,
	}).Info("Job added")

	log.WithFields(log.Fields{
		"uuid":       rpc.Job.UUID,
		"parameters": rpc.Job.Parameters,
	})

	// Add a defered catch for panic from within the tools
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("Recovered from Panic in Resource.AddTask: %v", err)
		}
	}()

	// Check authentication token
	if rpc.Auth != q.authToken {
		log.Warn("Authentication token was not recognized")
		return errors.New(ERROR_AUTH)
	}

	// variable to hold the tasker
	var tasker common.Tasker
	var err error
	// loop through common.Toolers for matching tool
	q.RLock()
	for i, _ := range q.tools {
		if q.tools[i].UUID() == rpc.Job.ToolUUID {
			tasker, err = q.tools[i].NewTask(rpc.Job)
			if err != nil {
				return err
			}
		}
	}
	q.RUnlock()

	// Check if no tool was found and return error
	if tasker == nil {
		log.Warn("An error occured, we could not find the tool requested")
		return errors.New(ERROR_NO_TOOL)
	}
	log.WithFields(log.Fields{
		"task": rpc.Job.UUID,
	}).Debug("Tasker created")

	// Looks good so lets add to the stack
	q.Lock()
	if q.stack == nil {
		q.stack = make(map[string]common.Tasker)
	}

	q.stack[rpc.Job.UUID] = tasker

	// Everything should be paused by the control queue so start this job
	err = q.stack[rpc.Job.UUID].Run()
	if err != nil {
		log.Debug("Error starting task on resource")
		return errors.New("Error starting task on the resource: " + err.Error())
	}

	// Grab the status and return that job to the control queue
	*rj = q.stack[rpc.Job.UUID].Status()
	q.Unlock()

	return nil
}

func (q *Queue) TaskStatus(rpc common.RPCCall, j *common.Job) error {
	log.WithField("task", j.UUID).Debug("Attempting to gather task status")

	// Add a defered catch for panic from within the tools
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("Recovered from Panic in Resource.TaskStatus: %v", err)
		}
	}()

	// Check authentication token
	if rpc.Auth != q.authToken {
		log.Warn("Authentication token was not matched")
		return errors.New(ERROR_AUTH)
	}

	// Grab the task specified by the UUID and return its status
	q.Lock()
	_, ok := q.stack[rpc.Job.UUID]

	// Check for a bad UUID
	if ok != false {
		log.WithField("task", j.UUID).Error("Task with UUID provided does not exist.")
		errors.New("Task with UUID provided does not exist.")
	}

	*j = q.stack[rpc.Job.UUID].Status()

	q.Unlock()

	return nil
}

func (q *Queue) TaskPause(rpc common.RPCCall, j *common.Job) error {
	log.WithField("task", j.UUID).Debug("Attempting to pause task")

	// Add a defered catch for panic from within the tools
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("Recovered from Panic in Resource.TaskPause: %v", err)
		}
	}()

	// Check authentication token
	if rpc.Auth != q.authToken {
		return errors.New(ERROR_AUTH)
	}

	// Grab the task specified by the UUID
	q.Lock()
	_, ok := q.stack[rpc.Job.UUID]

	// Check for a bad UUID
	if ok {
		log.WithField("task", j.UUID).Debug("Task with UUID provided does not exist.")
		errors.New("Task with UUID provided does not exist.")
	}

	// Pause the task
	err := q.stack[rpc.Job.UUID].Pause()
	if err != nil {
		// return the error but quit the job with status Failed
		// This is a definied behavior that we will not for all tools
		q.stack[rpc.Job.UUID].Quit()
		return err
	}

	*j = q.stack[rpc.Job.UUID].Status()
	q.Unlock()

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

	// Check authentication token
	if rpc.Auth != q.authToken {
		return errors.New(ERROR_AUTH)
	}

	// Grab the task specified by the UUID
	q.Lock()
	defer q.Unlock()
	log.WithField("Stack", q.stack).Debug("Stack")
	_, ok := q.stack[rpc.Job.UUID]

	// Check for a bad UUID
	if ok == false {
		log.WithField("task", rpc.Job.UUID).Debug("Task with UUID provided does not exist.")
		return errors.New("Task with UUID provided does not exist.")
	}

	// Start or resume the task
	err := q.stack[rpc.Job.UUID].Run()
	if err != nil {
		return err
	}

	*j = q.stack[rpc.Job.UUID].Status()

	log.WithField("task", j.UUID).Debug("Task ran successfully")

	return nil

}

func (q *Queue) TaskQuit(rpc common.RPCCall, j *common.Job) error {
	log.WithField("task", j.UUID).Debug("Attempting to quit task")

	// Add a defered catch for panic from within the tools
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("Recovered from Panic in Resource.TaskQuit: %v", err)
		}
	}()

	// Check authentication token
	if rpc.Auth != q.authToken {
		return errors.New(ERROR_AUTH)
	}

	// Grab the task specified by the UUID
	q.Lock()
	_, ok := q.stack[rpc.Job.UUID]

	// Check for a bad UUID
	if ok != false {
		log.WithField("task", j.UUID).Debug("Task with UUID provided does not exist.")
		errors.New("Task with UUID provided does not exist.")
	}

	// Quit the task and return the final result
	*j = q.stack[rpc.Job.UUID].Quit()

	// Remove quit job from stack
	delete(q.stack, rpc.Job.UUID)
	q.Unlock()

	log.WithField("task", j.UUID).Debug("Task ran successfully")

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

	// Check authentication token
	if rpc.Auth != q.authToken {
		log.Warn("An error occured while trying to match the authentication token")
		return errors.New(ERROR_AUTH)
	}

	log.Debug("Gathering status on all jobs")

	// Loop through any tasks in the stack and update their status while
	// grabing the Job object output
	var jobs []common.Job

	q.Lock()

	for i, _ := range q.stack {
		jobs = append(jobs, q.stack[i].Status())
	}

	*j = jobs

	q.Unlock()

	return nil
}
