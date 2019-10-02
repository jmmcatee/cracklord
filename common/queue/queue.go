package queue

import (
	"crypto/tls"
	"errors"
	"net"
	"net/rpc"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/emperorcow/protectedmap"
	"github.com/jmmcatee/cracklord/common"
	"github.com/pborman/uuid"
)

const (
	STATUS_EMPTY     = "Empty"
	STATUS_RUNNING   = "Running"
	STATUS_PAUSED    = "Paused"
	STATUS_EXHAUSTED = "Exhausted"
)

var KeeperDuration time.Duration // Duration between the Queue keeper executions
var NetworkTimeout time.Duration // Timeout for networking connections
var Hooks HookParameters         // Parameter to pass to hook calls

// Queue is the primary structure of the Queue package
type Queue struct {
	status string // Empty, Running, Paused, Exhausted
	pool   ResourcePool
	//stack    []common.Job
	db       *JobDB
	managers protectedmap.ProtectedMap
	stats    Stats
	jpurge   int
	sync.RWMutex
	qk chan bool
}

// NewQueue returns a new instance of the Queue structure initialized
func NewQueue(updatetime int, timeout int, hooks HookParameters, purgetime int, jdb *JobDB) *Queue {
	//Setup the options
	KeeperDuration = time.Duration(updatetime) * time.Second
	NetworkTimeout = time.Duration(timeout) * time.Second
	Hooks = hooks

	// Build the queue
	q := Queue{
		status: STATUS_EMPTY,
		pool:   NewResourcePool(),
		//stack:    []common.Job{},
		db:       jdb,
		managers: protectedmap.New(),
		stats:    NewStats(),
		jpurge:   purgetime,
	}

	log.WithFields(log.Fields{
		"bboltdb":    q.db.boltdb.GoString(),
		"keepertime": KeeperDuration,
		"nettimeout": NetworkTimeout,
	}).Debug("Setup a new queue")

	return &q
}

// AddJob adds a job to the queue at the end of the stack
func (q *Queue) AddJob(j common.Job) error {
	logger := log.WithFields(log.Fields{
		"uuid":   j.UUID,
		"name":   j.Name,
		"params": common.CleanJobParamsForLogging(j),
	})

	// Lock the queue for the adding work
	q.Lock()
	defer q.Unlock()

	logger.Debug("Queue locked.")

	// Add job to stack
	err := q.db.AddJob(j)
	if err != nil {
		return err
	}

	// Call out to the registered hooks to inform them of job creation
	go HookOnJobCreate(Hooks.JobCreate, j)

	// Add stats
	// TODO: Add more stats
	q.stats.IncJob()

	// Check if the Queue was empty
	if q.status == STATUS_EMPTY {
		logger.Debug("Queue is empty, job needs starting.")
		// The Queue is empty so we need to start this job and the keeper

		// Start the keeper
		logger.Debug("Keeper started")
		q.qk = make(chan bool)
		go q.keeper()

		// We have started the keeper so change the status
		q.status = STATUS_RUNNING

		// Find the first open resource
		for i := range q.pool {
			logger.WithField("resource", q.pool[i].Name).Debug("Looking for resource")

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
					log.WithField("toolid", tool.UUID).Debug("Changed tool UUID to match resource")
					j.ToolUUID = tool.UUID
				}

				// Tool exist, lets start the job on this resource and assign the resource to the job
				j.ResAssigned = i
				addJob := common.RPCCall{Job: j}
				var retJob common.Job

				logger.Debug("Queue.AddTask RPC call started.")
				err := q.resCall(j.ResAssigned, q.pool[i].Client, "Queue.AddTask", addJob, &retJob)
				if err != nil {
					logger.Error(err)
					j.Status = common.STATUS_FAILED

					updateErr := q.db.UpdateJob(j)
					if updateErr != nil {
						logger.WithFields(log.Fields{
							"uuid":   j.UUID,
							"name":   j.Name,
							"params": common.CleanJobParamsForLogging(j),
							"action": "failed updating job after unsuccesful RPC call",
						}).Error(updateErr)

						// We could not update the job so try and delete it
						delErr := q.db.DeleteJob(j.UUID)
						if delErr != nil {
							logger.WithFields(log.Fields{
								"uuid":   j.UUID,
								"name":   j.Name,
								"params": common.CleanJobParamsForLogging(j),
								"action": "failed delete job after unsuccesful update",
							}).Error(delErr)
						}
					}

					return err
				}

				if common.IsEmpty(retJob) {
					// The RPC call returned and empty job... something bad has happened
					// so let's quit this job in the Queue
					log.WithFields(log.Fields{
						"jobuuid":       j.UUID,
						"jobname":       j.Name,
						"resource_uuid": j.ResAssigned,
					}).Error("task add returned an empty job so it has failed")

					j.Status = common.STATUS_FAILED

					err = q.db.UpdateJob(j)
					if err != nil {
						log.Error(err)
					}

					continue
				}

				log.WithFields(log.Fields{
					"uuid":   retJob.UUID,
					"name":   retJob.Name,
					"status": retJob.Status,
					"resid":  retJob.ResAssigned,
					"params": common.CleanJobParamsForLogging(retJob),
					"action": "failed updating job after succesful RPC call",
				}).Debug("Saving RPC returned Job to Queue")

				// Update the job in the stack
				err = q.db.UpdateJob(retJob)
				if err != nil {
					logger.WithFields(log.Fields{
						"uuid":   retJob.UUID,
						"name":   retJob.Name,
						"params": common.CleanJobParamsForLogging(retJob),
						"action": "failed updating job after succesful RPC call",
					}).Error(err)

					return err
				}

				// Note the resources as being used
				if retJob.Status === common.STATUS_RUNNING {
					q.pool[i].Hardware[tool.Requirements] = false
				}

				// Call out to our registered hooks to note job start
				go HookOnJobStart(Hooks.JobStart, retJob)

				// We should be done so return no errors
				return nil
			}

			// Tool did not exist... return error
			return errors.New("Tool did not exist for jobs provided")
		}
	}

	// If the queue is running or paused all we need to have done is add it to the queue
	return nil
}

// AllJobs gets the full queue from the database
func (q *Queue) AllJobs() []common.Job {
	log.Debug("Gathering all jobs from queue.")

	jobs, err := q.db.GetAllJobs()
	if err != nil {
		log.Error(err)
	}

	return jobs
}

// AllJobsByResource gets a list of all jobs assigned to a resource
func (q *Queue) AllJobsByResource(resourceid string) []common.Job {
	log.WithField("uuid", resourceid).Debug("Gathering all jobs from queue for the given resource")

	jobs, err := q.db.GetAllJobs()
	if err != nil {
		log.WithField("uuid", resourceid).Error(err)
	}

	var outJobs []common.Job
	for i := range jobs {
		if jobs[i].ResAssigned == resourceid {
			outJobs = append(outJobs, jobs[i])
		}
	}

	return outJobs
}

// JobInfo gets one specific job
func (q *Queue) JobInfo(jobUUID string) common.Job {
	log.WithField("uuid", jobUUID).Debug("Gathering information on a job")

	job, err := q.db.GetJob(jobUUID)
	if err != nil {
		log.Error(err)
	}
	return job
}

// PauseJob will attempt to pause a job by the given UUID
func (q *Queue) PauseJob(jobuuid string) error {
	log.WithField("job", jobuuid).Info("Attempting to pause job.")
	q.Lock()
	defer q.Unlock()

	job, err := q.db.GetJob(jobuuid)
	if err != nil {
		return err
	}

	// We have found the job so lets see if it running
	if job.Status == common.STATUS_RUNNING {
		// Job is running so lets tell it to pause
		pauseJob := common.RPCCall{Job: job}
		var retJob common.Job

		log.WithField("job", pauseJob.Job.UUID).Debug("Calling Queue.TaskPause on remote resource.")
		err := q.resCall(job.ResAssigned, q.pool[job.ResAssigned].Client, "Queue.TaskPause", pauseJob, &retJob)
		if err != nil {
			log.WithFields(log.Fields{
				"uuid":  pauseJob.Job.UUID,
				"error": err.Error(),
			}).Error("An error occurred while trying to pause a remote job")
			return err
		}

		if common.IsEmpty(retJob) {
			// The RPC call returned and empty job... something bad has happened
			// so let's quit this job in the Queue and try and quit it on the resource
			log.WithFields(log.Fields{
				"jobuuid":       job.UUID,
				"jobname":       job.Name,
				"resource_uuid": job.ResAssigned,
			}).Error("task pause returned an empty job so it has failed")

			job.Status = common.STATUS_FAILED

			err = q.db.UpdateJob(job)
			if err != nil {
				log.Error(err)
			}

			err := q.resCall(job.ResAssigned, q.pool[job.ResAssigned].Client, "Queue.TaskQuit", pauseJob, &retJob)
			if err != nil {
				// This probably means something really bad happened on the resource, so throw an error in the log
				log.Error(err)
				log.WithField("resource_uuid", job.ResAssigned).Error("resource might be dead or broken")
			}

			return errors.New("empty job returned by pause RPC call")
		}

		log.WithFields(log.Fields{
			"uuid":   retJob.UUID,
			"name":   retJob.Name,
			"resid":  retJob.ResAssigned,
			"params": common.CleanJobParamsForLogging(retJob),
		}).Debug("Saving RPC returned Job to Queue")

		// Task is now paused so update the resource
		// Find the real ToolUUID since the Job's might have changed (See AddJob)
		var tUUID, hw string
		for qUUID, tool := range q.pool[retJob.ResAssigned].Tools {
			if job.ToolUUID == tool.UUID {
				// We found the UUID of the tool is so store it
				tUUID = qUUID
			}
		}
		hw = q.pool[retJob.ResAssigned].Tools[tUUID].Requirements
		q.pool[retJob.ResAssigned].Hardware[hw] = true

		err = q.db.UpdateJob(retJob)

		return err
	}

	// The job was found but was not running so lets return an error
	return errors.New("Job given is not running. Current status is " + job.Status)
}

// QuitJob attempts to quit the job given a UUID
func (q *Queue) QuitJob(jobuuid string) error {
	log.WithField("job", jobuuid).Info("Attempting to pause job")

	q.Lock()
	defer q.Unlock()

	job, err := q.db.GetJob(jobuuid)
	if err != nil {
		return err
	}

	if job.Status != common.STATUS_DONE &&
		job.Status != common.STATUS_FAILED &&
		job.Status != common.STATUS_QUIT &&
		job.Status != common.STATUS_CREATED {
		// Lets build the call to stop the job
		quitJob := common.RPCCall{Job: job}
		var retJob common.Job

		log.WithField("uuid", quitJob.Job.UUID).Debug("Attempting to call Queue.TaskQuit on remote resource.")
		err := q.resCall(job.ResAssigned, q.pool[job.ResAssigned].Client, "Queue.TaskQuit", quitJob, &retJob)
		if err != nil {
			log.WithFields(log.Fields{
				"job":   quitJob.Job.UUID,
				"error": err.Error(),
			}).Error("An error occurred while trying to quit a remote job")

			return err
		}

		if common.IsEmpty(retJob) {
			// The RPC call returned and empty job... something bad has happened
			// so let's quit this job in the Queue and try and quit it on the resource
			log.WithFields(log.Fields{
				"jobuuid":       job.UUID,
				"jobname":       job.Name,
				"resource_uuid": job.ResAssigned,
			}).Error("task quit returned an empty job so it has failed")

			job.Status = common.STATUS_FAILED

			err = q.db.UpdateJob(job)
			if err != nil {
				log.Error(err)
			}

			err := q.resCall(job.ResAssigned, q.pool[job.ResAssigned].Client, "Queue.TaskQuit", quitJob, &retJob)
			if err != nil {
				// This probably means something really bad happened on the resource, so throw an error in the log
				log.Error(err)
				log.WithField("resource_uuid", job.ResAssigned).Error("resource might be dead or broken")
			}

			return errors.New("empty job returned by quit RPC call")
		}

		// Set a purge time
		retJob.PurgeTime = time.Now().Add(time.Duration(q.jpurge*24) * time.Hour)
		// Log purge time
		log.WithFields(log.Fields{
			"uuid":       retJob.UUID,
			"purge_time": retJob.PurgeTime,
		}).Debug("Updated PurgeTime value")

		log.WithFields(log.Fields{
			"uuid":   retJob.UUID,
			"name":   retJob.Name,
			"resid":  retJob.ResAssigned,
			"params": common.CleanJobParamsForLogging(retJob),
		}).Debug("Saving RPC returned Job to Queue")

		// Task has been quit without errors so update the available hardware and return
		// Find the real ToolUUID since the Job's might have changed (See AddJob)
		var tUUID, hw string
		for qUUID, tool := range q.pool[retJob.ResAssigned].Tools {
			if retJob.ToolUUID == tool.UUID {
				// We found the UUID of the tool is so store it
				tUUID = qUUID
			}
		}
		hw = q.pool[retJob.ResAssigned].Tools[tUUID].Requirements
		q.pool[retJob.ResAssigned].Hardware[hw] = true

		err = q.db.UpdateJob(retJob)
		return err
	}

	if job.Status == common.STATUS_CREATED {
		// We need to set the new status for the job to quit
		job.Status = common.STATUS_QUIT
		err := q.db.UpdateJob(job)
		return err
	}

	// The Jobs status is already stopped so lets return an error
	return errors.New("Job is already not running. Current status is " + job.Status)

}

// RemoveJob attempts to remove a job from the Queue database
func (q *Queue) RemoveJob(jobuuid string) error {
	log.WithField("uuid", jobuuid).Debug("Attempting to remove job")

	job, err := q.db.GetJob(jobuuid)
	if err != nil {
		return err
	}
	if job.Status == common.STATUS_RUNNING {
		// Quit the job
		err := q.QuitJob(job.UUID)
		if err != nil {
			return err
		}
	}

	// Remove the job
	return q.db.DeleteJob(job.UUID)
}

// PauseResource attempt to pause a connective resource so it will not continue processing jobs.
func (q *Queue) PauseResource(resUUID string) error {
	log.WithField("resource", resUUID).Debug("Attempting to pause resource")

	q.Lock()
	defer q.Unlock()

	// Check for UUID existance
	if _, ok := q.pool[resUUID]; !ok {
		return errors.New("Resource with UUID provided does not exist")
	}

	jobs, err := q.db.GetAllJobs()
	if err != nil {
		return err
	}

	// Loop through and pause any tasks running on the selected resource
	for i := range jobs {
		log.WithFields(log.Fields{
			"resuuid":   jobs[i].ResAssigned,
			"jobuuid":   jobs[i].UUID,
			"jobstatus": jobs[i].Status,
		}).Debug("Identifying running jobs on paused resource")

		if jobs[i].ResAssigned == resUUID && jobs[i].Status == common.STATUS_RUNNING {
			// We found a task that is running so lets pause it
			pauseJob := common.RPCCall{Job: jobs[i]}
			var retJob common.Job

			err := q.resCall(resUUID, q.pool[resUUID].Client, "Queue.TaskPause", pauseJob, &retJob)
			if err != nil {
				return err
			}

			if common.IsEmpty(retJob) {
				// The RPC call returned and empty job... something bad has happened
				// so let's quit this job in the Queue and try and quit it on the resource
				log.WithFields(log.Fields{
					"jobuuid":       jobs[i].UUID,
					"jobname":       jobs[i].Name,
					"resource_uuid": jobs[i].ResAssigned,
				}).Error("task pausing returned an empty job so it has failed")

				jobs[i].Status = common.STATUS_FAILED

				err = q.db.UpdateJob(jobs[i])
				if err != nil {
					log.Error(err)
				}

				err := q.resCall(jobs[i].ResAssigned, q.pool[jobs[i].ResAssigned].Client, "Queue.TaskPause", pauseJob, &retJob)
				if err != nil {
					// This probably means something really bad happened on the resource, so throw an error in the log
					log.Error(err)
					log.WithField("resource_uuid", jobs[i].ResAssigned).Error("resource might be dead or broken")
				}

				continue
			}

			log.WithFields(log.Fields{
				"uuid":   retJob.UUID,
				"name":   retJob.Name,
				"resid":  retJob.ResAssigned,
				"params": common.CleanJobParamsForLogging(retJob),
			}).Debug("Saving RPC returned Job to Queue")

			// Task should now be paused to free up the resource
			// Find the real ToolUUID since the Job's might have changed (See AddJob)
			var tUUID, hw string
			for qUUID, tool := range q.pool[retJob.ResAssigned].Tools {
				if retJob.ToolUUID == tool.UUID {
					// We found the UUID of the tool is so store it
					tUUID = qUUID
				}
			}
			hw = q.pool[retJob.ResAssigned].Tools[tUUID].Requirements
			q.pool[retJob.ResAssigned].Hardware[hw] = true

			err = q.db.UpdateJob(retJob)
			if err != nil {
				return err
			}
		}
	}

	// All tasks that would be running should now be paused so lets pause the resource
	res, _ := q.pool[resUUID]
	res.Status = common.STATUS_PAUSED
	q.pool[resUUID] = res

	return nil
}

// ResumeResource attempt to start a resource so it can process jobs
func (q *Queue) ResumeResource(resUUID string) error {
	log.WithField("resource", resUUID).Debug("Attempting to resume resource")

	q.Lock()
	defer q.Unlock()

	// Check for UUID existance
	if _, ok := q.pool[resUUID]; !ok {
		return errors.New("Resource with UUID provided does not exist")
	}

	if q.pool[resUUID].Status != common.STATUS_PAUSED {
		return errors.New("Resource is not paused")
	}

	// Pool exists so unpause it
	res, _ := q.pool[resUUID]
	res.Status = common.STATUS_RUNNING
	q.pool[resUUID] = res

	// The keeper will take it from here
	return nil
}

// PauseQueue attempts to pause the whole queue and return any and all errors pausing active jobs/tasks
func (q *Queue) PauseQueue() []error {
	log.Debug("Attempting to pause entire queue")

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
	jobs, err := q.db.GetAllJobs()
	if err != nil {
		return []error{err}
	}

	for i := range jobs {
		joblog := log.WithFields(log.Fields{
			"resource":  jobs[i].ResAssigned,
			"job":       jobs[i].UUID,
			"jobstatus": jobs[i].Status,
		})
		joblog.Debug("Processing job.")

		if jobs[i].Status == common.STATUS_RUNNING {
			joblog.Debug("Found running job, attempting to stop")

			// Get some helpful values
			resuuid := jobs[i].ResAssigned

			// This task is running and needs to be paused
			pauseJob := common.RPCCall{Job: jobs[i]}
			var retJob common.Job

			joblog.Debug("Calling Queue.TaskPause on job")
			err := q.resCall(resuuid, q.pool[resuuid].Client, "Queue.TaskPause", pauseJob, &retJob)
			if err != nil {
				// Note the error but now mark the job as Failed
				// This is a definied way of dealing with this to avoid complicated error handling
				jobs[i].Status = common.STATUS_FAILED
				jobs[i].Error = err.Error()
				e = append(e, err)

				joblog.Debug("There was a problem pausing the remote job.")
			}

			if common.IsEmpty(retJob) {
				// The RPC call returned and empty job... something bad has happened
				// so let's quit this job in the Queue and try and quit it on the resource
				log.WithFields(log.Fields{
					"jobuuid":       jobs[i].UUID,
					"jobname":       jobs[i].Name,
					"resource_uuid": jobs[i].ResAssigned,
				}).Error("task pausing returned an empty job so it has failed")

				jobs[i].Status = common.STATUS_FAILED

				err = q.db.UpdateJob(jobs[i])
				if err != nil {
					log.Error(err)
				}

				err := q.resCall(jobs[i].ResAssigned, q.pool[jobs[i].ResAssigned].Client, "Queue.TaskQuit", pauseJob, &retJob)
				if err != nil {
					// This probably means something really bad happened on the resource, so throw an error in the log
					log.Error(err)
					log.WithField("resource_uuid", jobs[i].ResAssigned).Error("resource might be dead or broken")
				}

				// Update available hardware
				// Find the real ToolUUID since the Job's might have changed (See AddJob)
				var tUUID, hw string
				for qUUID, tool := range q.pool[jobs[i].ResAssigned].Tools {
					if jobs[i].ToolUUID == tool.UUID {
						// We found the UUID of the tool is so store it
						tUUID = qUUID
					}
				}
				hw = q.pool[jobs[i].ResAssigned].Tools[tUUID].Requirements
				q.pool[jobs[i].ResAssigned].Hardware[hw] = true

				continue
			}

			log.WithFields(log.Fields{
				"uuid":   retJob.UUID,
				"name":   retJob.Name,
				"resid":  retJob.ResAssigned,
				"params": common.CleanJobParamsForLogging(retJob),
			}).Debug("Saving RPC returned Job to Queue")

			// Update available hardware
			// Find the real ToolUUID since the Job's might have changed (See AddJob)
			var tUUID, hw string
			for qUUID, tool := range q.pool[retJob.ResAssigned].Tools {
				if retJob.ToolUUID == tool.UUID {
					// We found the UUID of the tool is so store it
					tUUID = qUUID
				}
			}
			hw = q.pool[retJob.ResAssigned].Tools[tUUID].Requirements
			q.pool[retJob.ResAssigned].Hardware[hw] = true

			err = q.db.UpdateJob(retJob)
			if err != nil {
				e = append(e, err)
			}
		}
	}

	// All jobs/tasks should now be paused so lets set the Queue Status
	q.status = STATUS_PAUSED
	log.Debug("Queue paused")

	return e
}

// ResumeQueue attempts to restart the Queue
func (q *Queue) ResumeQueue() {
	log.Debug("Attempting to restart the queue")
	q.Lock()
	defer q.Unlock()

	if q.status == STATUS_PAUSED {

		// Change status and start the keeper
		q.status = STATUS_RUNNING
		q.qk = make(chan bool)
		go q.keeper()
	}

	return
}

// StackReorder given a new slice of the UUIDs for all jobs in order, reorder the stack
func (q *Queue) StackReorder(uuids []string) error {
	es := q.PauseQueue()
	if len(es) != 0 {
		return errors.New("Error pausing Queue")
	}
	q.Lock()

	err := q.db.ReorderJobs(uuids)
	if err != nil {
		q.Unlock()
		return err
	}

	// Resume the Queue
	q.Unlock()
	q.ResumeQueue()

	jobs, err := q.db.GetAllJobs()
	if err != nil {
		return err
	}

	go HookOnQueueReorder(Hooks.QueueReorder, jobs)

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
	jobs, err := q.db.GetAllJobs()
	if err != nil {
		log.Error(err)
	}

	for i := range jobs {
		joblog := log.WithFields(log.Fields{
			"resource":  jobs[i].ResAssigned,
			"job":       jobs[i].UUID,
			"jobstatus": jobs[i].Status,
		})
		joblog.Debug("Looping through all jobs and quitting.")

		s := jobs[i].Status

		// If the job is running quit it
		if s == common.STATUS_RUNNING || s == common.STATUS_PAUSED {
			// Build the quit call
			quitJob := common.RPCCall{Job: jobs[i]}
			var retJob common.Job

			joblog.Debug("Quiting tasks")
			err := q.resCall(jobs[i].ResAssigned, q.pool[jobs[i].ResAssigned].Client, "Queue.TaskQuit", quitJob, &retJob)
			// Log any errors but we don't care from a flow perspective
			if err != nil {
				log.Error(err.Error())
			}

			if common.IsEmpty(retJob) {
				// The RPC call returned and empty job... something bad has happened
				// so let's quit this job in the Queue and try and quit it on the resource
				log.WithFields(log.Fields{
					"jobuuid":       jobs[i].UUID,
					"jobname":       jobs[i].Name,
					"resource_uuid": jobs[i].ResAssigned,
				}).Error("task quit returned an empty job so it has failed")

				jobs[i].Status = common.STATUS_FAILED

				err = q.db.UpdateJob(jobs[i])
				if err != nil {
					log.Error(err)
				}

			} else {
				log.WithFields(log.Fields{
					"uuid":   retJob.UUID,
					"name":   retJob.Name,
					"resid":  retJob.ResAssigned,
					"params": common.CleanJobParamsForLogging(retJob),
				}).Debug("Saving RPC returned Job to Queue")

				err = q.db.UpdateJob(retJob)
				if err != nil {
					log.WithFields(log.Fields{
						"uuid": retJob.UUID,
						"name": retJob.Name,
					}).Error("Error updating returned job")
				}
			}
		}
	}

	// Get rid of all the resource
	for i := range q.pool {
		log.WithField("resource", q.pool[i].Name).Info("Stopping resource.")
		q.pool[i].Client.Close()
		delete(q.pool, i)
	}

	// We have looped through all jobs so return the last status
	// The Queue should be deleted or nulled from outside this context
	return jobs
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

				jobs, err := q.db.GetAllJobs()
				if err != nil {
					log.Error(err)
				}

				// Quit jobs without a tool in the current resource list
				for j := range jobs {
					var foundTool bool
					// Log checking for tool
					log.WithField("job", jobs[j].UUID).Debug("Checking if tool still exist for job.")

					if jobs[j].Status != common.STATUS_CREATED {
						// Running and Paused tasks will be quit by the act of removing the resources.
						// Done or Quit jobs do not need to be checked as they are not pending.
						// Only created jobs need to be quit if no tool exists
						continue
					}

					for r := range q.pool {
						// Log looking at resource for tools
						log.WithField("resource", q.pool[r].Name).Debug("Checking resource for job tool UUID.")

						// Check for tool on resource from Job
						_, ok := q.pool[r].Tools[jobs[j].ToolUUID]
						if ok {
							// We found a tool for this job so change our bool to note it
							foundTool = true

							log.WithFields(log.Fields{
								"job":      jobs[j].UUID,
								"resource": q.pool[r].Name,
							}).Debug("Job tool found on resources.")
						}
					}

					// We have now been through all resources to if we did not find a tool
					// we should then quit the job
					if !foundTool {
						log.WithFields(log.Fields{
							"job":      jobs[j].UUID,
							"toolUUID": jobs[j].ToolUUID,
						}).Debug("Job tool not found. Job quit")
						jobs[j].Status = common.STATUS_QUIT
						jobs[j].Error = "No tool available in current resource pool."
					}

					err = q.db.UpdateJob(jobs[j])
					if err != nil {
						log.Error(err)
					}
				}

				jobs, err = q.db.GetAllJobs()
				if err != nil {
					log.Error(err)
				}

				// Look for open resources
				// ResourceLoop:
				for resKey := range q.pool {
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
								for jobKey := range jobs {
									logger := log.WithFields(log.Fields{
										"resource": q.pool[resKey].Name,
										"job":      jobs[jobKey].UUID,
									})

									// Are we looking to start or resume the job?
									switch jobs[jobKey].Status {
									case common.STATUS_CREATED: // We are going to start the job fresh
										// We first need to check if this tool exists on this resource
										if tool, ok := q.pool[resKey].Tools[jobs[jobKey].ToolUUID]; ok {
											// We now need to get the hardware requirements for this tool
											if q.pool[resKey].Tools[jobs[jobKey].ToolUUID].Requirements == hardwareKey {
												// We now know we have an open resource and a job that needs that resource
												logger.Debug("Attempting to start new job on resource")

												// It is now possible that the Tool UUID on the resource does not match
												// the Tool UUID the job and Queue have. This is because the Queue changes
												// all UUIDs (keys in the pool map) to make identical Tools across multiple
												// resources show up as one Tool available to the system. We now need to set
												// the Job's ToolUUID field to the correct UUID for the resource if this has
												// happened.
												if tool.UUID != jobs[jobKey].ToolUUID {
													jobs[jobKey].ToolUUID = tool.UUID
												}

												// Add the resource ID
												jobs[jobKey].ResAssigned = resKey

												logger.Debug("Calling Queue.AddTask to start the job.")
												var retJob common.Job
												err := q.resCall(resKey, q.pool[resKey].Client, "Queue.AddTask", common.RPCCall{Job: jobs[jobKey]}, &retJob)
												if err != nil {
													// Something failed so let's mark the job as failed
													logger.WithField("error", err.Error()).Error("Error while attempting to start job on remote resource.")
													retJob.Status = common.STATUS_FAILED
													continue JobLoop
												}

												if common.IsEmpty(retJob) {
													// The RPC call returned and empty job... something bad has happened
													// so let's quit this job in the Queue and try and quit it on the resource
													log.WithFields(log.Fields{
														"jobuuid":       jobs[jobKey].UUID,
														"jobname":       jobs[jobKey].Name,
														"resource_uuid": jobs[jobKey].ResAssigned,
													}).Error("task add returned an empty job so it has failed")

													jobs[jobKey].Status = common.STATUS_FAILED

													err = q.db.UpdateJob(jobs[jobKey])
													if err != nil {
														log.Error(err)
													}
												} else {
													// Job has been started so mark the hardware as in use and assign the resource ID
													if retJob.Status == STATUS_RUNNING {
														q.pool[resKey].Hardware[hardwareKey] = false
													}

													log.WithFields(log.Fields{
														"uuid":   retJob.UUID,
														"name":   retJob.Name,
														"status": retJob.Status,
														"resid":  retJob.ResAssigned,
														"params": common.CleanJobParamsForLogging(retJob),
													}).Debug("Saving RPC returned Job to Queue")

													err = q.db.UpdateJob(retJob)
													if err != nil {
														log.Error(err)
													}

													// Call out to our registered hooks to note job has started
													go HookOnJobStart(Hooks.JobStart, retJob)
												}

												break HardwareLoop
											}
										}
									case common.STATUS_PAUSED: // We are going to resume the job were it is
										// We are resuming a job so we first need to check if the job was assigned to this resource
										if jobs[jobKey].ResAssigned == resKey {
											// This job was assigned to this resource so we need to find the correct local UUID of the tool
											for _, resTool := range q.pool[resKey].Tools {
												if resTool.UUID == jobs[jobKey].ToolUUID {
													// We have found the correct UUID, so check if this is the available hardware
													if resTool.Requirements == hardwareKey {
														// The job requires the hardware that is available on this resource to resume
														logger.Debug("Attempting to resume job.")

														var retJob common.Job
														err := q.resCall(resKey, q.pool[resKey].Client, "Queue.TaskRun", common.RPCCall{Job: jobs[jobKey]}, &retJob)
														if err != nil {
															// Something failed so let's mark the job as failed
															logger.WithField("error", err.Error()).Error("Error while attempting to resume job on remote resource.")
															retJob.Status = common.STATUS_FAILED
															continue JobLoop
														}

														if common.IsEmpty(retJob) {
															// The RPC call returned and empty job... something bad has happened
															// so let's quit this job in the Queue and try and quit it on the resource
															log.WithFields(log.Fields{
																"jobuuid":       jobs[jobKey].UUID,
																"jobname":       jobs[jobKey].Name,
																"status":        retJob.Status,
																"resource_uuid": jobs[jobKey].ResAssigned,
															}).Error("task run returned an empty job so it has failed")

															jobs[jobKey].Status = common.STATUS_FAILED

															err = q.db.UpdateJob(jobs[jobKey])
															if err != nil {
																log.Error(err)
															}
														} else {
															// Job has been started so mark the hardware as in use
															if retJob.Status == STATUS_RUNNING {
																q.pool[resKey].Hardware[hardwareKey] = false
															}

															log.WithFields(log.Fields{
																"uuid":   retJob.UUID,
																"name":   retJob.Name,
																"resid":  retJob.ResAssigned,
																"params": common.CleanJobParamsForLogging(retJob),
															}).Debug("Saving RPC returned Job to Queue")

															err = q.db.UpdateJob(retJob)
															if err != nil {
																log.Error(err)
															}
														}

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
	jobs, err := q.db.GetAllJobs()
	if err != nil {
		log.Error(err)
	}

	// Loop through jobs and get the status of running jobs
	for i := range jobs {
		if jobs[i].Status == common.STATUS_RUNNING {
			// Build status update call
			jobStatus := common.RPCCall{Job: jobs[i]}
			var retJob common.Job

			log.WithFields(log.Fields{
				"uuid":    jobs[i].UUID,
				"name":    jobs[i].Name,
				"status":  jobs[i].Status,
				"resuuid": jobs[i].ResAssigned,
				"client":  q.pool[jobs[i].ResAssigned].Client,
			}).Debug("RPC call for task status")
			err := q.resCall(jobs[i].ResAssigned, q.pool[jobs[i].ResAssigned].Client, "Queue.TaskStatus", jobStatus, &retJob)
			// we care about the errors, but only from a logging perspective
			if err != nil {
				log.WithField("rpc error", err.Error()).Error("Error during RPC call.")
			}

			if common.IsEmpty(retJob) {
				// The RPC call returned and empty job... something bad has happened
				// so let's quit this job in the Queue and try and quit it on the resource
				log.WithFields(log.Fields{
					"jobuuid":       jobs[i].UUID,
					"jobname":       jobs[i].Name,
					"resource_uuid": jobs[i].ResAssigned,
				}).Error("task status returned an empty job so it has failed")

				jobs[i].Status = common.STATUS_FAILED

				err = q.db.UpdateJob(retJob)
				if err != nil {
					log.Error(err)
				}

				err := q.resCall(jobs[i].ResAssigned, q.pool[jobs[i].ResAssigned].Client, "Queue.TaskQuit", jobStatus, &retJob)
				if err != nil {
					// This probably means something really bad happened on the resource, so throw an error in the log
					log.Error(err)
					log.WithField("resource_uuid", jobs[i].ResAssigned).Error("resource might be dead or broken")
				}

				continue
			}

			log.WithFields(log.Fields{
				"uuid":   retJob.UUID,
				"name":   retJob.Name,
				"resid":  retJob.ResAssigned,
				"params": common.CleanJobParamsForLogging(retJob),
			}).Debug("Returned job from Status Call")

			// Check if this is now no longer running
			if retJob.Status != common.STATUS_RUNNING {
				// Release the resources from this change
				log.WithField("JobID", retJob.UUID).Debug("Job has finished.")

				// Call out to the registered hooks that the job is complete
				go HookOnJobFinish(Hooks.JobFinish, retJob)

				var hw string
				for _, v := range q.pool[retJob.ResAssigned].Tools {
					if v.UUID == retJob.ToolUUID {
						hw = v.Requirements
					}
				}
				log.WithFields(log.Fields{
					"resassigned": retJob.ResAssigned,
					"hardware":    hw,
				}).Debug("Resetting hardware availability")
				q.pool[retJob.ResAssigned].Hardware[hw] = true

				// Set a purge time
				retJob.PurgeTime = time.Now().Add(time.Duration(q.jpurge*24) * time.Hour)
				// Log purge time
				log.WithFields(log.Fields{
					"JobID":     retJob.UUID,
					"PurgeTime": retJob.PurgeTime,
				}).Debug("Updated PurgeTime value")

				err := q.resCall(jobs[i].ResAssigned, q.pool[jobs[i].ResAssigned].Client, "Queue.TaskDone", jobStatus, &retJob)
				// we care about the errors, but only from a logging perspective
				if err != nil {
					log.WithField("rpc error", err.Error()).Error("Error during RPC call.")
				}
			}

			err = q.db.UpdateJob(retJob)
			if err != nil {
				log.Error(err)
			}
		}

		// Check and delete jobs past their purge timer
		if jobs[i].Status == common.STATUS_DONE || jobs[i].Status == common.STATUS_FAILED || jobs[i].Status == common.STATUS_QUIT {
			if time.Now().After(jobs[i].PurgeTime) {
				err = q.db.DeleteJob(jobs[i].UUID)
				if err != nil {
					log.Error(err)
				}
			}
		}
	}
}

// Types returns all the different tool types such as GPU, CPU, NET, etc.
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

// ActiveTools allows you to get tools that can actively have jobs created for them
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

// AllTools is used to get all tools that have ever been available
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

// AllResourceManagers is used to gather all currently available resource managers
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

// GetResourceManager will return a copy of a single resource manager from the Queue
// and all of the associated etails.  It takes a parameter of the system name of
// the manager desired.
func (q *Queue) GetResourceManager(systemname string) (ResourceManager, bool) {
	manager, ok := q.managers.Get(systemname)
	if ok == true {
		mgrtype := manager.(ResourceManager)
		return mgrtype, ok
	}

	return nil, ok
}

// AddResourceManager is used to add a resource manager to the map of all available
// managers and is used to add resourcemanager plugins during their creation.
func (q *Queue) AddResourceManager(resmgr ResourceManager) error {
	//Get the ID of the manager we're adding
	id := resmgr.SystemName()

	//Let's check and see if it already exists, if so we should error
	if _, ok := q.managers.Get(id); ok {
		log.WithField("id", id).Error("ResourceManager cannot be added twice.")
		return errors.New("ResourceManager cannot be added twice")
	}

	//Otherwise, set our resource manager and be done with it.
	q.managers.Set(id, resmgr)

	//Log that we did it, because that's just good practice.
	log.WithField("id", id).Info("Added resource manager into the queue.")

	//Return with no error.
	return nil
}

// KeepAllResourceManagers will loop through all resource managers and executes thier keeper
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

// ConnectResource will connect to a resource
func (q *Queue) ConnectResource(resUUID, addr string, tlsconfig *tls.Config) error {
	q.RLock()
	localRes := q.pool[resUUID]
	q.RUnlock()

	// First, setup the address we're going to connect to
	localRes.Address = addr
	// Then store a local version in the event we need to add the default port
	target := localRes.Address

	// Check to see if we have a port, otherwise use the default 9443
	if !strings.Contains(target, ":") {
		target += ":9443"
	}
	log.WithField("addr", target).Info("Connecting to resource")

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

	// Call out to the registered hooks about resource creation
	go HookOnResourceConnect(Hooks.ResourceConnect, resUUID, localRes)

	return nil
}

// reconnectResourceNoLock function will reconnect to a resource that has failed
func (q *Queue) reconnectResourceNoLock(resUUID string, tlsconfig *tls.Config) error {
	localRes := q.pool[resUUID]

	// Get the existing resource address
	target := localRes.Address

	// Check to see if we have a port, otherwise use the default 9443
	if !strings.Contains(target, ":") {
		target += ":9443"
	}
	log.WithField("addr", target).Info("Reconnecting to resource")

	dialer := &net.Dialer{
		Timeout: 15 * time.Second,
	}

	conn, err := tls.DialWithDialer(dialer, "tcp", target, tlsconfig)
	if err != nil {
		log.WithFields(log.Fields{
			"addr":       target,
			"servername": localRes.Address,
		}).Debug("An error occured while building the TLS connection to reconnect to the resource")
		return err
	}

	// Build the RPC client for the resource
	localRes.Client = rpc.NewClient(conn)
	if err != nil {
		log.WithField("addr", target).Debug("An error occured while creating new client for the reconnection to the resource")
		return err
	}

	// Let the user know we connected
	log.WithField("target", localRes.Address).Info("Successfully reconnected to resource")
	localRes.Status = common.STATUS_RUNNING

	q.pool[resUUID] = localRes

	// Call out to the registered hooks about resource creation
	go HookOnResourceConnect(Hooks.ResourceConnect, resUUID, localRes)

	return nil
}

// ReconnectResource function will reconnect to a resource that has failed
func (q *Queue) ReconnectResource(resUUID string, tlsconfig *tls.Config) error {
	q.Lock()
	defer q.Unlock()

	return q.reconnectResourceNoLock(resUUID, tlsconfig)
}

// CheckResourceConnectionStatus checks to see if our RPC connection to a resource is still valid, if not it
// will return false, otherwise it will return true.
func (q *Queue) CheckResourceConnectionStatus(res *Resource) bool {
	if res.Client == nil {
		return false
	}

	var reply int
	ping := int(12345)
	err := res.Client.Call("Queue.Ping", ping, &reply)
	//if err == rpc.ErrShutdown || err == io.EOF || err == io.ErrUnexpectedEOF {
	if err != nil {
		log.WithField("error", err.Error()).Debug("Error pinging RPC server")
		return false
	}

	if reply != (ping * ping) {
		log.WithFields(log.Fields{
			"ping":  ping,
			"reply": reply,
		}).Error("ping did not return the correct value")
		return false
	}

	return true
}

// LoadRemoteResourceHardware loads all of the hardware for a remote resource
func (q *Queue) LoadRemoteResourceHardware(resUUID string) {
	q.RLock()
	localRes := q.pool[resUUID]
	q.RUnlock()

	// Get Hardware
	err := q.resCall(resUUID, localRes.Client, "Queue.ResourceHardware", common.RPCCall{}, &localRes.Hardware)
	if err != nil {
		log.WithFields(log.Fields{
			"error":    err.Error(),
			"resource": resUUID,
		}).Error("Unable to gather resource hardware.")
		return
	}

	// Set all hardware as available
	for key := range localRes.Hardware {
		localRes.Hardware[key] = true
	}

	q.Lock()
	q.pool[resUUID] = localRes
	q.Unlock()

	log.WithField("resources", resUUID).Debug("Loaded hardware for resource")
}

// LoadRemoteResourceTools returns the tool information from a resource
func (q *Queue) LoadRemoteResourceTools(resUUID string) {
	q.RLock()
	localRes := q.pool[resUUID]
	q.RUnlock()

	// Get Tools
	var tools []common.Tool
	err := q.resCall(resUUID, localRes.Client, "Queue.ResourceTools", common.RPCCall{}, &tools)
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
		for i := range q.pool {
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

// AddResource will add a resource to the queue.  Returns the UUID.
func (q *Queue) AddResource(name string) (string, error) {
	// Check that the address is already in use
	for _, v := range q.pool {
		if v.Name == name && v.Status != common.STATUS_QUIT {
			// We have found a resource with the same address so error
			log.WithField("name", name).Debug("Resource already exists.")
			return "", errors.New("Resource already exists")
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

// GetResource returns a resource given a resouce UUID
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
		return errors.New("Given Resource UUID does not exist")
	}

	// Lock the queue
	q.Lock()
	defer q.Unlock()

	jobs, err := q.db.GetAllJobs()
	if err != nil {
		return err
	}

	// Loop through any jobs assigned to the resource and quit them if they are not completed
	for i := range jobs {
		if jobs[i].ResAssigned == resUUID {
			// Check status
			if jobs[i].Status == common.STATUS_RUNNING || jobs[i].Status == common.STATUS_PAUSED {
				// Quit the task
				quitTask := common.RPCCall{Job: jobs[i]}
				var retJob common.Job

				err := q.resCall(resUUID, q.pool[resUUID].Client, "Queue.TaskQuit", quitTask, &retJob)
				if err != nil {
					log.Println(err.Error())
				}

				if common.IsEmpty(retJob) {
					// The RPC call returned and empty job... something bad has happened
					// so let's quit this job in the Queue and try and quit it on the resource
					log.WithFields(log.Fields{
						"jobuuid":       jobs[i].UUID,
						"jobname":       jobs[i].Name,
						"resource_uuid": jobs[i].ResAssigned,
					}).Error("task quit returned an empty job so it has failed")

					jobs[i].Status = common.STATUS_FAILED

					err = q.db.UpdateJob(jobs[i])
					if err != nil {
						log.Error(err)
					}
				} else {
					log.WithFields(log.Fields{
						"uuid":   retJob.UUID,
						"name":   retJob.Name,
						"resid":  retJob.ResAssigned,
						"params": common.CleanJobParamsForLogging(retJob),
					}).Debug("Saving RPC returned Job to Queue")

					err = q.db.UpdateJob(retJob)
					if err != nil {
						log.Error(err)
					}
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
	for key := range res.Tools {
		delete(res.Tools, key)
	}
	q.pool[resUUID] = res
	for i := range q.pool[resUUID].Hardware {
		q.pool[resUUID].Hardware[i] = false
	}

	return nil
}

// resCall attempts to make a RPC call, but checks for a nil client first
// This really feels like a work around for something broken elsewhere...
func (q *Queue) resCall(resID string, client *rpc.Client, serviceMethod string, args interface{}, reply interface{}) error {

	return client.Call(serviceMethod, args, reply)
}
