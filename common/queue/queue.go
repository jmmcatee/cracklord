package queue

import (
	"github.com/pborman/uuid"
	"crypto/tls"
	"encoding/json"
	"errors"
	log "github.com/Sirupsen/logrus"
	"github.com/emperorcow/protectedmap"
	"github.com/jmmcatee/cracklord/common"
	"io"
	"net"
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
	status   string // Empty, Running, Paused, Exhausted
	pool     ResourcePool
	stack    []common.Job
	managers protectedmap.ProtectedMap
	stats    Stats
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
		status:   STATUS_EMPTY,
		pool:     NewResourcePool(),
		stack:    []common.Job{},
		managers: protectedmap.New(),
		stats:    NewStats(),
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

	s.Stack = make([]common.Job, len(q.stack))
	copy(s.Stack, q.stack)

	s.Pool = make(map[string]Resource)
	for k, v := range q.pool {
		s.Pool[k] = v
	}

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

		for tool, _ := range v.Tools {
			delete(v.Tools, tool)
		}

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
				addJob := common.RPCCall{Job: j}

				logger.Debug("Queue.AddTask RPC call started.")
				err := q.pool[i].Client.Call("Queue.AddTask", addJob, &j)
				if err != nil {
					logger.WithField("error", err.Error()).Error("There was a problem making an RPC call.")
					q.DeleteJobFromStackByIndex(jobIndex)
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

func (q *Queue) DeleteJobFromStackByIndex(idx int) {
	tmp := make([]common.Job, len(q.stack))
	copy(tmp, q.stack)
	q.stack = make([]common.Job, len(tmp)-1)
	q.stack = append(tmp[:idx], tmp[idx+1:]...)
}

// Get the full queue stack
func (q *Queue) AllJobs() []common.Job {
	log.Debug("Gathering all jobs from queue.")

	q.Lock()

	q.Unlock()
	return q.stack
}

// Get a list of all jobs assigned to a resource
func (q *Queue) AllJobsByResource(resourceid string) []common.Job {
	jobs := q.AllJobs()
	outJobs := make([]common.Job, 0)

	for _, job := range jobs {
		if job.ResAssigned == resourceid {
			outJobs = append(outJobs, job)
		}
	}

	return outJobs
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
				pauseJob := common.RPCCall{Job: q.stack[i]}

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
				// Find the real ToolUUID since the Job's might have changed (See AddJob)
				var tUUID, hw string
				for qUUID, tool := range q.pool[q.stack[i].ResAssigned].Tools {
					if q.stack[i].ToolUUID == tool.UUID {
						// We found the UUID of the tool is so store it
						tUUID = qUUID
					}
				}
				hw = q.pool[q.stack[i].ResAssigned].Tools[tUUID].Requirements
				q.pool[q.stack[i].ResAssigned].Hardware[hw] = true

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
				quitJob := common.RPCCall{Job: q.stack[i]}

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
				// Find the real ToolUUID since the Job's might have changed (See AddJob)
				var tUUID, hw string
				for qUUID, tool := range q.pool[q.stack[i].ResAssigned].Tools {
					if q.stack[i].ToolUUID == tool.UUID {
						// We found the UUID of the tool is so store it
						tUUID = qUUID
					}
				}
				hw = q.pool[q.stack[i].ResAssigned].Tools[tUUID].Requirements
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
			pauseJob := common.RPCCall{Job: q.stack[i]}

			err := q.pool[resUUID].Client.Call("Queue.TaskPause", pauseJob, q.stack[i])
			if err != nil {
				return err
			}

			// Task should now be paused to free up the resource
			// Find the real ToolUUID since the Job's might have changed (See AddJob)
			var tUUID, hw string
			for qUUID, tool := range q.pool[q.stack[i].ResAssigned].Tools {
				if q.stack[i].ToolUUID == tool.UUID {
					// We found the UUID of the tool is so store it
					tUUID = qUUID
				}
			}
			hw = q.pool[q.stack[i].ResAssigned].Tools[tUUID].Requirements
			q.pool[q.stack[i].ResAssigned].Hardware[hw] = true
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

	// Let's run the keep functions on all of our resource managers
	q.KeepAllResourceManagers()

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
			resuuid := q.stack[i].ResAssigned

			// This task is running and needs to be paused
			pauseJob := common.RPCCall{Job: q.stack[i]}

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
			// Find the real ToolUUID since the Job's might have changed (See AddJob)
			var tUUID, hw string
			for qUUID, tool := range q.pool[q.stack[i].ResAssigned].Tools {
				if q.stack[i].ToolUUID == tool.UUID {
					// We found the UUID of the tool is so store it
					tUUID = qUUID
				}
			}
			hw = q.pool[q.stack[i].ResAssigned].Tools[tUUID].Requirements
			q.pool[q.stack[i].ResAssigned].Hardware[hw] = true
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
			quitJob := common.RPCCall{Job: q.stack[i]}

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

				// Run all resource manager keep routines
				q.KeepAllResourceManagers()

				// Get lock
				q.Lock()

				// Update all running jobs
				q.updateQueue()

				//Write our state file
				if StateFileLocation != "" {
					q.writeState()
				}

				// Look for open resources
				//ResourceLoop:
				for resKey, _ := range q.pool {
					// Check that the resource is running
					if q.pool[resKey].Status == common.STATUS_RUNNING {
						// Loop through hardware the resouce offers (CPU, GPU, etc.)
					HardwareLoop:
						for hardwareKey, hardwareFree := range q.pool[resKey].Hardware {
							// If the hardware is free
							if hardwareFree {
								log.WithFields(log.Fields{
									"resource": q.pool[resKey].Name,
									"hardware": hardwareKey,
								}).Debug("Found empty resource hardware")

								// This resource is free, so lets find a job for it
							JobLoop:
								for jobKey, _ := range q.stack {
									logger := log.WithFields(log.Fields{
										"resource": q.pool[resKey].Name,
										"job":      q.stack[jobKey].UUID,
									})

									// Are we looking to start or resume the job?
									switch q.stack[jobKey].Status {
									case common.STATUS_CREATED: // We are going to start the job fresh
										// We first need to check if this tool exists on this resource
										if tool, ok := q.pool[resKey].Tools[q.stack[jobKey].ToolUUID]; ok {
											// We now need to get the hardware requirements for this tool
											if q.pool[resKey].Tools[q.stack[jobKey].ToolUUID].Requirements == hardwareKey {
												// We now know we have an open resource and a job that needs that resource
												logger.Debug("Attempting to start new job on resource")

												// It is now possible that the Tool UUID on the resource does not match
												// the Tool UUID the job and Queue have. This is because the Queue changes
												// all UUIDs (keys in the pool map) to make identical Tools across multiple
												// resources show up as one Tool available to the system. We now need to set
												// the Job's ToolUUID field to the correct UUID for the resource if this has
												// happened.
												if tool.UUID != q.stack[jobKey].ToolUUID {
													q.stack[jobKey].ToolUUID = tool.UUID
												}

												logger.Debug("Calling Queue.AddTask to start the job.")
												err := q.pool[resKey].Client.Call("Queue.AddTask", common.RPCCall{Job: q.stack[jobKey]}, &q.stack[jobKey])
												if err != nil {
													// Something failed so let's mark the job as failed
													logger.WithField("error", err.Error()).Error("Error while attempting to start job on remote resource.")
													q.stack[jobKey].Status = common.STATUS_FAILED
													continue JobLoop
												}

												// Job has been started so mark the hardware as in use and assign the resource ID
												q.stack[jobKey].ResAssigned = resKey
												q.pool[resKey].Hardware[hardwareKey] = false
												break HardwareLoop
											}
										}
									case common.STATUS_PAUSED: // We are going to resume the job were it is
										// We are resuming a job so we first need to check if the job was assigned to this resource
										if q.stack[jobKey].ResAssigned == resKey {
											// This job was assigned to this resource so we need to find the correct local UUID of the tool
											for _, resTool := range q.pool[resKey].Tools {
												if resTool.UUID == q.stack[jobKey].ToolUUID {
													// We have found the correct UUID, so check if this is the available hardware
													if resTool.Requirements == hardwareKey {
														// The job requires the hardware that is available on this resource to resume
														logger.Debug("Attempting to resume job.")

														err := q.pool[resKey].Client.Call("Queue.TaskRun", common.RPCCall{Job: q.stack[jobKey]}, &q.stack[jobKey])
														if err != nil {
															// Something failed so let's mark the job as failed
															logger.WithField("error", err.Error()).Error("Error while attempting to resume job on remote resource.")
															q.stack[jobKey].Status = common.STATUS_FAILED
															continue JobLoop
														}

														// Job has been started so mark the hardware as in use
														q.pool[resKey].Hardware[hardwareKey] = false
														break HardwareLoop
													}
												}
											}
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
			jobStatus := common.RPCCall{Job: q.stack[i]}

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

// This function is used to gather all currently available resource managers
// and provide them back to the API or any other needed functions.
func (q *Queue) AllResourceManagers() map[string]ResourceManager {
	//We'll make a slice to hold our resource managers
	managers := make(map[string]ResourceManager, q.managers.Count())

	// Now we'll create a loop on our protected map
	i := q.managers.Iterator()

	// Loop through each one, with the manager variable being a standard tuple
	for manager := range i.Loop() {
		// Convert our generic interface from the protectedmap into the right type
		mgrtype := (manager.Val).(ResourceManager)
		// Get the data into our temporary map
		managers[manager.Key] = mgrtype
	}

	//Return the results
	return managers
}

// This function will return a copy of a single resource manager from the Queue
// and all of the associated etails.  It takes a parameter of the system name of
// the manager desired.
func (q *Queue) GetResourceManager(systemname string) (ResourceManager, bool) {
	manager, ok := q.managers.Get(systemname)
	if ok == true {
		mgrtype := manager.(ResourceManager)
		return mgrtype, ok
	} else {
		return nil, ok
	}
}

// This function is used to add a resource manager to the map of all available
// managers and is used to add resourcemanager plugins during their creation.
func (q *Queue) AddResourceManager(resmgr ResourceManager) error {
	//Get the ID of the manager we're adding
	id := resmgr.SystemName()

	//Let's check and see if it already exists, if so we should error
	if _, ok := q.managers.Get(id); ok {
		log.WithField("id", id).Error("ResourceManager cannot be added twice.")
		return errors.New("ResourceManager cannot be added twice.")
	}

	//Otherwise, set our resource manager and be done with it.
	q.managers.Set(id, resmgr)

	//Log that we did it, because that's just good practice.
	log.WithField("id", id).Info("Added resource manager into the queue.")

	//Return with no error.
	return nil
}

// This function will loop through all resource managers and executes thier keeper
// functions, causing them to update the status for all of their resources.
// The actions of each will be dependent on the resource manager.
func (q *Queue) KeepAllResourceManagers() {
	log.Debug("ResourceManager keep loop starting.")

	//First we need to setup an iterator to loop through the entire map and then loop
	i := q.managers.Iterator()
	for manager := range i.Loop() {
		// First we need to convert our resource manager over to the proper type
		mgrtype := (manager.Val).(ResourceManager)

		log.WithField("manager", mgrtype.SystemName()).Debug("Running resource manager keep function")

		// Now for each resource manager, let's call it's Keep() function
		mgrtype.Keep()
	}

	log.Debug("ResourceManager keep loop complete.")
}

//This function will connect to a resource
func (q *Queue) ConnectResource(resUUID, addr string, tlsconfig *tls.Config) error {
	q.RLock()
	localRes := q.pool[resUUID]
	q.RUnlock()

	// First, setup the address we're going to connect to
	localRes.Address = addr
	// Then store a local version in the event we need to add the default port
	target := localRes.Address

	//Check to see if we have a port, otherwise use the default 9443
	if !strings.Contains(target, ":") {
		target += ":9443"
	}
	log.WithField("addr", target).Info("Connecting to resource")

	// Dial the target and see if we get a connection in 15 seconds
	/*
		conn, err := net.DialTimeout("tcp", target, time.Second*15)
		if err != nil {
			log.WithField("addr", target).Debug("Unable to dial the resource.")
			return err
		}

		// Now we need to set the ServerName in the tls config.  We'll make a copy
		// to make sure we don't mess with anything
		localConfig := *tlsconfig
		localConfig.ServerName = localRes.Address

		// Now let's build a TLS connection object and force a handshake to make
		// sure it's working
		tlsConn := tls.Client(conn, &localConfig)
		err = tlsConn.Handshake()
		if err != nil {
			log.WithFields(log.Fields{
				"addr":       target,
				"servername": localRes.Address,
			}).Debug("An error occured while building the TLS connection")
			return err
		}
	*/

	dialer := &net.Dialer{
		Timeout: 15 * time.Second,
	}

	conn, err := tls.DialWithDialer(dialer, "tcp", target, tlsconfig)
	if err != nil {
		log.WithFields(log.Fields{
			"addr":       target,
			"servername": localRes.Address,
		}).Debug("An error occured while building the TLS connection")
		return err
	}

	// Build the RPC client for the resource
	localRes.Client = rpc.NewClient(conn)
	if err != nil {
		log.WithField("addr", target).Debug("An error occured while creating new client")
		return err
	}

	// Let the user know we connected
	log.WithField("target", localRes.Address).Info("Successfully connected to resource")
	localRes.Status = common.STATUS_RUNNING

	q.Lock()
	q.pool[resUUID] = localRes
	q.Unlock()

	// Now let's make sure the tools and hardware are loaded
	q.LoadRemoteResourceHardware(resUUID)
	q.LoadRemoteResourceTools(resUUID)

	return nil
}

//Checks to see if our RPC connection to a resource is still valid, if not it
//will return false, otherwise it will return true.
func (q *Queue) CheckResourceConnectionStatus(res *Resource) bool {
	var reply int64
	err := res.Client.Call("Queue.Ping", 12345, &reply)
	if err == rpc.ErrShutdown || err == io.EOF || err == io.ErrUnexpectedEOF {
		return false
	}

	return true
}

//This loads all of the hardware for a remote resource
func (q *Queue) LoadRemoteResourceHardware(resUUID string) {
	q.RLock()
	localRes := q.pool[resUUID]
	q.RUnlock()

	// Get Hardware
	err := localRes.Client.Call("Queue.ResourceHardware", common.RPCCall{}, &localRes.Hardware)
	if err != nil {
		log.WithFields(log.Fields{
			"error":    err.Error(),
			"resource": resUUID,
		}).Error("Unable to gather resource hardware.")
		return
	}

	// Set all hardware as available
	for key, _ := range localRes.Hardware {
		localRes.Hardware[key] = true
	}

	q.Lock()
	q.pool[resUUID] = localRes
	q.Unlock()

	log.WithField("resources", resUUID).Debug("Loaded hardware for resource")
}

func (q *Queue) LoadRemoteResourceTools(resUUID string) {
	q.RLock()
	localRes := q.pool[resUUID]
	q.RUnlock()

	// Get Tools
	var tools []common.Tool
	err := localRes.Client.Call("Queue.ResourceTools", common.RPCCall{}, &tools)
	if err != nil {
		log.WithFields(log.Fields{
			"error":    err.Error(),
			"resource": resUUID,
		}).Error("Unable to gather resource tools.")
		return
	}

	for _, v := range tools {
		localRes.Tools[v.UUID] = v
	}

	q.RLock()
	// Loop through new tools and look for those we already have
	// If we find a tool we already have we need to make the UUID the same
	// If we don't already have this we can just leave the UUID assigned by the resource
	for _, aValue := range localRes.Tools {
		// Loop through the pool
	ToolBreak:
		for i, _ := range q.pool {
			for _, cValue := range q.pool[i].Tools {
				if i == resUUID {
					continue
				}
				if common.CompareTools(aValue, cValue) {
					localRes.Tools[cValue.UUID] = aValue
					delete(localRes.Tools, aValue.UUID)
					break ToolBreak
				}
			}
		}
	}
	q.RUnlock()

	q.Lock()
	q.pool[resUUID] = localRes
	q.Unlock()

	log.WithField("resource", resUUID).Debug("Loaded tools for resource")
}

//This function will add a resource to the queue.  Returns the UUID.
func (q *Queue) AddResource(name string) (string, error) {
	// Check that the address is already in use
	for _, v := range q.pool {
		if v.Name == name && v.Status != common.STATUS_QUIT {
			// We have found a resource with the same address so error
			log.WithField("name", name).Debug("Resource already exists.")
			return "", errors.New("Resource already exists!")
		}
	}

	// Create empty resource
	res := NewResource()

	res.Name = name
	res.Status = common.STATUS_PENDING

	//Generate a UUID for the resource
	resourceuuid := uuid.New()

	// Add resource to resource pool with generated UUID
	q.Lock()
	q.pool[resourceuuid] = res
	q.Unlock()

	return resourceuuid, nil
}

func (q *Queue) GetResource(resUUID string) (*Resource, bool) {
	log.WithField("resourceid", resUUID).Debug("Gathering data on resource.")
	q.Lock()
	defer q.Unlock()

	res, ok := q.pool[resUUID]
	if !ok {
		return &Resource{}, false
	}
	log.WithField("resourceid", resUUID).Debug("Found resource.")
	return &res, ok
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
				quitTask := common.RPCCall{Job: v}

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
