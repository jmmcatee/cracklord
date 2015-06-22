package hashcatdict

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

var regLastStatusIndex *regexp.Regexp
var regStatus *regexp.Regexp
var regRuleType *regexp.Regexp
var regInputMode *regexp.Regexp
var regHashTarget *regexp.Regexp
var regHashType *regexp.Regexp
var regTimeStarted *regexp.Regexp
var regTimeEstimated *regexp.Regexp
var regGPUSpeed *regexp.Regexp
var regRecovered *regexp.Regexp
var regProgress *regexp.Regexp
var regRejected *regexp.Regexp
var regGPUHWMon *regexp.Regexp

var regGetGPUCount *regexp.Regexp
var regGetNumerator *regexp.Regexp
var regGetDenominator *regexp.Regexp
var regGetPercent *regexp.Regexp

var speedMagH = map[string]float64{
	"H/s":  1,
	"kH/s": 1000,
	"MH/s": 1000000,
	"GH/s": 1000000000,
}

var speedMagK = map[string]float64{
	"H/s":  1 / 1000,
	"kH/s": 1,
	"MH/s": 1000,
	"GH/s": 1000000,
}

var speedMagM = map[string]float64{
	"H/s":  1 / 1000000,
	"kH/s": 1 / 1000,
	"MH/s": 1,
	"GH/s": 1000,
}

var speedMagG = map[string]float64{
	"H/s":  1 / 1000000000,
	"kH/s": 1 / 1000000,
	"MH/s": 1 / 1000,
	"GH/s": 1,
}

func init() {
	var err error
	regLastStatusIndex, err = regexp.Compile(`Session\.Name\.\.\.\:`)
	regStatus, err = regexp.Compile(`Status\.\.\.\.\.\.\.\.\.\:\s+(\w+)`)
	regRuleType, err = regexp.Compile(`Rules\.Type\.\.\.\.\.\:\s+(\w+)\s+\((.+)\)`)
	regInputMode, err = regexp.Compile(`Input\.Mode\.\.\.\.\.\:\s+(\w+)\s+\((.+)\)`)
	regHashTarget, err = regexp.Compile(`Hash\.Target\.\.\.\.\:\s+([0-9a-fA-F]+)`)
	regHashType, err = regexp.Compile(`Hash\.Type\.\.\.\.\.\.\:\s+(\w+)`)
	regTimeStarted, err = regexp.Compile(`Time\.Started\.\.\.\:\s+(.+)\(.+\)`)
	regTimeEstimated, err = regexp.Compile(`Time\.Estimated\.: .*\((.*?)\)`)
	regGPUSpeed, err = regexp.Compile(`Speed\.GPU\.#([\d|\*]+)\.\.\.\:\s+(\d+\.\d+)\s+(.H/s)`)
	regRecovered, err = regexp.Compile(`Recovered\.+:\s+(\d+)\/(\d+)`)
	regProgress, err = regexp.Compile(`Progress\.{7}: (\d*)/(\d*) \((\d{1,3}\.\d{2})%\)`)
	regRejected, err = regexp.Compile(`(Rejected)\.\.\.\.\.\.\.\:\s+(\d+\/\d+.+)`)
	regGPUHWMon, err = regexp.Compile(`(HWMon\.GPU\.#\d+)\.\.\.\:\s+(.+)`)

	regGetGPUCount, err = regexp.Compile(`\#(\d)`)
	regGetNumerator, err = regexp.Compile(`(\d+\)/\d+`)
	regGetDenominator, err = regexp.Compile(`(d+\/(\d+)`)
	regGetPercent, err = regexp.Compile(`\(\d+\.\d+\%\)`)

	if err != nil {
		panic(err.Error())
	}
}

type hascatTasker struct {
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
	h := hascatTasker{}
	h.waitChan = make(chan struct{}, 1)

	h.job = j

	// Build a working directory for this job
	h.wd = filepath.Join(config.WorkDir, h.job.UUID)
	err := os.Mkdir(h.wd, 0700)
	if err != nil {
		// Couldn't make a directory so kill the job
		log.WithFields(log.Fields{
			"path":  h.wd,
			"error": err.Error(),
		}).Error("hashcatdict could not create a working directory")
		return &hascatTasker{}, errors.New("Could not create a working directory.")
	}
	log.WithField("path", h.wd).Debug("Working directory created")

	// Build the arguements for hashcat
	args := []string{}

	// Get the hash type and add an argument
	htype, ok := config.HashTypes[h.job.Parameters["algorithm"]]
	if !ok {
		log.WithFields(log.Fields{
			"algoritm": htype,
			"err":      ok,
		}).Error("Could not find the algorithm provided")
		return &hascatTasker{}, errors.New("Could not find the algorithm provided.")
	}
	log.WithField("algorithm", htype).Debug("Added algorithm")

	args = append(args, "-m", htype)

	// Add the rule file to use if one was given
	ruleKey, ok := h.job.Parameters["rules"]
	if ok {
		// We have a rule file, check for blank
		if ruleKey != "" {
			rulePath, ok := config.Rules[ruleKey]
			if ok {
				args = append(args, "-r", rulePath)
			}
		}
	}
	log.WithField("rules", ruleKey).Debug("Added rules")

	args = append(args, "--status", "--status-timer=10", "--force")

	// Add an output file
	args = append(args, "-o", filepath.Join(h.wd, "hashes-output.txt"))

	//Append config file arguments
	if config.Arguments != "" {
		args = append(args, config.Arguments)
	}

	// Take the hashes given and create a file
	hashFile, err := os.Create(filepath.Join(h.wd, "hashes.txt"))
	if err != nil {
		log.WithFields(log.Fields{
			"file":  hashFile,
			"error": err.Error(),
		}).Error("Unable to create hash file")
		return &hascatTasker{}, err
	}
	log.WithField("hashfile", hashFile).Debug("Created hashfile")

	hashFile.WriteString(h.job.Parameters["hashes"])

	var lines int64
	linescanner := bufio.NewScanner(hashFile)
	for linescanner.Scan() {
		lines++
	}

	h.job.TotalHashes = lines

	// Append that file to the arguments
	args = append(args, filepath.Join(h.wd, "hashes.txt"))

	// Check for dictionary given
	dictKey, ok := h.job.Parameters["dictionaries"]
	if !ok {
		log.Error("No dictionary was provided.")
		return &hascatTasker{}, errors.New("No dictionary provided.")
	}

	dictPath, ok := config.Dictionaries[dictKey]
	if !ok {
		log.Error("Dictionary key provided was not present")
		return &hascatTasker{}, errors.New("Dictionary key provided was not present.")
	}

	// Check for additions to the dictionary
	if h.job.Parameters["customdictadd"] != "" {
		// We need to prepend the values here to a dictionary
		newDictPath := filepath.Join(h.wd, "custom-dict-"+dictKey+".txt")
		newDict, err := os.Create(newDictPath)
		if err != nil {
			log.Error("Custom dictionary file could not be created.")
			return &hascatTasker{}, errors.New("Custom dictionary file could not be created.")
		}

		// Copy the user content into the file
		newDict.WriteString(h.job.Parameters["customdictadd"])

		// Get the contents of the dictionary and append it to the new file
		dictFile, err := os.Open(dictPath)
		if err != nil {
			log.Error("Dictionary could not be opened to copy to the custom dictionary.")
			return &hascatTasker{}, errors.New("Dictionary could not be opened to copy to the custom dictionary.")
		}

		io.Copy(newDict, dictFile)

		// Finally let's change the dictPath to the new file
		dictPath = newDictPath
	}

	// Add dictionary to arguments
	log.WithField("dictionary", dictPath).Debug("Dictionary added")
	args = append(args, dictPath)

	log.WithField("arguments", args).Debug("Arguments complete")

	// Get everything except the session identifier because the Resume command will be different
	h.start = append(h.start, "--session="+h.job.UUID)
	h.resume = append(h.resume, "--session="+h.job.UUID)
	h.resume = append(h.resume, "--restore")

	h.start = append(h.start, args...)
	h.resume = append(h.resume, args...)

	// Configure the return values
	h.job.OutputTitles = []string{"Hash", "Plaintext"}

	return &h, nil
}

func (v *hascatTasker) Status() common.Job {
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
			kvp = append(kvp, linescanner.Text()[:i-1])
			kvp = append(kvp, linescanner.Text()[i+1:])

			linetmp = append(linetmp)
		}
		if len(linetmp) > 0 {
			v.job.OutputData = linetmp
		}
	}

	v.stdout.Reset()

	v.job.Error = v.stderr.String()

	log.WithFields(log.Fields{
		"task":   v.job.UUID,
		"status": v.job.Status,
	}).Info("Ongoing task status")

	return v.job
}

func (v *hascatTasker) Run() error {
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
		v.waitChan <- struct{}{}
		v.mux.Unlock()
	}()

	return nil
}

// Pause the hashcat run
func (v *hascatTasker) Pause() error {
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

func (v *hascatTasker) Quit() common.Job {
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

func (v *hascatTasker) IOE() (io.Writer, io.Reader, io.Reader) {
	return v.stdinPipe, v.stdoutPipe, v.stderrPipe
}
