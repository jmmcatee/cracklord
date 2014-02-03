package queue

import (
	"cl/common"
	"code.google.com/p/uuid"
	"errors"
	"log"
	"net/rpc"
	"sync"
	"time"
)

const (
	STATUS_EMPTY     = "Empty"
	STATUS_RUNNING   = "Running"
	STATUS_PAUSED    = "Paused"
	STATUS_EXHAUSTED = "Exhausted"
)

var KeeperDuration = 30 * time.Second
var NetworkTimeout = 2 * time.Second

type Queue struct {
	status string // Empty, Running, Paused, Exhausted
	pool   ResourcePool
	stack  []common.Job
	stats  Stats
	sync.RWMutex
	qk chan bool
}

// Add a job to the queue at the end of the stack
func (q *Queue) AddJob(j common.Job) error {
	// Lock the queue for the adding work
	q.Lock()
	defer q.Unlock()

	// Add job to stack
	q.stack = append(q.stack, j)
	jobIndex := len(q.stack) - 1

	// Add stats
	// TODO: Add more stats
	q.stats.IncJob()

	// Check if the Queue was empty
	if q.status == STATUS_EMPTY || q.status == STATUS_EXHAUSTED {
		// The Queue is empty so we need to start this job and the keeper
		// Find the first open resource
		for i, _ := range q.pool {
			// Make sure this resource isn't paused
			if q.pool[i].Paused {
				continue
			}

			// See if the tool exist on this resource
			tool, ok := q.pool[i].Tools[j.ToolUUID]
			if ok {
				// Tool exist, lets start the job on this resource and assign the resource to the job
				j.ResAssigned = i
				addJob := common.RPCCall{
					Auth: q.pool[i].RPCCall.Auth,
					Job:  j,
				}

				err := q.pool[i].Client.Call("Queue.AddTask", addJob, &j)
				if err != nil {
					return err
				}

				// Update the job in the stack
				q.stack[jobIndex] = j

				// We started a job so change the Queue status
				q.status = STATUS_RUNNING

				// Note the resources as being used
				q.pool[i].Hardware[tool.Requirements] = false

				// start the keeper as long as the status wasn't exhausted
				if q.status == STATUS_EMPTY {
					q.qk = make(chan bool)
					go q.keeper()
				}

				// We should be done so return no errors
				return nil
			}
		}
	}

	// If the queue is running or paused all we need to have done is add it to the queue
	return nil
}

func (q *Queue) PauseJob(jobuuid string) error {
	q.Lock()
	defer q.Unlock()

	// Loop through the stack looking for the job with a matching UUID
	for i, _ := range q.stack {
		if q.stack[i].UUID == jobuuid {
			// We have found the job so lets see if it running
			if q.stack[i].Status == common.STATUS_RUNNING {
				// Job is running so lets tell it to pause
				pauseJob := common.RPCCall{
					Auth: q.pool[q.stack[i].ResAssigned].RPCCall.Auth,
					Job:  q.stack[i],
				}

				err := q.pool[q.stack[i].ResAssigned].Client.Call("Queue.TaskPasue", pauseJob, q.stack[i])
				if err != nil {
					return err
				}

				// Task is now paused so update the resource
				hardware := q.pool[q.stack[i].ResAssigned].Tools[q.stack[i].ToolUUID].Requirements
				q.pool[q.stack[i].ResAssigned].Hardware[hardware] = true

				return nil
			} else {
				// The job was found but was not running so lets return an error
				return errors.New("Job given is not running. Current status is " + q.stack[i].Status)
			}
		}
	}

	// We didn't find the job so return an error
	return errors.New("Job does not exist!")
}

func (q *Queue) QuitJob(jobuuid string) error {
	q.Lock()
	defer q.Unlock()

	// Loop through the stack looking for the job to quit
	for i, _ := range q.stack {
		if q.stack[i].UUID == jobuuid {
			// We have found the job so lets check that it isn't already done
			s := q.stack[i].Status
			if s != common.STATUS_DONE && s != common.STATUS_FAILED && s != common.STATUS_QUIT {
				// Lets build the call to stop the job
				quitJob := common.RPCCall{
					Auth: q.pool[q.stack[i].ResAssigned].RPCCall.Auth,
					Job:  q.stack[i],
				}

				err := q.pool[q.stack[i].ResAssigned].Client.Call("Queue.QuitTask", quitJob, q.stack[i])
				if err != nil {
					return err
				}

				// Task has been quit without errors so update the available hardware and return
				hw := q.pool[q.stack[i].ResAssigned].Tools[q.stack[i].ToolUUID].Requirements
				q.pool[q.stack[i].ResAssigned].Hardware[hw] = true

				return nil
			}

			// The Jobs status is already stopped so lets return an error
			return errors.New("Job is already not running. Current status is " + s)
		}
	}

	// No job was found so return error
	return errors.New("Job does not exist!")
}

func (q *Queue) PauseResource(resUUID string) error {
	q.Lock()
	defer q.Unlock()

	// Check for UUID existance
	if _, ok := q.pool[resUUID]; !ok {
		return errors.New("Resource with UUID provided does not exist!")
	}

	// Loop through and pause any tasks running on the selected resource
	for i, _ := range q.stack {
		if q.stack[i].ResAssigned == resUUID && q.stack[i].Status == common.STATUS_RUNNING {
			// We found a task that is running so lets pause it
			pauseJob := common.RPCCall{
				Auth: q.pool[resUUID].RPCCall.Auth,
				Job:  q.stack[i],
			}

			err := q.pool[resUUID].Client.Call("Queue.TaskPause", pauseJob, q.stack[i])
			if err != nil {
				return err
			}

			// Task should now be paused to free up the resource
			hw := q.pool[resUUID].Tools[q.stack[i].ToolUUID].Requirements
			q.pool[resUUID].Hardware[hw] = true
		}
	}

	// All tasks that would be running should now be paused so lets pause the resource
	res, _ := q.pool[resUUID]
	res.Paused = true
	q.pool[resUUID] = res

	return nil
}

func (q *Queue) ResumeResource(resUUID string) error {
	q.Lock()
	q.Unlock()

	// Check for UUID existance
	if _, ok := q.pool[resUUID]; !ok {
		return errors.New("Resource with UUID provided does not exist!")
	}

	// Pool exists so unpause it
	res, _ := q.pool[resUUID]
	res.Paused = false
	q.pool[resUUID] = res

	// The keeper will take it from here
	return nil
}

// Pause the whole queue and return any and all errors pausing active jobs/tasks
func (q *Queue) PauseQueue() []error {
	var e []error

	q.Lock()
	defer q.Unlock()

	// First order of business is to kill the keeper
	q.qk <- true
	q.qk = nil

	// Now we need to be 100% up-to-date
	q.updateQueue()

	// Loop through and pause all active jobs
	for i, _ := range q.stack {
		if q.stack[i].Status == common.STATUS_RUNNING {
			// Get some helpful values
			tooluuid := q.stack[i].ToolUUID
			resuuid := q.stack[i].ResAssigned

			// This task is running and needs to be paused
			pauseJob := common.RPCCall{
				Auth: q.pool[resuuid].RPCCall.Auth,
				Job:  q.stack[i],
			}

			err := q.pool[resuuid].Client.Call("Queue.TaskPause", pauseJob, &q.stack[i])
			if err != nil {
				// Note the error but now mark the job as Failed
				// This is a definied way of dealing with this to avoid complicated error handling
				q.stack[i].Status = common.STATUS_FAILED
				q.stack[i].Error = err.Error()
				e = append(e, err)
			}

			// Update available hardware
			q.pool[resuuid].Hardware[q.pool[resuuid].Tools[tooluuid].Requirements] = true
		}
	}

	// All jobs/tasks should now be paused so lets set the Queue Status
	q.status = STATUS_PAUSED

	return e
}

func (q *Queue) ResumeQueue() {
	if q.status == STATUS_PAUSED {
		q.Lock()
		defer q.Unlock()

		// Change status and start the keeper
		q.status = STATUS_RUNNING
		q.qk = make(chan bool)
		go q.keeper()
	}

	return
}

// Given a new slice of the UUIDs for all jobs in order, reorder the stack
func (q *Queue) StackReorder(uuids []string) []error {
	q.Lock()
	defer q.Unlock()

	// Check all the UUIDs before we do anything
	l := len(uuids)
	if l != len(q.stack) {
		return []error{errors.New("The wrong number of UUIDs were provided.")}
	}

	// Get the UUIDs in the check map
	uuidCheck := make(map[string]common.Job)
	for _, v := range uuids {
		uuidCheck[v] = common.Job{}
	}

	// Loop through the stack and check for bad UUIDs
	for i, _ := range q.stack {
		j := q.stack[i]
		if _, ok := uuidCheck[j.UUID]; ok {
			return []error{errors.New("All Job UUIDs must be provided!")}
		}
	}

	// UUIDs are good so pause the queue (we will return the errors at the end)
	err := q.PauseQueue()

	// Get Job information to build new stack
	for i, _ := range q.stack {
		uuidCheck[q.stack[i].UUID] = q.stack[i]
	}

	newStack := []common.Job{}
	for _, v := range uuidCheck {
		newStack = append(newStack, v)
	}

	// If no errors were given we now have a new stack so lets assign it and finally unlock the Queue
	q.stack = newStack

	// Resume the Queue
	q.ResumeQueue()

	// Return the errors from the QueuePause if there were any
	if err != nil {
		return err
	}

	return nil
}

// Quit the queue
func (q *Queue) Quit() []common.Job {
	q.Lock()
	defer q.Unlock()

	// First order of business is to kill the keeper
	q.qk <- true
	q.qk = nil

	// Now we need to be 100% up-to-date
	q.updateQueue()

	// Loop through and quit any job that is not done, failed, quit
	for i, _ := range q.stack {
		s := q.stack[i].Status

		// If the job is running quit it
		if s == common.STATUS_RUNNING && s == common.STATUS_PAUSED {
			// Build the quit call
			quitJob := common.RPCCall{
				Auth: q.pool[q.stack[i].ResAssigned].RPCCall.Auth,
				Job:  q.stack[i],
			}

			err := q.pool[q.stack[i].ResAssigned].Client.Call("Queue.TaskQuit", quitJob, &q.stack[i])
			// Log any errors but we don't care from a flow perspective
			if err != nil {
				log.Println(err.Error())
			}
		}
	}

	// We have looped through all jobs so return the last status
	// The Queue should be deleted or nulled from outside this context
	return q.stack
}

// The Keeper runs in a different goroutine to keep the queue roughly up-to-date
// It will need to aquire a lock each time it does this
// The q.qk channel needs to be created and maintained outside of this function
func (q *Queue) keeper() {
	go func() {
		// Setup timer for keeper
		kTimer := time.After(KeeperDuration)

		for {
			select {
			case <-kTimer:
				// Get lock
				q.Lock()

				// Update all running jobs
				q.updateQueue()

				// Look for open resources
				for i, _ := range q.pool {
					for r, b := range q.pool[i].Hardware {
						if b {
							// This resource is free, so lets find a job for it
							for ji, _ := range q.stack {
								requirement := q.pool[i].Tools[q.stack[ji].ToolUUID].Requirements
								s := q.stack[ji].Status
								startable := s == common.STATUS_CREATED && s == common.STATUS_PAUSED
								if requirement == r && startable {
									// Found a task that can be started and needs this resource
									// Build the call
									startJob := common.RPCCall{
										Auth: q.pool[i].RPCCall.Auth,
										Job:  q.stack[ji],
									}

									err := q.pool[i].Client.Call("Queue.AddTask", startJob, &q.stack[ji])
									if err != nil {
										// something went wrong so log it, but the keeper will get this
										// resource the next time through
										log.Println(err.Error())
									} else {
										// there were no errors so lets take up the hardware resource
										q.pool[i].Hardware[r] = false
									}
								}
							}
						}
					}
				}

				// Release the Lock
				q.Unlock()
			case <-q.qk:

			}
		}
	}()
}

// This is an internal function used to update the status of all Jobs.
// A LOCK SHOULD ALREADY BE HELD TO CALL THIS FUNCTION.
func (q *Queue) updateQueue() {
	// Loop through jobs and get the status of running jobs
	for i, _ := range q.stack {
		if q.stack[i].Status == common.STATUS_RUNNING {
			// Build status update call
			jobStatus := common.RPCCall{
				Auth: q.pool[q.stack[i].ResAssigned].RPCCall.Auth,
				Job:  q.stack[i],
			}

			err := q.pool[q.stack[i].ResAssigned].Client.Call("Queue.TaskStatus", jobStatus, &q.stack[i])
			// we care about the errors, but only from a logging perspective
			if err != nil {
				log.Println(err.Error())
			}

			// Check if this is now no longer running
			if q.stack[i].Status != common.STATUS_RUNNING {
				// Release the resources from this change
				hw := q.pool[q.stack[i].ResAssigned].Tools[q.stack[i].ToolUUID].Requirements
				q.pool[q.stack[i].ResAssigned].Hardware[hw] = true
			}
		}
	}
}

func (q *Queue) Types() []string {
	q.RLock()
	defer q.RUnlock()

	// Loop through tools and get the types
	var types []string
	for _, v := range q.pool {
		for _, t := range v.Tools {
			for _, j := range types {
				if j != t.Type {
					types = append(types, t.Type)
				}
			}
		}
	}

	return types
}

func (q *Queue) Tools() map[string]common.Tool {
	q.RLock()
	defer q.RUnlock()

	// cycle through all the attached resources for unique tools
	var tools = make(map[string]common.Tool)
	for _, res := range q.pool {
		for uuid, t := range res.Tools {
			// Check if tool already exists in the tools map
			_, ok := tools[uuid]
			if !ok {
				// Tool doesn't exit already so add it
				tools[uuid] = t
			}
		}
	}

	return tools
}

func (q *Queue) AddResource(n string, addr string, auth string) error {
	// Create empty resource
	res := new(Resource)

	// Build the RPC client for the resource
	var err error
	res.Client, err = rpc.Dial(n, addr)
	if err != nil {
		return err
	}

	// Build the default RPCCall struct
	res.RPCCall = common.RPCCall{Auth: auth}

	// Get Hardware
	res.Client.Call("Queue.ResourceHardware", res.RPCCall, &res.Hardware)

	// Set all hardware as available
	for key, _ := range res.Hardware {
		res.Hardware[key] = true
	}

	// Get Tools
	res.Client.Call("Queue.ResourceTools", res.RPCCall, &res.Tools)

	q.Lock()
	defer q.Unlock()

	// Loop through new tools and look for those we already have
	// If we find a tool we already have we need to make the UUID the same
	// If we don't already have this we can just leave the UUID assigned by the resource
	for _, v := range q.pool {
		for _, t := range v.Tools {
			// Loop through new tools
			for newUUID, newT := range res.Tools {
				if common.CompareTools(newT, t) {
					// Change UUID to already set one
					res.Tools[t.UUID] = newT
					delete(res.Tools, newUUID)
				}
			}
		}
	}

	// Add resource to resource pool with generated UUID
	q.pool[uuid.New()] = *res

	return nil
}

func (q *Queue) RemoveResource(uuid string) error {
	// Check for the resource with given UUID
	_, ok := q.pool[uuid]
	if ok {
		return errors.New("Given Resource UUID does not exist.")
	}

	// Lock the queue
	q.Lock()
	defer q.Unlock()

	// Loop through any jobs assigned to the resource and quit them if they are not completed
	for i, v := range q.stack {
		if v.ResAssigned == uuid {
			// Check status
			if v.Status == common.STATUS_RUNNING || v.Status == common.STATUS_PAUSED {
				// Quit the task
				quitTask := common.RPCCall{
					Auth: q.pool[uuid].RPCCall.Auth,
					Job:  v,
				}

				err := q.pool[uuid].Client.Call("Queue.TaskQuit", quitTask, &q.stack[i])
				if err != nil {
					log.Println(err.Error())
				}
			}
		}
	}

	// Remove from pool
	delete(q.pool, uuid)

	return nil
}
