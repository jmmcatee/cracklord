package queue

import (
	"code.google.com/p/go-uuid/uuid"
	"encoding/json"
	"errors"
	log "github.com/Sirupsen/logrus"
	"github.com/jmmcatee/cracklord/common"
	"net/rpc"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	STATUS_EMPTY     = "Empty"
	STATUS_RUNNING   = "Running"
	STATUS_PAUSED    = "Paused"
	STATUS_EXHAUSTED = "Exhausted"
)

var KeeperDuration time.Duration
var NetworkTimeout time.Duration
var StateFileLocation string

type Queue struct {
	status string // Empty, Running, Paused, Exhausted
	pool   ResourcePool
	stack  []common.Job
	stats  Stats
	sync.RWMutex
	qk chan bool
}

type StateFile struct {
	Stack []common.Job `json:"stack"`
	Pool  ResourcePool `json:"pool"`
}

func NewQueue(statefile string, updatetime int, timeout int) Queue {
	//Setup the options
	StateFileLocation = statefile
	KeeperDuration = time.Duration(updatetime) * time.Second
	NetworkTimeout = time.Duration(timeout) * time.Second

	// Build the queue
	q := Queue{
		status: STATUS_EMPTY,
		pool:   NewResourcePool(),
		stack:  []common.Job{},
		stats:  NewStats(),
	}

	if _, err := os.Stat(StateFileLocation); err == nil {
		q.parseState()
	}

	log.WithFields(log.Fields{
		"statefile":  StateFileLocation,
		"keepertime": KeeperDuration,
		"nettimeout": NetworkTimeout,
	}).Debug("Setup a new queue")

	return q
}

func (q *Queue) writeState() error {
	var s StateFile

	//Create a state fila in case we are rebooted
	stateFile, err := os.Create(StateFileLocation)
	if err != nil {
		log.WithField("error", err.Error()).Fatal("Unable to write to state file")
		return err
	}
	stateEncoder := json.NewEncoder(stateFile)
	s.Stack = q.stack
	s.Pool = q.pool

	stateEncoder.Encode(s)
	stateFile.Close()

	log.Debug("State file written successfully.")

	return nil
}

func (q *Queue) parseState() error {
	var s StateFile

	stateFile, err := os.Open(StateFileLocation)
	if err != nil {
		log.WithField("error", err.Error()).Error("An error occured opening the state file.")
		return err
	}

	stateDecoder := json.NewDecoder(stateFile)
	err = stateDecoder.Decode(&s)
	if err != nil {
		log.WithField("error", err.Error()).Error("An error occured decoding the state file.")
		return err
	}
	stateFile.Close()

	for id, v := range s.Pool {
		log.WithFields(log.Fields{
			"name": v.Name,
			"id":   id,
		}).Debug("Added resource from state file.")

		v.Address = "(disconnected)"
		v.Status = common.STATUS_QUIT
		q.pool[id] = v
	}
	for i, _ := range s.Stack {
		log.WithFields(log.Fields{
			"name": s.Stack[i].Name,
			"id":   s.Stack[i].UUID,
		}).Debug("Added job from state file.")
		s.Stack[i].Status = common.STATUS_QUIT
		q.stack = append(q.stack, s.Stack[i])
	}

	return nil
}

// Add a job to the queue at the end of the stack
func (q *Queue) AddJob(j common.Job) error {
	logger := log.WithFields(log.Fields{
		"jobid":   j.UUID,
		"jobname": j.Name,
	})

	// Lock the queue for the adding work
	q.Lock()
	defer q.Unlock()

	logger.Debug("Queue locked.")

	// Add job to stack
	q.stack = append(q.stack, j)
	jobIndex := len(q.stack) - 1
	logger.Debug("job added to stack.")

	// Add stats
	// TODO: Add more stats
	q.stats.IncJob()

	// Check if the Queue was empty
	if q.status == STATUS_EMPTY || q.status == STATUS_EXHAUSTED {
		logger.Debug("Queue is empty, job needs starting.")
		// The Queue is empty so we need to start this job and the keeper
		// Find the first open resource
		for i, _ := range q.pool {
			logger.WithField("resource", q.pool[i].Name).Debug("Looking for resource.")

			// Make sure this resource isn't paused
			if q.pool[i].Status == common.STATUS_PAUSED || q.pool[i].Status == common.STATUS_QUIT {
				continue
			}

			// See if the tool exist on this resource
			tool, ok := q.pool[i].Tools[j.ToolUUID]
			if ok {
				logger.WithFields(log.Fields{
					"resource": q.pool[i].Name,
					"tool":     tool.Name,
				}).Debug("Tool exists on resource.")

				// It is now possible that the job about to be assigned to the resource
				// actually has a Tool UUID different than the one the Queue has for this
				// tool. This is because the Queue changes the UUID if it has two identical
				// tools. We need to set the Job's ToolUUID to the real UUID of the tool
				// if this is the case
				if tool.UUID != j.ToolUUID {
					log.WithField("toolid", tool.UUID).Debug("Changed tool UUID to match resource.")
					j.ToolUUID = tool.UUID
				}

				// Tool exist, lets start the job on this resource and assign the resource to the job
				j.ResAssigned = i
				addJob := common.RPCCall{
					Auth: q.pool[i].RPCCall.Auth,
					Job:  j,
				}

				logger.Debug("Queue.AddTask RPC call started.")
				err := q.pool[i].Client.Call("Queue.AddTask", addJob, &j)
				if err != nil {
					logger.WithField("error", err.Error()).Error("There was a problem making an RPC call.")
					return err
				}

				// Update the job in the stack
				q.stack[jobIndex] = j

				// Note the resources as being used
				q.pool[i].Hardware[tool.Requirements] = false

				// start the keeper as long as the status wasn't exhausted
				if q.status == STATUS_EMPTY {
					logger.Debug("Keeper started")
					q.qk = make(chan bool)
					go q.keeper()
				}

				// We started a job so change the Queue status
				q.status = STATUS_RUNNING

				// We should be done so return no errors
				return nil
			}

			// Tool did not exist... return error
			return errors.New("Tool did not exist for jobs provided.")
		}
	}

	// If the queue is running or paused all we need to have done is add it to the queue
	return nil
}

// Get the full queue stack
func (q *Queue) AllJobs() []common.Job {
	log.Debug("Gathering all jobs from queue.")

	q.Lock()

	q.Unlock()
	return q.stack
}

// Get one specific job
func (q *Queue) JobInfo(jobUUID string) common.Job {
	log.WithField("job", jobUUID).Debug("Gathering information on job.")
	q.Lock()
	defer q.Unlock()

	for _, job := range q.stack {
		if job.UUID == jobUUID {
			return job
		}
	}

	return common.Job{}
}

func (q *Queue) PauseJob(jobuuid string) error {
	log.WithField("job", jobuuid).Info("Attempting to pause job.")
	q.Lock()
	defer q.Unlock()

	// Loop through the stack looking for the job with a matching UUID
	for i, _ := range q.stack {
		if q.stack[i].UUID == jobuuid {
			log.WithFields(log.Fields{
				"job":    jobuuid,
				"status": q.stack[i].Status,
			}).Debug("Job found in queue.")

			// We have found the job so lets see if it running
			if q.stack[i].Status == common.STATUS_RUNNING {
				// Job is running so lets tell it to pause
				pauseJob := common.RPCCall{
					Auth: q.pool[q.stack[i].ResAssigned].RPCCall.Auth,
					Job:  q.stack[i],
				}

				err := q.pool[q.stack[i].ResAssigned].Client.Call("Queue.TaskPause", pauseJob, &q.stack[i])
				log.WithField("job", jobuuid).Debug("Calling Queue.TaskPause on remote resource.")
				if err != nil {
					log.WithFields(log.Fields{
						"job":   jobuuid,
						"error": err.Error(),
					}).Error("An error occurred while trying to pause a remote job.")
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
	log.WithField("job", jobuuid).Info("Attempting to pause job.")

	q.Lock()
	defer q.Unlock()

	// Loop through the stack looking for the job to quit
	for i, _ := range q.stack {
		if q.stack[i].UUID == jobuuid {
			log.WithFields(log.Fields{
				"job":    jobuuid,
				"status": q.stack[i].Status,
			}).Debug("Job found in queue.")

			// We have found the job so lets check that it isn't already done
			s := q.stack[i].Status
			if s != common.STATUS_DONE && s != common.STATUS_FAILED && s != common.STATUS_QUIT {
				// Lets build the call to stop the job
				quitJob := common.RPCCall{
					Auth: q.pool[q.stack[i].ResAssigned].RPCCall.Auth,
					Job:  q.stack[i],
				}

				err := q.pool[q.stack[i].ResAssigned].Client.Call("Queue.TaskQuit", quitJob, &q.stack[i])
				log.WithField("job", jobuuid).Debug("Attempting to call Queue.TaskQuit on remote resource.")
				if err != nil {
					log.WithFields(log.Fields{
						"job":   jobuuid,
						"error": err.Error(),
					}).Error("An error occurred while trying to quit a remote job.")
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

func (q *Queue) RemoveJob(jobuuid string) error {
	log.WithField("job", jobuuid).Debug("Attempting to remove job")
	q.Lock()

	// Loop through and find the job
	for i, _ := range q.stack {
		if q.stack[i].UUID == jobuuid {
			log.WithFields(log.Fields{
				"job":    jobuuid,
				"status": q.stack[i].Status,
			}).Debug("Job found in queue.")

			// We have the job so check to make sure it isn't running
			s := q.stack[i].Status
			if s == common.STATUS_RUNNING {
				// Quit the job
				q.Unlock()
				err := q.QuitJob(jobuuid)
				q.Lock()
				if err != nil {
					q.Unlock()
					return err
				}
			}

			// Job should now be quit so lets rebuild the stack
			newStack := []common.Job{}
			for _, v := range q.stack {
				if v.UUID != jobuuid {
					newStack = append(newStack, v)
				}
			}

			// Rest stack
			q.stack = newStack

			// Stack has been cleaned so return no errors
			q.Unlock()
			return nil
		}
	}

	q.Unlock()
	return errors.New("Job not found.")
}

func (q *Queue) PauseResource(resUUID string) error {
	log.WithField("resource", resUUID).Debug("Attempting to pause resource")

	q.Lock()
	defer q.Unlock()

	// Check for UUID existance
	if _, ok := q.pool[resUUID]; !ok {
		return errors.New("Resource with UUID provided does not exist!")
	}

	// Loop through and pause any tasks running on the selected resource
	for i, _ := range q.stack {
		log.WithFields(log.Fields{
			"resource":  q.stack[i].ResAssigned,
			"job":       q.stack[i].UUID,
			"jobstatus": q.stack[i].Status,
		}).Debug("Identifying running jobs on paused resource.")

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
	res.Status = common.STATUS_PAUSED
	q.pool[resUUID] = res

	return nil
}

func (q *Queue) ResumeResource(resUUID string) error {
	log.WithField("resource", resUUID).Debug("Attempting to resume resource.")

	q.Lock()
	defer q.Unlock()

	// Check for UUID existance
	if _, ok := q.pool[resUUID]; !ok {
		return errors.New("Resource with UUID provided does not exist!")
	}

	if q.pool[resUUID].Status != common.STATUS_PAUSED {
		return errors.New("Resource is not paused!")
	}

	// Pool exists so unpause it
	res, _ := q.pool[resUUID]
	res.Status = common.STATUS_RUNNING
	q.pool[resUUID] = res

	// The keeper will take it from here
	return nil
}

// Pause the whole queue and return any and all errors pausing active jobs/tasks
func (q *Queue) PauseQueue() []error {
	log.Debug("Attempting to pause entire queue.")

	var e []error

	// First order of business is to kill the keeper
	q.qk <- true
	q.Lock()
	defer q.Unlock()
	q.qk = nil

	// Now we need to be 100% up-to-date
	q.updateQueue()

	log.Debug("Queue update completed.")

	// Loop through and pause all active jobs
	for i, _ := range q.stack {
		joblog := log.WithFields(log.Fields{
			"resource":  q.stack[i].ResAssigned,
			"job":       q.stack[i].UUID,
			"jobstatus": q.stack[i].Status,
		})
		joblog.Debug("Processing job.")

		if q.stack[i].Status == common.STATUS_RUNNING {
			joblog.Debug("Found running job, attempting to stop")

			// Get some helpful values
			tooluuid := q.stack[i].ToolUUID
			resuuid := q.stack[i].ResAssigned

			// This task is running and needs to be paused
			pauseJob := common.RPCCall{
				Auth: q.pool[resuuid].RPCCall.Auth,
				Job:  q.stack[i],
			}

			joblog.Debug("Calling Queue.TaskPause on job")
			err := q.pool[resuuid].Client.Call("Queue.TaskPause", pauseJob, &q.stack[i])
			if err != nil {
				// Note the error but now mark the job as Failed
				// This is a definied way of dealing with this to avoid complicated error handling
				q.stack[i].Status = common.STATUS_FAILED
				q.stack[i].Error = err.Error()
				e = append(e, err)

				joblog.Debug("There was a problem pausing the remote job.")
			}

			// Update available hardware
			q.pool[resuuid].Hardware[q.pool[resuuid].Tools[tooluuid].Requirements] = true
		}
	}

	// All jobs/tasks should now be paused so lets set the Queue Status
	q.status = STATUS_PAUSED
	log.Debug("Queue paused.")

	return e
}

func (q *Queue) ResumeQueue() {
	log.Debug("Attempting to restart the queue")

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
func (q *Queue) StackReorder(uuids []string) error {
	q.Lock()

	// Check all the UUIDs before we do anything
	l := len(uuids)
	if l != len(q.stack) {
		return errors.New("The wrong number of UUIDs were provided.")
	}

	// Get the UUIDs in the check map
	uuidCheck := make(map[string]common.Job)
	for _, v := range uuids {
		uuidCheck[v] = common.Job{}
	}

	// Loop through the stack and check for bad UUIDs
	for i, _ := range q.stack {
		j := q.stack[i]
		log.WithField("uuid", j.UUID).Debug("Checking UUID on reorder stack.")
		if _, ok := uuidCheck[j.UUID]; !ok {
			return errors.New("All Job UUIDs must be provided!")
		}
	}

	// UUIDs are good so pause the queue (we will return the errors at the end)
	q.Unlock()
	err := q.PauseQueue()
	q.Lock()

	// Get Job information to build new stack
	for i, _ := range q.stack {
		log.WithField("uuid", q.stack[i].UUID).Debug("Building new job stack.")
		uuidCheck[q.stack[i].UUID] = q.stack[i]
	}

	newStack := []common.Job{}
	for _, v := range uuids {
		newStack = append(newStack, uuidCheck[v])
	}

	// If no errors were given we now have a new stack so lets assign it and finally unlock the Queue
	log.Debug("Assigning new stack to queue stack")
	q.stack = newStack

	// Resume the Queue
	q.Unlock()
	q.ResumeQueue()
	q.Lock()
	defer q.Unlock()

	// Return the errors from the QueuePause if there were any
	if err != nil {
		return errors.New("There was an error pausing jobs.")
	}

	return nil
}

// Quit the queue
func (q *Queue) Quit() []common.Job {
	log.Debug("Attempting to quit the entire queue.")

	// First order of business is to kill the keeper
	log.Info("Quitting Keeper")
	if q.qk != nil {
		q.qk <- true
		log.Debug("Kill message sent")
	}

	q.Lock()
	defer q.Unlock()

	q.qk = nil

	// Now we need to be 100% up-to-date
	q.updateQueue()

	// Loop through and quit any job that is not done, failed, quit
	log.Debug("Looping through stack")
	for i, _ := range q.stack {
		joblog := log.WithFields(log.Fields{
			"resource":  q.stack[i].ResAssigned,
			"job":       q.stack[i].UUID,
			"jobstatus": q.stack[i].Status,
		})
		joblog.Debug("Looping through all jobs and quitting.")

		s := q.stack[i].Status

		// If the job is running quit it
		if s == common.STATUS_RUNNING && s == common.STATUS_PAUSED {
			// Build the quit call
			quitJob := common.RPCCall{
				Auth: q.pool[q.stack[i].ResAssigned].RPCCall.Auth,
				Job:  q.stack[i],
			}

			joblog.Debug("Quiting tasks")
			err := q.pool[q.stack[i].ResAssigned].Client.Call("Queue.TaskQuit", quitJob, &q.stack[i])
			// Log any errors but we don't care from a flow perspective
			if err != nil {
				log.Error(err.Error())
			}
		}
	}

	// Get rid of all the resource
	for i, _ := range q.pool {
		log.WithField("resource", q.pool[i].Name).Info("Stopping resource.")
		q.pool[i].Client.Close()
		delete(q.pool, i)
	}

	// We have looped through all jobs so return the last status
	// The Queue should be deleted or nulled from outside this context
	return q.stack
}

// The Keeper runs in a different goroutine to keep the queue roughly up-to-date
// It will need to aquire a lock each time it does this
// The q.qk channel needs to be created and maintained outside of this function
func (q *Queue) keeper() {
	log.Debug("Starting keeper loop.")
	go func() {
	keeperLoop:
		for {
			// Setup timer for keeper
			kTimer := time.After(KeeperDuration)

			select {
			case <-kTimer:
				log.Info("Updating queue status and keeping jobs.")

				q.Lock()

				//Write our state file
				if StateFileLocation != "" {
					q.writeState()
				}

				// Get lock
				// Update all running jobs
				q.updateQueue()

				// Look for open resources
				for i, _ := range q.pool {
					// Check that the resource is running
					if q.pool[i].Status == common.STATUS_RUNNING {
						for r, b := range q.pool[i].Hardware {
							if b {
								log.WithFields(log.Fields{
									"resource": q.pool[i].Name,
									"hardware": r,
								}).Debug("Found empty resource hardware")
								// This resource is free, so lets find a job for it
								for ji, _ := range q.stack {
									logger := log.WithFields(log.Fields{
										"resource": q.pool[i].Name,
										"job":      q.stack[ji].UUID,
									})

									requirement := q.pool[i].Tools[q.stack[ji].ToolUUID].Requirements
									tool, ok := q.pool[i].Tools[q.stack[ji].ToolUUID]

									s := q.stack[ji].Status

									startable := s == common.STATUS_CREATED
									resumable := s == common.STATUS_PAUSED

									if requirement == r && resumable && ok {
										// The job is paused so let's see if the resource that is free is the one the job needs
										logger.Debug("Attempting to resume job.")

										if q.stack[ji].ResAssigned == i {
											// We have the resource the job needs so resume it

											resumeJob := common.RPCCall{
												Auth: q.pool[i].RPCCall.Auth,
												Job:  q.stack[ji],
											}

											logger.Debug("Calling Queue.TaskRun to resume job.")
											err := q.pool[i].Client.Call("Queue.TaskRun", resumeJob, &q.stack[ji])
											if err != nil {
												// We had an error resuming the job
												logger.WithField("error", err.Error()).Error("Error while attempting to resume job on remote resource.")
											} else {
												// No errors so mark the resource as used
												q.pool[i].Hardware[r] = false
												break
											}
										}
									}

									if requirement == r && startable && ok {
										// Found a task that can be started and needs this resource
										// Build the call
										logger.Debug("Attempting to start job.")

										// It is now possible that the job about to be assigned to the resource
										// actually has a Tool UUID different than the one the Queue has for this
										// tool. This is because the Queue changes the UUID if it has two identical
										// tools. We need to set the Job's ToolUUID to the real UUID of the tool
										// if this is the case
										if tool.UUID != q.stack[ji].ToolUUID {
											q.stack[ji].ToolUUID = tool.UUID
										}

										startJob := common.RPCCall{
											Auth: q.pool[i].RPCCall.Auth,
											Job:  q.stack[ji],
										}

										logger.Debug("Calling Queue.AddTask to resume job.")
										err := q.pool[i].Client.Call("Queue.AddTask", startJob, &q.stack[ji])
										if err != nil {
											// something went wrong so log it, but the keeper will get this
											// resource the next time through
											logger.WithField("error", err.Error()).Error("Error while attempting to resume job on remote resource.")
										} else {
											// there were no errors so lets take up the hardware resource
											q.stack[ji].ResAssigned = i
											q.pool[i].Hardware[r] = false
											break
										}
									}
								}
							}
						}
					}
				}

				// Release the Lock
				q.Unlock()
			case <-q.qk:
				log.Debug("Keeper has been quit.")
				break keeperLoop
			}
		}
		log.Info("Queue keeper has successfully stopped.")
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
				log.WithField("rpc error", err.Error()).Error("Error during RPC call.")
			}

			// Check if this is now no longer running
			if q.stack[i].Status != common.STATUS_RUNNING {
				// Release the resources from this change
				log.WithField("JobID", q.stack[i].UUID).Debug("Job has finished.")
				var hw string
				for _, v := range q.pool[q.stack[i].ResAssigned].Tools {
					if v.UUID == q.stack[i].ToolUUID {
						hw = v.Requirements
					}
				}
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

// This function allows you to get tools that can actively have jobs created for them
func (q *Queue) ActiveTools() map[string]common.Tool {
	q.RLock()
	defer q.RUnlock()

	// Cycle through all the attached resources for unique tools
	var tools = make(map[string]common.Tool)
	for _, res := range q.pool {
		// Check if the tool is active for jobs (AKA running or paused)
		if res.Status != common.STATUS_QUIT {
			// Resource is paused or running so get the tools it provides
			for uuid, t := range res.Tools {
				// Check if tool already exists in the tools map
				_, ok := tools[uuid]
				if !ok {
					// Tool doesn't exit already so add it
					tools[uuid] = t
				}
			}
		}
	}

	return tools
}

// This function is used to get all tools that have ever been available
func (q *Queue) AllTools() map[string]common.Tool {
	q.RLock()
	defer q.RUnlock()

	// Cycle through all the attached resources for unique tools
	var tools = make(map[string]common.Tool)
	for _, res := range q.pool {
		// Get all tools regardless of active resources
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

func (q *Queue) AddResource(addr, name, auth string) error {
	// Create empty resource
	res := NewResource()

	//Check to see if we have a port, otherwise use the default 9443
	if !strings.Contains(addr, ":") {
		addr += ":9443"
	}

	// Check that the address is already in use
	for _, v := range q.pool {
		if v.Address == addr {
			// We have found a resource with the same address so error
			log.WithField("address", addr).Debug("Resource already exists.")
			return errors.New("Resource already exists!")
		}
	}

	log.Printf("Connecting to resource %s\n", addr)

	// Build the RPC client for the resource
	var err error
	res.Client, err = rpc.Dial("tcp", addr)
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
	var tools []common.Tool
	res.Client.Call("Queue.ResourceTools", res.RPCCall, &tools)

	for _, v := range tools {
		res.Tools[v.UUID] = v
	}

	q.Lock()
	defer q.Unlock()

	// Loop through new tools and look for those we already have
	// If we find a tool we already have we need to make the UUID the same
	// If we don't already have this we can just leave the UUID assigned by the resource
	for _, aValue := range res.Tools {
		// Loop through the pool
	ToolBreak:
		for i, _ := range q.pool {
			for _, cValue := range q.pool[i].Tools {
				if common.CompareTools(aValue, cValue) {
					res.Tools[cValue.UUID] = aValue
					delete(res.Tools, aValue.UUID)
					break ToolBreak
				}
			}
		}
	}

	res.Name = name
	res.Address = addr
	res.Status = common.STATUS_RUNNING

	// Add resource to resource pool with generated UUID
	q.pool[uuid.New()] = res

	return nil
}

func (q *Queue) GetResources() []common.Resource {
	q.Lock()
	defer q.Unlock()

	var resources []common.Resource
	for id, v := range q.pool {
		r := common.Resource{}
		r.UUID = id
		r.Name = v.Name
		r.Address = v.Address
		r.Tools = v.Tools
		r.Status = v.Status
		r.Hardware = v.Hardware

		resources = append(resources, r)
	}

	return resources
}

// RemoveResource closes the resource RPC client, and removes it from service.
// It does not delete it however, because that information is needed by the API
// even after it is no longer in service.
func (q *Queue) RemoveResource(resUUID string) error {
	// Check for the resource with given UUID
	_, ok := q.pool[resUUID]
	if !ok {
		return errors.New("Given Resource UUID does not exist.")
	}

	// Lock the queue
	q.Lock()
	defer q.Unlock()

	// Loop through any jobs assigned to the resource and quit them if they are not completed
	for i, v := range q.stack {
		if v.ResAssigned == resUUID {
			// Check status
			if v.Status == common.STATUS_RUNNING || v.Status == common.STATUS_PAUSED {
				// Quit the task
				quitTask := common.RPCCall{
					Auth: q.pool[resUUID].RPCCall.Auth,
					Job:  v,
				}

				err := q.pool[resUUID].Client.Call("Queue.TaskQuit", quitTask, &q.stack[i])
				if err != nil {
					log.Println(err.Error())
				}
			}
		}
	}

	// Close the connection to the client
	q.pool[resUUID].Client.Close()

	// Remove information that might affect additional resource adding
	res, _ := q.pool[resUUID]
	res.Address = "closed"
	res.Status = common.STATUS_QUIT
	q.pool[resUUID] = res
	for i, _ := range q.pool[resUUID].Hardware {
		q.pool[resUUID].Hardware[i] = false
	}

	return nil
}
