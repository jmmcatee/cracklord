package resource

import (
	"errors"
	"github.com/jmmcatee/cracklord/common"
	"log"
	"net"
	"net/rpc"
	"sync"
)

// TODO: Add function for adding tools and assign a UUID

const (
	ERROR_AUTH    = "Call to resource did not have the proper authentication token."
	ERROR_NO_TOOL = "Tool specified does not exit."
)

// This will need to be called with a WaitGroup to handle other calls without
// the program closing.
func StartResource(n string, addr string, q *Queue) net.Listener {
	ns := rpc.NewServer()
	ns.Register(q)

	listen, err := net.Listen(n, addr)
	if err != nil {
		log.Fatal("Resource Listener Error: ", err)
	}

	go func() {
		for {
			conn, err := listen.Accept()
			if err != nil {
				log.Printf("Error with Resource Listener: %s", err.Error())
			}

			go ns.ServeConn(conn)
		}
	}()

	return listen
}

type Queue struct {
	stack map[string]common.Tasker
	tools []common.Tooler
	sync.RWMutex
	authToken string
	hardware  map[string]bool
}

// Task RPC functions

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
	// Check authentication token
	if rpc.Auth != q.authToken {
		return errors.New(ERROR_AUTH)
	}

	// variable to hold the tasker
	var tasker common.Tasker
	// loop through common.Toolers for matching tool
	q.RLock()
	for i, _ := range q.tools {
		if q.tools[i].UUID() == rpc.Job.ToolUUID {
			tasker = q.tools[i].NewTask(rpc.Job)
		}
	}
	q.RUnlock()

	// Check if no tool was found and return error
	if tasker == nil {
		return errors.New(ERROR_NO_TOOL)
	}

	// Looks good so lets add to the stack
	q.Lock()
	if q.stack == nil {
		q.stack = make(map[string]common.Tasker)
	}

	q.stack[rpc.Job.UUID] = tasker

	// Everything should be paused by the control queue so start this job
	err := q.stack[rpc.Job.UUID].Run()
	if err != nil {
		return errors.New("Error starting task on the resource: " + err.Error())
	}

	// Grab the status and return that job to the control queue
	*rj = q.stack[rpc.Job.UUID].Status()
	q.Unlock()

	return nil
}

func (q *Queue) TaskStatus(rpc common.RPCCall, j *common.Job) error {
	// Check authentication token
	if rpc.Auth != q.authToken {
		return errors.New(ERROR_AUTH)
	}

	// Grab the task specified by the UUID and return its status
	q.Lock()
	_, ok := q.stack[rpc.Job.UUID]

	// Check for a bad UUID
	if ok != false {
		errors.New("Task with UUID provided does not exist.")
	}

	*j = q.stack[rpc.Job.UUID].Status()

	q.Unlock()

	return nil
}

func (q *Queue) TaskPause(rpc common.RPCCall, j *common.Job) error {
	// Check authentication token
	if rpc.Auth != q.authToken {
		return errors.New(ERROR_AUTH)
	}

	// Grab the task specified by the UUID
	q.Lock()
	_, ok := q.stack[rpc.Job.UUID]

	// Check for a bad UUID
	if ok {
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

	return nil
}

func (q *Queue) TaskRun(rpc common.RPCCall, j *common.Job) error {
	// Check authentication token
	if rpc.Auth != q.authToken {
		return errors.New(ERROR_AUTH)
	}

	// Grab the task specified by the UUID
	q.Lock()
	_, ok := q.stack[rpc.Job.UUID]

	// Check for a bad UUID
	if ok != false {
		errors.New("Task with UUID provided does not exist.")
	}

	// Start or resume the task
	err := q.stack[rpc.Job.UUID].Run()
	if err != nil {
		return err
	}

	*j = q.stack[rpc.Job.UUID].Status()
	q.Unlock()

	return nil

}

func (q *Queue) TaskQuit(rpc common.RPCCall, j *common.Job) error {
	// Check authentication token
	if rpc.Auth != q.authToken {
		return errors.New(ERROR_AUTH)
	}

	// Grab the task specified by the UUID
	q.Lock()
	_, ok := q.stack[rpc.Job.UUID]

	// Check for a bad UUID
	if ok != false {
		errors.New("Task with UUID provided does not exist.")
	}

	// Quit the task and return the final result
	*j = q.stack[rpc.Job.UUID].Quit()

	// Remove quit job from stack
	delete(q.stack, rpc.Job.UUID)
	q.Unlock()

	return nil
}

// Queue Tasks

func (q *Queue) ResourceTools(rpc common.RPCCall, tools *[]common.Tool) error {
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

		ts = append(ts, tool)
	}

	*tools = ts

	return nil
}

func (q *Queue) AllTaskStatus(rpc common.RPCCall, j *[]common.Job) error {
	// Check authentication token
	if rpc.Auth != q.authToken {
		return errors.New(ERROR_AUTH)
	}

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
