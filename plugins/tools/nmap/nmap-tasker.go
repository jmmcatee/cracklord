package nmap

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/jmmcatee/cracklord/common"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

var regHostsCompleted *regexp.Regexp
var regTimeEstimate *regexp.Regexp

func init() {
	var err error
	regHostsCompleted, err = regexp.Compile(`(\d+) hosts completed`)
	regTimeEstimate, err = regexp.Compile(`About (\d{1,3}\.\d\d)% done;.*\((.*)\)`)

	if err != nil {
		panic(err.Error())
	}
}

type nmapTasker struct {
	job        common.Job
	wd         string
	cmd        exec.Cmd
	start      []string
	resume     []string
	stderr     *bytes.Buffer
	stdout     *bytes.Buffer
	stderrPipe io.ReadCloser
	stdoutPipe io.ReadCloser
	stdinPipe  io.WriteCloser

	waitChan chan struct{}

	mux sync.Mutex
}

func newHashcatTask(j common.Job) (common.Tasker, error) {
	t := nmapTasker{}
	t.waitChan = make(chan struct{}, 1)

	t.job = j

	// Build a working directory for this job
	t.wd = filepath.Join(config.WorkDir, t.job.UUID)
	err := os.Mkdir(h.wd, 0700)
	if err != nil {
		// Couldn't make a directory so kill the job
		log.WithFields(log.Fields{
			"path":  t.wd,
			"error": err.Error(),
		}).Error("Nmap could not create a working directory")
		return &nmapTasker{}, errors.New("Could not create a working directory.")
	}
	log.WithField("path", t.wd).Debug("Nmap working directory created")

	// Build the arguements for hashcat
	args := []string{}

	//Setup our configuration items we always want
	args = append(args, "--stats-every 30s")

	// Get the scan type from the job
	scantypekey, ok := t.job.Parameters["scantype"]
	if !ok {
		log.WithFields(log.Fields{
			"algoritm": t.job.Parameters["scantype"],
			"err":      ok,
		}).Error("Could not find the scan type provided")
		return &nmapTasker{}, errors.New("Could not find the scan type provided.")
	}
	args = append(args, scanTypes[scantypekey])

	// Get the timing settings we should be using
	timingkey, ok := t.job.Parameters["timing"]
	if !ok {
		log.WithFields(log.Fields{
			"timing": t.job.Parameters["timing"]
			"err":      ok,
		}).Error("Could not find the scan type provided")
		return &nmapTasker{}, errors.New("Could not find the scan type provided.")
	}
	args = append(args, scanTypes[timingkey])

	serviceversion, ok := t.job.Parameters["serviceversion"]
	if ok {
		if serviceversion == "true" {
			args = append(args, "-sV")
		}
	}

	hostdiscovery, ok := t.job.Parameters["hostdiscovery"]
	if ok {
		if hostdiscovery == "true" {
			args = append(args, "-Pn")
		}
	}

	// Add our output files
	args = append(args, "-oG", filepath.Join(t.wd, "greppable-output.txt"))
	args = append(args, "-oX", filepath.Join(t.wd, "xml-output.xml"))

	//Append config file arguments
	if config.Arguments != "" {
		args = append(args, config.Arguments)
	}

	// Take the target addresses given and create a file
	inFile, err := os.Create(filepath.Join(h.wd, "input.txt"))
	if err != nil {
		log.WithFields(log.Fields{
			"file":  inFile,
			"error": err.Error(),
		}).Error("Unable to create input target list file")
		return &nmapTasker{}, err
	}

	inFile.WriteString(t.job.Parameters["targets"])

	// Append that file to the arguments
	args = append(args, "-iL", filepath.Join(h.wd, "input.txt"))

	log.WithField("arguments", args).Debug("Arguments complete")

	h.start = append(h.start, args...)
	h.resume = append(h.resume, "--resume", filepath.Join(t.wd, "greppable-output.txt"))

	return &h, nil
}

func (v *nmapTasker) Status() common.Job {
	log.WithField("task", v.job.UUID).Debug("Gathering task details")
	v.mux.Lock()
	defer v.mux.Unlock()

	index := regLastStatusIndex.FindAllStringIndex(v.stdout.String(), -1)
	if len(index) >= 1 {
		// We found a status so start processing the last status in Stdout
		status := string(v.stdout.Bytes()[index[len(index)-1][0]:])

		//Time to gather the progress
		progMatch := regProgress.FindStringSubmatch(status)
		log.WithField("progMatch", progMatch).Debug("Matching progress info")

		if len(progMatch) == 4 {
			prog, err := strconv.ParseFloat(progMatch[3], 64)
			if err == nil {
				v.job.Progress = prog
				log.WithField("progress", v.job.Progress).Debug("Job progress updated.")
			} else {
				log.WithField("error", err.Error()).Error("There was a problem converting progress to a number.")
			}
		}

		etcMatch := regTimeEstimated.FindStringSubmatch(status)
		log.WithField("etcMatch", etcMatch).Debug("Matching estimated time of completion.")
		if len(etcMatch) == 2 {
			v.job.ETC = etcMatch[1]
		}

		// Get the speed of one or more GPUs
		speeds := regGPUSpeed.FindAllStringSubmatch(status, -1)
		if len(speeds) > 1 {
			// We have more than one GPU so loop through and find the combined total
			for _, speedString := range speeds {
				if speedString[1] == "*" && len(speedString) == 4 {
					// We have the total so grab the pieces
					timestamp := fmt.Sprintf("%d", time.Now().Unix())

					// Check if we have a performance unit yet
					if v.job.PerformanceTitle == "" {
						// We don't so just take the one provided
						v.job.PerformanceTitle = speedString[3]

						v.job.PerformanceData[timestamp] = speedString[2]
					} else {
						// See what we need to do with the number to match our
						// original units
						var mag float64
						switch v.job.PerformanceTitle {
						case "H/s":
							mag = speedMagH[speedString[3]]
						case "kH/s":
							mag = speedMagK[speedString[3]]
						case "MH/s":
							mag = speedMagM[speedString[3]]
						case "GH/s":
							mag = speedMagG[speedString[3]]
						}

						// Convert our string into a float
						speed, err := strconv.ParseFloat(speedString[2], 64)
						if err == nil {
							// change magnitude and save as string
							v.job.PerformanceData[timestamp] = fmt.Sprintf("%f", speed*mag)
							log.WithFields(log.Fields{
								"speed": speed,
								"mag":   mag,
							}).Debug("Speed calculated.")
						}
					}
				}
			}
		} else if len(speeds) == 1 {
			// We have just one GPU
			speedString := speeds[0]
			if speedString[1] == "1" && len(speedString) == 4 {
				// We have the total so grab the pieces
				timestamp := fmt.Sprintf("%d", time.Now().Unix())

				// Check if we have a performance unit yet
				if v.job.PerformanceTitle == "" {
					// We don't so just take the one provided
					v.job.PerformanceTitle = speedString[3]

					v.job.PerformanceData[timestamp] = speedString[2]
				} else {
					// See what we need to do with the number to match our
					// original units
					var mag float64
					switch v.job.PerformanceTitle {
					case "H/s":
						mag = speedMagH[speedString[3]]
					case "kH/s":
						mag = speedMagK[speedString[3]]
					case "MH/s":
						mag = speedMagM[speedString[3]]
					case "GH/s":
						mag = speedMagG[speedString[3]]
					}

					// Convert our string into a float
					speed, err := strconv.ParseFloat(speedString[2], 64)
					if err == nil {
						// change magnitude and save as string
						v.job.PerformanceData[timestamp] = fmt.Sprintf("%f", speed*mag)
						log.WithFields(log.Fields{
							"speed": speed,
							"mag":   mag,
						}).Debug("Speed calculated.")
					}
				}
			}
		}

		// Check for number of recovered hashes
		recovered := regRecovered.FindStringSubmatch(status)
		log.WithField("recovered", recovered).Debug("Recovered hashes.")
		if len(recovered) == 3 {
			if r, err := strconv.ParseInt(recovered[1], 10, 64); err == nil {
				v.job.CrackedHashes = r
			}

			if r, err := strconv.ParseInt(recovered[2], 10, 64); err == nil {
				v.job.TotalHashes = r
			}
		}
	}

	// Get the output results
	if file, err := os.Open(filepath.Join(v.wd, "hashes-output.txt")); err == nil {
		log.Debug("Checking hashes-output file")
		linescanner := bufio.NewScanner(file)
		var linetmp [][]string
		for linescanner.Scan() {
			var kvp []string
			i := strings.LastIndex(linescanner.Text(), ":")
			kvp = append(kvp, linescanner.Text()[:i])
			kvp = append(kvp, linescanner.Text()[i+1:])

			linetmp = append(linetmp, kvp)
		}
		if len(linetmp) > 0 {
			v.job.OutputData = linetmp
		}
	}

	log.WithFields(log.Fields{
		"Stdout": v.stdout,
		"Stderr": v.stderr,
	}).Debug("Stdout & Stderr")

	v.stdout.Reset()

	v.job.Error = v.stderr.String()

	log.WithFields(log.Fields{
		"task":   v.job.UUID,
		"status": v.job.Status,
	}).Info("Ongoing task status")

	return v.job
}

func (v *nmapTasker) Run() error {
	v.mux.Lock()
	defer v.mux.Unlock()
	// Check that we have not already finished this job
	done := v.job.Status == common.STATUS_DONE || v.job.Status == common.STATUS_QUIT || v.job.Status == common.STATUS_FAILED
	if done {
		log.WithField("Status", v.job.Status).Debug("Unable to start hashcatdict job")
		return errors.New("Job already finished.")
	}

	// Check if this job is running
	if v.job.Status == common.STATUS_RUNNING {
		// Job already running so return no errors
		log.Debug("hashcatdict job already running, doing nothing")
		return nil
	}

	// Set commands for restore or start
	if v.job.Status == common.STATUS_CREATED {
		v.cmd = *exec.Command(config.BinPath, v.start...)
	} else {
		v.cmd = *exec.Command(config.BinPath, v.resume...)
	}

	v.cmd.Dir = v.wd

	log.WithFields(log.Fields{
		"status": v.job.Status,
		"dir":    v.cmd.Dir,
	}).Debug("Setup exec.command")

	// Assign the stderr, stdout, stdin pipes
	var err error
	v.stderrPipe, err = v.cmd.StderrPipe()
	if err != nil {
		return err
	}

	v.stdoutPipe, err = v.cmd.StdoutPipe()
	if err != nil {
		return err
	}

	v.stdinPipe, err = v.cmd.StdinPipe()
	if err != nil {
		return err
	}

	v.stderr = bytes.NewBuffer([]byte(""))
	v.stdout = bytes.NewBuffer([]byte(""))

	go func() {
		for {
			io.Copy(v.stderr, v.stderrPipe)
		}
	}()
	go func() {
		for {
			io.Copy(v.stdout, v.stdoutPipe)
		}
	}()

	// Start the command
	log.WithField("argument", v.cmd.Args).Debug("Running command.")
	err = v.cmd.Start()
	if err != nil {
		// We had an error starting to return that and quit the job
		v.job.Status = common.STATUS_FAILED
		log.Errorf("There was an error starting the job: %v", err)
		return err
	}

	v.job.StartTime = time.Now()
	v.job.Status = common.STATUS_RUNNING

	// Build goroutine to alert that the job has finished
	go func() {
		// Listen on commmand wait and then send signal when finished
		// This will be read on the Status() function
		v.cmd.Wait()

		v.mux.Lock()
		v.job.Status = common.STATUS_DONE
		v.job.Progress = 100.00
		v.waitChan <- struct{}{}
		v.mux.Unlock()
	}()

	return nil
}

// Pause the hashcat run
func (v *nmapTasker) Pause() error {
	log.WithField("task", v.job.UUID).Debug("Attempting to pause hashcatdict task")

	// Call status to update the job internals before pausing
	v.Status()

	v.mux.Lock()

	// Because this is queue managed, we should just need to kill the process.
	// It will be resumed automatically
	if runtime.GOOS == "windows" {
		v.cmd.Process.Kill()
	} else {
		v.cmd.Process.Signal(syscall.SIGINT)
	}

	v.mux.Unlock()

	// Wait for the program to actually exit
	<-v.waitChan

	// Change status to pause
	v.mux.Lock()
	v.job.Status = common.STATUS_PAUSED
	v.mux.Unlock()

	log.WithField("task", v.job.UUID).Debug("Task paused successfully")

	return nil
}

func (v *nmapTasker) Quit() common.Job {
	log.WithField("task", v.job.UUID).Debug("Attempting to quit hashcatdict task")

	// Call status to update the job internals before quiting
	v.Status()

	v.mux.Lock()

	if runtime.GOOS == "windows" {
		v.cmd.Process.Kill()
	} else {
		v.cmd.Process.Signal(syscall.SIGINT)
	}

	v.mux.Unlock()

	// Wait for the program to actually exit
	<-v.waitChan

	v.mux.Lock()
	v.job.Status = common.STATUS_QUIT
	v.mux.Unlock()

	log.WithField("task", v.job.UUID).Debug("Task quit successfully")

	return v.job
}

func (v *nmapTasker) IOE() (io.Writer, io.Reader, io.Reader) {
	return v.stdinPipe, v.stdoutPipe, v.stderrPipe
}
