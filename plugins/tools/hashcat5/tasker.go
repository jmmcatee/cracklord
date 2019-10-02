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
	speedMag     float64        // magnitude of the speed to display (will divide by this number 1,000 [kH/s], 1,000,000 [MH/s], or 1,000,000,000 [GH/s] )
}

// Status returns the common.Job option of the Tasker
func (t *Tasker) Status() common.Job {
	log.WithField("task", t.job.UUID).Debug("Status call for hashcat5 Tasker")

	t.mux.Lock()
	defer t.mux.Unlock()

	if t.job.Status == common.STATUS_RUNNING {
		if !t.stderrCp {
			go func() {
				log.WithFields(common.LogJob(t.job)).Debug("stopping Stderr Pipe Copy")

				t.stderrCp = true
				for t.job.Status == common.STATUS_RUNNING {
					io.Copy(t.stderr, t.stderrPipe)
					// We do not care about this error as we will just keep trying until the job status changes.
				}
				t.stderrCp = false

				log.WithFields(common.LogJob(t.job)).Debug("stopping Stderr Pipe Copy")
			}()
		}

		if !t.stdoutCp {
			go func() {
				log.WithFields(common.LogJob(t.job)).Debug("stopping Stdout Pipe Copy")

				t.stdoutCp = true
				for t.job.Status == common.STATUS_RUNNING {
					io.Copy(t.stdout, t.stdoutPipe)
					// We do not care about this error as we will just keep trying until the job status changes.
				}
				t.stdoutCp = false

				log.WithFields(common.LogJob(t.job)).Debug("stopping Stdout Pipe Copy")
			}()
		}

		if t.stdout.Len() != 0 {
			status, err := ParseMachineOutput(t.stdout.String())

			if err == nil {
				var totalSpeed float64
				for i := range status.Speed {
					totalSpeed += status.Speed[i]
				}

				if t.job.PerformanceTitle == "" {
					if totalSpeed < 1000 {
						t.job.PerformanceTitle = "H/s"
						t.speedMag = 1
					} else if totalSpeed < 1000000 {
						t.job.PerformanceTitle = "kH/s"
						t.speedMag = 1000
					} else if totalSpeed < 1000000000 {
						t.job.PerformanceTitle = "MH/s"
						t.speedMag = 1000000
					} else {
						t.job.PerformanceTitle = "GH/s"
						t.speedMag = 1000000000
					}
				}
				t.job.Progress = status.Progress
				t.job.ETC = status.EstimateTime

				t.job.PerformanceData[fmt.Sprintf("%d", time.Now().Unix())] = fmt.Sprintf("%f", totalSpeed/t.speedMag)

				if status.RecoveredHashes > t.job.CrackedHashes {
					t.job.CrackedHashes = status.RecoveredHashes
				}
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
	hashFile, err := os.Open(filepath.Join(t.wd, ConstHashcatOutputFilename))
	if err == nil {
		_, hashes = ParseHashcatOutputFile(hashFile, t.inputSplits, t.hashMode)
	}

	// Add in the pot file items
	for i := range t.showPotOutput {
		hashes = append(hashes, t.showPotOutput[i])
	}

	if len(hashes) != 0 {
		t.job.OutputData = dedupHashes(hashes)
	}

	// Check if our hashes are in line with our CrackedHashes numbers as sometimes the status will miss this
	if int64(len(t.job.OutputData)) > t.job.CrackedHashes {
		t.job.CrackedHashes = int64(len(t.job.OutputData))
	}

	t.stderr.Reset()
	t.stdout.Reset()

	log.WithFields(common.LogJob(t.job)).Debug("Returning job with status call")
	return t.job
}

// Run starts or resumes the job
func (t *Tasker) Run() error {
	// Get the tasker luck so we can do some work on the job
	t.mux.Lock()
	defer t.mux.Unlock()

	// Check that we have not already finished this job
	if t.job.Status == common.STATUS_DONE || t.job.Status == common.STATUS_QUIT || t.job.Status == common.STATUS_FAILED {
		log.WithFields(common.LogJob(t.job)).Debug("unable to start hashcat5 job as it is done, quit or failed already")
		return errors.New("job already finished")
	}

	// Check if this job is running
	if t.job.Status == common.STATUS_RUNNING {
		// Job already running so return no errors
		log.Debug("hashcat5 job already running")
		return nil
	}

	// We need to first parse the stuff we were given by the user for the hash file.
	// We will do this via hashcat's --left output, which also will create our hash file for cracking
	hashcatLeftExec := exec.Command(config.BinPath, t.showPotLeft...)
	hashcatLeftExec.Dir = t.wd
	log.WithField("arguements", hashcatLeftExec.Args).Debug("executing left flag command")
	showPotLeftStdout, err := hashcatLeftExec.Output()
	if err != nil {
		log.WithFields(log.Fields{
			"returncode": err,
			"stdout":     showPotLeftStdout,
		}).Error("error running hashcat --left command.")
	}

	// Get the first line of the Left output to count our separators (:)
	hashcatLeftFilename := filepath.Join(t.wd, ConstHashcatLeftFilename)
	hashcatLeftFile, err := os.Open(hashcatLeftFilename)
	if err != nil {
		log.Error(err)
		return errors.New("error opening hashcat left flag file")
	}

	// Get the count of hashes and the split count
	var leftCount int64
	leftCount, t.inputSplits = ParseLeftHashFile(hashcatLeftFile)

	// Create and pull the pot file search
	hashcatShowPotExec := exec.Command(config.BinPath, t.showPot...)
	hashcatShowPotExec.Dir = t.wd
	log.WithField("args", hashcatShowPotExec.Args).Debug("show command executing")
	showPotStdout, err := hashcatShowPotExec.Output()
	if err != nil {
		log.WithField("error", err).Error("error running hashcat show command.")
	}
	log.WithField("showStdout", string(showPotStdout)).Debug("Show command stdout.")

	// Get the output of the show pot file
	hashcatPotShowFilename := filepath.Join(t.wd, ConstHashcatPotShowFilename)
	hashcatPotShowFile, err := os.Open(hashcatPotShowFilename)
	if err != nil {
		log.Error(err)
		return errors.New("error opening hashcat show file")
	}
	var potCount int64
	potCount, t.showPotOutput = ParseShowPotFile(hashcatPotShowFile, t.inputSplits, t.hashMode)

	// Set some totals
	t.job.TotalHashes = leftCount + potCount
	t.job.CrackedHashes = potCount

	// Set commands for restore or start
	// We need to check for a restore file. If it does not exist we have to start over and not give the --restore command
	hashcatBinFolder := filepath.Dir(config.BinPath)
	_, err = os.Stat(filepath.Join(hashcatBinFolder, t.job.UUID+".restore"))
	if t.job.Status == common.STATUS_CREATED || os.IsNotExist(err) {
		t.exec = *exec.Command(config.BinPath, t.start...)
	} else {
		t.exec = *exec.Command(config.BinPath, t.resume...)
	}

	// Set the working directory
	t.exec.Dir = t.wd
	log.WithFields(log.Fields{
		"workingdir": t.exec.Dir,
	}).Debug("setup working directory")

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
	log.WithField("arguments", t.exec.Args).Debug("running command")
	err = t.exec.Start()
	t.doneWG.Add(1)
	if err != nil {
		// We had an error starting to return that and quit the job
		t.job.Status = common.STATUS_FAILED
		log.WithFields(common.LogJob(t.job)).Errorf("there was an error starting the job: %v", err)
		return err
	}

	t.job.StartTime = time.Now()
	t.job.Status = common.STATUS_RUNNING
	t.job.ETC = "Warming up..."

	go func() {
		// Wait for the job to finish
		t.exec.Wait()
		t.mux.Lock()
		log.WithFields(common.LogJob(t.job)).Debug("job execution returned wait function")

		switch t.returnStatus {
		case "":
			t.job.Status = common.STATUS_DONE
		case common.STATUS_PAUSED:
			t.job.Status = common.STATUS_PAUSED
		case common.STATUS_QUIT:
			t.job.Status = common.STATUS_QUIT
		}

		t.returnStatus = ""

		t.doneWG.Done()
		t.mux.Unlock()
	}()

	return nil
}

func (t *Tasker) quitExec(returnStatus string) {
	t.mux.Lock()

	// First set the reason we are quiting the process (Pause or Quit)
	t.returnStatus = returnStatus

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

	t.mux.Unlock()
}

// Pause kills the hashcat process and marks the job as paused
func (t *Tasker) Pause() error {
	log.WithFields(common.LogJob(t.job)).Debug("attempting to pause hashcat task")

	if t.job.Status == common.STATUS_RUNNING {
		// Call status to update the job internals before pausing
		t.Status()

		t.quitExec(common.STATUS_PAUSED)

		log.WithFields(common.LogJob(t.job)).Debug("task paused successfully")
	}

	return nil
}

// Quit kills the hashcat process and then returns the most up-to-date status
func (t *Tasker) Quit() common.Job {
	log.WithFields(common.LogJob(t.job)).Debug("attempting to quit hashcat task")

	if t.job.Status == common.STATUS_RUNNING {
		// Call status to update the job internals before quiting
		t.Status()

		t.quitExec(common.STATUS_QUIT)

		log.WithFields(common.LogJob(t.job)).Debug("task quit successfully")
	}
	return t.job
}

// IOE is no longer used and is a empty interface for
func (t *Tasker) IOE() (io.Writer, io.Reader, io.Reader) {
	return nil, nil, nil
}
