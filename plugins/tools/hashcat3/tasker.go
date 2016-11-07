package hashcat3

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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
	stderr        *bytes.Buffer
	stderrCp      bool
	stdout        *bytes.Buffer
	stdoutCp      bool
	stderrPipe    io.ReadCloser
	stdoutPipe    io.ReadCloser
	stdinPipe     io.WriteCloser

	doneWG sync.WaitGroup // Used for checking if the job is done
}

// Status returns the common.Job option of the Tasker
func (t *Tasker) Status() common.Job {
	log.WithField("task", t.job.UUID).Debug("Status call for hashcat3 Tasker")

	t.mux.Lock()
	defer t.mux.Unlock()

	if t.job.Status == common.STATUS_RUNNING {
		if !t.stderrCp {
			go func() {
				t.stderrCp = true
				for t.job.Status == common.STATUS_RUNNING {
					cpNE, err := io.Copy(t.stderr, t.stderrPipe)
					if err != nil {
						log.WithField("error", err.Error()).Warn("Error copying from CMD Stderr Pipe.")
					}

					log.WithFields(log.Fields{
						"stderrCount": cpNE,
					}).Debug("Number of bytes copied from Stderr of CMD.")
				}
				t.stderrCp = false
			}()
		}

		if !t.stdoutCp {
			go func() {
				t.stdoutCp = true
				for t.job.Status == common.STATUS_RUNNING {
					cpNO, err := io.Copy(t.stdout, t.stdoutPipe)
					if err != nil {
						log.WithField("error", err.Error()).Warn("Error copying from CMD Stdout Pipe.")
					}

					log.WithFields(log.Fields{
						"stdoutCount": cpNO,
					}).Debug("Number of bytes copied from Stdout of CMD.")
				}
				t.stdoutCp = false
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
				log.Debug(err.Error)
			}
		}

		if t.stderr.Len() != 0 {
			t.job.Error = t.stderr.String()
		}
	}

	// Get the hash file
	var tempOutputData [][]string
	hashFile, err := os.Open(filepath.Join(t.wd, "output.txt"))
	if err == nil {
		defer hashFile.Close()

		// Pull the lines from the file for each individual hash
		hashScanner := bufio.NewScanner(hashFile)
		for hashScanner.Scan() {
			// Parse the line with the default separator |
			parts := strings.Split(hashScanner.Text(), ":")
			splitCount := strings.Count(hashScanner.Text(), ":")

			// Add the parts to the output array
			// 1 => 1:o             l=2, split=0, split+2=2
			// 1:2 => 1:2:o         l=3, split=1, split+2=3
			// 1:2:3 => 1:2:3:o     l=4, split=2, split+2=4

			// 1:2 => 1:2:o:: parts=4, split=1, split+2=3
			var lineHash string
			var password string
			if splitCount > t.inputSplits {
				for i := 0; i < t.inputSplits+1; i++ {
					if i < len(parts)-1 {
						lineHash += parts[i]
					}

					if i < t.inputSplits {
						lineHash += ":"
					}
				}

				password = parts[t.inputSplits+1]
				if t.inputSplits+1 < splitCount {
					for i := 0; i < splitCount-(t.inputSplits+1); i++ {
						password += ":"
					}
				}
			} else {
				// We need to rebuild the hash from the stored verion we recieved (PWDUMP exception)
				for x := range t.hashes {
					if bytes.Contains(bytes.ToLower(t.hashes[x]), []byte(parts[0])) {
						lineHash = string(t.hashes[x])
					}
				}

				password = hashScanner.Text()[len(parts[0])+1:]
			}

			// We now need to add back any accidentally removed :
			tempOutputData = append(tempOutputData, []string{password, lineHash})
		}
	}
	if err != nil {
		log.WithField("io_error", err).Error("Failed to open output.txt")
	}

	// Add in the pot file items
	for i := range t.showPotOutput {
		tempOutputData = append(tempOutputData, t.showPotOutput[i])
	}

	if len(tempOutputData) != 0 {
		t.job.OutputData = tempOutputData
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
		log.WithField("Status", t.job.Status).Debug("Unable to start hashcat3 job as it is done.")
		return errors.New("Job already finished.")
	}

	// Check if this job is running
	if t.job.Status == common.STATUS_RUNNING {
		// Job already running so return no errors
		log.Debug("hashcat3 job already running, doing nothing")
		return nil
	}

	// Execute the Hashcat --show command to get any potfile entries
	showExec := exec.Command(config.BinPath, t.showPot...)
	showExec.Dir = t.wd
	log.WithField("Show Command", showExec.Args).Debug("Executing Show Command")
	showPotStdout, err := showExec.Output()
	if err != nil {
		log.WithField("execError", err).Error("Error running hashcat --show command.")
	}
	log.WithField("showStdout", string(showPotStdout)).Debug("Show command stdout.")
	t.showPotOutput = ParseShowPotOutput(string(showPotStdout), t.inputSplits)

	showLeftExec := exec.Command(config.BinPath, t.showPotLeft...)
	showLeftExec.Dir = t.wd
	log.WithField("Left Command", showLeftExec.Args).Debug("Executing Left Command")
	showPotLeftStdout, err := showLeftExec.Output()
	if err != nil {
		log.WithField("execError", err).Error("Error running hashcat --left command.")
	}
	log.WithField("showLeftStdout", string(showPotLeftStdout)).Debug("Show Left command stdout.")
	leftOut := ParseShowPotLeftOutput(string(showPotLeftStdout))

	t.job.TotalHashes = int64(len(leftOut) + len(t.showPotOutput))
	t.job.CrackedHashes = int64(len(t.showPotOutput))

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

	go func() {
		// Wait for the job to finish
		t.exec.Wait()
		log.WithField("task", t.job.UUID).Debug("Job exec returned Wait().")

		t.mux.Lock()
		log.WithField("task", t.job.UUID).Debug("Took lock on job to change status to done.")
		t.job.Status = common.STATUS_DONE
		t.mux.Unlock()
		log.WithField("task", t.job.UUID).Debug("Unlocked job after setting done.")

		// Get the status now because we need the last output of hashes
		log.WithField("task", t.job.UUID).Debug("Calling final status call for the job.")
		t.Status()

		log.WithField("task", t.job.UUID).Debug("Releasing wait group.")
		t.doneWG.Done()
	}()

	return nil
}

// Pause kills the hashcat process and marks the job as paused
func (t *Tasker) Pause() error {
	log.WithField("task", t.job.UUID).Debug("Attempting to pause hashcat task")

	// Call status to update the job internals before pausing
	t.Status()

	if t.job.Status == common.STATUS_RUNNING {
		t.mux.Lock()

		if runtime.GOOS == "windows" {
			t.exec.Process.Kill()
		} else {
			io.WriteString(t.stdinPipe, "c")

			time.Sleep(1 * time.Second)

			io.WriteString(t.stdinPipe, "q")
		}

		t.mux.Unlock()

		// Wait for the program to actually exit
		t.doneWG.Wait()
	}

	// Change status to pause
	t.mux.Lock()
	t.job.Status = common.STATUS_PAUSED
	t.mux.Unlock()

	log.WithField("task", t.job.UUID).Debug("Task paused successfully")

	return nil
}

// Quit kills the hashcat process and then returns the most up-to-date status
func (t *Tasker) Quit() common.Job {
	log.WithField("task", t.job.UUID).Debug("Attempting to quit hashcat task")

	// Call status to update the job internals before quiting
	t.Status()

	if t.job.Status == common.STATUS_RUNNING || t.job.Status == common.STATUS_PAUSED {
		t.mux.Lock()
		log.WithField("task", t.job.UUID).Debug("Grab lock to push quit signal to hashcat.")

		if runtime.GOOS == "windows" {
			log.WithField("task", t.job.UUID).Debug("Attempting to send Windows Process.Kill() Signal.")
			t.exec.Process.Kill()
		} else {
			log.WithField("task", t.job.UUID).Debug("Attempting UNIX kill with 'c' and 'q'.")
			io.WriteString(t.stdinPipe, "c")

			log.WithField("task", t.job.UUID).Debug("Sent 'c' command and waiting 1 second.")
			time.Sleep(1 * time.Second)

			log.WithField("task", t.job.UUID).Debug("Sending 'q' command to kill the process.")
			io.WriteString(t.stdinPipe, "q")
		}

		t.mux.Unlock()
		log.WithField("task", t.job.UUID).Debug("Unlocking job and waiting for quit WaitGroup.")

		// Wait for the program to actually exit
		t.doneWG.Wait()
		log.WithField("task", t.job.UUID).Debug("Wait group returned. Task quit successfully.")
	}

	t.mux.Lock()
	t.job.Status = common.STATUS_QUIT
	t.mux.Unlock()

	log.WithField("task", t.job.UUID).Debug("Task quit successfully")
	return t.job
}

// IOE is no longer used and is a empty interface for
func (t *Tasker) IOE() (io.Writer, io.Reader, io.Reader) {
	return nil, nil, nil
}
