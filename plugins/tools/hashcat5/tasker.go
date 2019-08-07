package hashcat5

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/jmmcatee/cracklord/common"
)

// Tasker is the structure that implements the Tasker inteface
type Tasker struct {
	mux           sync.Mutex // Used for locking componets of the Tasker
	job           common.Job
	wd            string
	exec          exec.Cmd
	start         []string
	resume        []string
	showPot       []string
	showPotLeft   []string
	showPotOutput [][]string
	hashes        [][]byte
	inputSplits   int
	hashMode      string

	stderr     *bytes.Buffer
	stderrCp   bool
	stdout     *bytes.Buffer
	stdoutCp   bool
	stderrPipe io.ReadCloser
	stdoutPipe io.ReadCloser
	stdinPipe  io.WriteCloser

	doneWG       sync.WaitGroup // Used for checking if the job is done
	returnStatus string         // Used to note if we quit from an error, completed, or paused
}

// Status returns the common.Job option of the Tasker
func (t *Tasker) Status() common.Job {
	log.WithField("task", t.job.UUID).Debug("Status call for hashcat3 Tasker")

	t.mux.Lock()
	defer t.mux.Unlock()

	if t.job.Status == common.STATUS_RUNNING {
		if !t.stderrCp {
			go func() {
				log.WithFields(log.Fields{
					"jobUUID": t.job.UUID,
				}).Debug("Stopping Stderr Pipe Copy")

				t.stderrCp = true
				for t.job.Status == common.STATUS_RUNNING {
					io.Copy(t.stderr, t.stderrPipe)
					// We do not care about this error as we will just keep trying until the job status changes.
				}
				t.stderrCp = false

				log.WithFields(log.Fields{
					"jobUUID": t.job.UUID,
				}).Debug("Stopping Stderr Pipe Copy")
			}()
		}

		if !t.stdoutCp {
			go func() {
				log.WithFields(log.Fields{
					"jobUUID": t.job.UUID,
				}).Debug("Stopping Stdout Pipe Copy")

				t.stdoutCp = true
				for t.job.Status == common.STATUS_RUNNING {
					io.Copy(t.stdout, t.stdoutPipe)
					// We do not care about this error as we will just keep trying until the job status changes.
				}
				t.stdoutCp = false

				log.WithFields(log.Fields{
					"jobUUID": t.job.UUID,
				}).Debug("Stopping Stdout Pipe Copy")
			}()
		}

		if t.stdout.Len() != 0 {
			status, err := ParseMachineOutput(t.stdout.String())

			if err == nil {
				if t.job.PerformanceTitle == "" {
					t.job.PerformanceTitle = "MH/s"
				}
				t.job.Progress = status.Progress
				t.job.ETC = status.EstimateTime

				var totalSpeed float64
				for i := range status.Speed {
					totalSpeed += status.Speed[i]
				}
				t.job.PerformanceData[fmt.Sprintf("%d", time.Now().Unix())] = fmt.Sprintf("%f", totalSpeed/1000000)

				t.job.CrackedHashes = status.RecoveredHashes
				t.job.TotalHashes = status.TotalHashes
			} else {
				log.Debug(err.Error())
			}
		}

		if t.stderr.Len() != 0 {
			t.job.Error = t.stderr.String()
		}
	}

	// Get the hash file
	var hashes [][]string
	hashFile, err := os.Open(filepath.Join(t.wd, HASH_OUTPUT_FILENAME))
	if err == nil {
		_, hashes = ParseHashcatOutputFile(hashFile, t.inputSplits, t.hashMode)
	} else {
		log.WithField("io_error", err).Debug("Failed to open output.txt")
	}

	// Add in the pot file items
	for i := range t.showPotOutput {
		hashes = append(hashes, t.showPotOutput[i])
	}

	if len(hashes) != 0 {
		t.job.OutputData = dedupHashes(hashes)
	}

	t.stderr.Reset()
	t.stdout.Reset()

	return t.job
}

// Run starts or resumes the job
func (t *Tasker) Run() error {
	// Get the tasker luck so we can do some work on the job
	t.mux.Lock()
	defer t.mux.Unlock()

	// Check that we have not already finished this job
	if t.job.Status == common.STATUS_DONE || t.job.Status == common.STATUS_QUIT || t.job.Status == common.STATUS_FAILED {
		log.WithField("Status", t.job.Status).Debug("Unable to start hashcat5 job as it is done.")
		return errors.New("Job already finished.")
	}

	// Check if this job is running
	if t.job.Status == common.STATUS_RUNNING {
		// Job already running so return no errors
		log.Debug("hashcat5 job already running, doing nothing")
		return nil
	}

	// We need to first parse the stuff we were given by the user for the hash file.
	// We will do this via hashcat's --left output, which also will create our hash file for cracking
	hashcatLeftExec := exec.Command(config.BinPath, t.showPotLeft...)
	hashcatLeftExec.Dir = t.wd
	log.WithField("Left Command", hashcatLeftExec.Args).Debug("Executing Left Command")
	showPotLeftStdout, err := hashcatLeftExec.Output()
	if err != nil {
		log.WithField("execError", err).Error("Error running hashcat --left command.")
	}
	log.WithField("showLeftStdout", string(showPotLeftStdout)).Debug("Show Left command stdout.")

	// Get the first line of the Left output to count our separators (:)
	hashcatLeftFilename := filepath.Join(t.wd, HASHCAT_LEFT_FILENAME)
	hashcatLeftFile, err := os.Open(hashcatLeftFilename)
	if err != nil {
		log.Error(err)
		return errors.New("Error opening LEFT Hash file")
	}

	// Get the count of hashes and the split count
	var leftCount int64
	leftCount, t.inputSplits = ParseLeftHashFile(hashcatLeftFile)

	// Create and pull the pot file search
	hashcatShowPotExec := exec.Command(config.BinPath, t.showPot...)
	hashcatShowPotExec.Dir = t.wd
	log.WithField("Show Command", hashcatShowPotExec.Args).Debug("Executing Show Command")
	showPotStdout, err := hashcatShowPotExec.Output()
	if err != nil {
		log.WithField("execError", err).Error("Error running hashcat --show command.")
	}
	log.WithField("showStdout", string(showPotStdout)).Debug("Show command stdout.")

	// Get the output of the show pot file
	hashcatPotShowFilename := filepath.Join(t.wd, HASHCAT_POT_SHOW_FILENAME)
	hashcatPotShowFile, err := os.Open(hashcatPotShowFilename)
	if err != nil {
		log.Error(err)
		return errors.New("Error opening LEFT Hash file")
	}
	var potCount int64
	potCount, t.showPotOutput = ParseShowPotFile(hashcatPotShowFile, t.inputSplits, t.hashMode)

	// Set some totals
	t.job.TotalHashes = leftCount + potCount
	t.job.CrackedHashes = potCount

	// Set commands for restore or start
	if t.job.Status == common.STATUS_CREATED {
		t.exec = *exec.Command(config.BinPath, t.start...)
	} else {
		t.exec = *exec.Command(config.BinPath, t.resume...)
	}

	// Set the working directory
	t.exec.Dir = t.wd
	log.WithFields(log.Fields{
		"dir": t.exec.Dir,
	}).Debug("Setup working directory")

	// Assign the stderr, stdout, stdin pipes
	t.stderrPipe, err = t.exec.StderrPipe()
	if err != nil {
		return err
	}

	t.stdoutPipe, err = t.exec.StdoutPipe()
	if err != nil {
		return err
	}

	t.stdinPipe, err = t.exec.StdinPipe()
	if err != nil {
		return err
	}

	t.stderr = bytes.NewBuffer([]byte(""))
	t.stdout = bytes.NewBuffer([]byte(""))

	// Start the command
	log.WithField("argument", t.exec.Args).Debug("Running command.")
	err = t.exec.Start()
	t.doneWG.Add(1)
	if err != nil {
		// We had an error starting to return that and quit the job
		t.job.Status = common.STATUS_FAILED
		log.Errorf("There was an error starting the job: %v", err)
		return err
	}

	t.job.StartTime = time.Now()
	t.job.Status = common.STATUS_RUNNING
	t.job.ETC = "Warming up..."

	go func() {
		// Wait for the job to finish
		t.exec.Wait()
		t.mux.Lock()
		log.WithField("task", t.job.UUID).Debug("Job exec returned Wait().")

		//log.WithField("task", t.job.UUID).Debug("Took lock on job to change status to done.")
		switch t.returnStatus {
		case "":
			t.job.Status = common.STATUS_DONE
		case common.STATUS_PAUSED:
			t.job.Status = common.STATUS_PAUSED
		case common.STATUS_QUIT:
			t.job.Status = common.STATUS_QUIT
		}

		//log.WithField("task", t.job.UUID).Debug("Unlocked job after setting done.")

		// Get the status now because we need the last output of hashes
		//log.WithField("task", t.job.UUID).Debug("Calling final status call for the job.")
		t.Status()

		//log.WithField("task", t.job.UUID).Debug("Releasing wait group.")
		t.doneWG.Done()
		t.mux.Unlock()
	}()

	return nil
}

func (t *Tasker) quitExec(returnStatus string) {
	t.mux.Lock()

	if runtime.GOOS == "windows" {
		t.exec.Process.Kill()
	} else {
		io.WriteString(t.stdinPipe, "q")
	}

	// We tried the soft quit so now let's wait and see if that works and if not, kill it hard
	c := make(chan struct{})
	go func() {
		defer close(c)
		t.doneWG.Wait()
	}()

	// Check for a 30 second timeout
	select {
	case <-c:
	// Task quit so we are good to go
	case <-time.After(30 * time.Second):
		t.exec.Process.Kill()
	}

	t.returnStatus = returnStatus

	t.mux.Unlock()
}

// Pause kills the hashcat process and marks the job as paused
func (t *Tasker) Pause() error {
	log.WithField("task", t.job.UUID).Debug("Attempting to pause hashcat task")

	// Call status to update the job internals before pausing
	t.Status()

	if t.job.Status == common.STATUS_RUNNING {
		t.quitExec(common.STATUS_PAUSED)

		log.WithField("task", t.job.UUID).Debug("Task paused successfully")
	}

	return nil
}

// Quit kills the hashcat process and then returns the most up-to-date status
func (t *Tasker) Quit() common.Job {
	log.WithField("task", t.job.UUID).Debug("Attempting to quit hashcat task")

	// Call status to update the job internals before quiting
	t.Status()

	if t.job.Status == common.STATUS_RUNNING {
		t.quitExec(common.STATUS_QUIT)

		log.WithField("task", t.job.UUID).Debug("Task quit successfully")
	}
	return t.job
}

// IOE is no longer used and is a empty interface for
func (t *Tasker) IOE() (io.Writer, io.Reader, io.Reader) {
	return nil, nil, nil
}
