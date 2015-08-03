package johndict

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/jmmcatee/cracklord/common"
)

/*
	Regex to use throughout the plugin
*/
var regStatusLine *regexp.Regexp

// Magnitude tables
var speedMagH = map[string]float64{
	"C/s":  1,
	"KC/s": 1000,
	"MC/s": 1000000,
	"GC/s": 1000000000,
}

var speedMagK = map[string]float64{
	"C/s":  1 / 1000,
	"KC/s": 1,
	"MC/s": 1000,
	"GC/s": 1000000,
}

var speedMagM = map[string]float64{
	"C/s":  1 / 1000000,
	"KC/s": 1 / 1000,
	"MC/s": 1,
	"GC/s": 1000,
}

var speedMagG = map[string]float64{
	"C/s":  1 / 1000000000,
	"KC/s": 1 / 1000000,
	"MC/s": 1 / 1000,
	"GC/s": 1,
}

/*
	Function that runs on the setup of this package once and only once, useful
	for one time setup of things
*/
func init() {
	var err error
	// The regular expression captures are as follows
	// 1 - Number of successful guessed passwords
	// 2 - Session run time in D:HH:MM:SS format
	// 3 - Percentage complete
	// 4 - The estimated time to completion
	// 5 - Number of guesses per second
	// 6 - Units for guesses per second
	regStatusLine, err = regexp.Compile(`(\d+)g (\d+:\d+:\d+:\d+) (?:(\d+.\d+)% \(ETA: ((?:\d+\-\d+\-\d+ \d+:\d+)|(?:\d+:\d+:\d+))\))? \d+g\/s \d+.?p\/s \d+.?c\/s (\d+)(.?C/s)`)

	if err != nil {
		panic("Error during John Dict setup: " + err.Error())
	}
}

/*
	Struct to hold our interface functions and some basic data that you would
	probably want
*/
type johndictTasker struct {
	mux          sync.Mutex
	job          common.Job
	wd           string
	cmd          exec.Cmd
	args         []string
	stderr       *bytes.Buffer
	stdout       *bytes.Buffer
	stderrPipe   io.ReadCloser
	stdoutPipe   io.ReadCloser
	stdinPipe    io.WriteCloser
	doneWaitChan chan struct{}
}

/*
	Create a new tasker for this tool.  As noted in the "tooler", the "tasker"
	is used to actually run individual jobs as tasks on the individual resource
	server
*/
func newJohnDictTask(j common.Job) (common.Tasker, error) {
	//You can also log.Fatal (which will kill the resourceserver, so avoid), log.Error, log.Warn, and log.Info
	log.Debug("Creating a new John Dict Tasker")

	v := johndictTasker{}
	v.doneWaitChan = make(chan struct{}, 1)

	// Assign the job information
	v.job = j

	// Build the working directory from the configuration and job UUID
	v.wd = filepath.Join(config.WorkingDir, v.job.UUID)

	// Create the working directory
	err := os.Mkdir(v.wd, 0700)
	if err != nil {
		// We could not make the directory so fail the job
		log.WithFields(log.Fields{
			"path":       v.wd,
			"mkdirerror": err.Error(),
		}).Error("Could not create the John Dict working directory")
		return &johndictTasker{}, err
	}

	log.WithField("path", v.wd).Debug("Created working directory")

	// Build the argument string for John
	args := []string{}

	// Get the format type
	var format string
	var ok bool
	for _, f := range config.Formats {
		if v.job.Parameters["algorithm"] == f {
			format = f
			ok = true
		}
	}
	if !ok {
		log.WithFields(log.Fields{
			"format": format,
		}).Error("Could not find format provided")
		return &johndictTasker{}, errors.New("Could not find format provided")
	}
	args = append(args, "--format="+format)
	log.WithField("format", format).Debug("Added algorithm")

	args = append(args, "--session="+v.job.UUID)
	args = append(args, "--pot="+v.job.UUID)

	// Add the dictionary files given
	dictKey, ok := v.job.Parameters["dictionaries"]
	if !ok {
		log.Error("No dictionary was provdied")
		return &johndictTasker{}, errors.New("No dictionary provided.")
	}

	dictPath, ok := config.Dictionaries[dictKey]
	if !ok {
		log.Error("Dictionary key provdied was not present")
		return &johndictTasker{}, errors.New("Dictionary key provided was not present")
	}

	// Check for additions to the dictionary
	if v.job.Parameters["customdictadd"] != "" {
		// We need to prepend the values here to a dictionary
		newDictPath := filepath.Join(v.wd, "custom-dict-"+dictKey+".txt")
		newDict, err := os.Create(newDictPath)
		if err != nil {
			log.Error("Custom dictionary file could not be created.")
			return &johndictTasker{}, errors.New("Custom dictionary file could not be created.")
		}

		// Copy the user content into the file
		newDict.WriteString(v.job.Parameters["customdictadd"])

		// Get the contents of the dictionary and append it to the new file
		dictFile, err := os.Open(dictPath)
		if err != nil {
			log.Error("Dictionary could not be opened to copy to the custom dictionary.")
			return &johndictTasker{}, errors.New("Dictionary could not be opened to copy to the custom dictionary.")
		}

		io.Copy(newDict, dictFile)

		// Finally let's change the dictPath to the new file
		dictPath = newDictPath
	}

	// Add dictionary to arguments
	log.WithField("dictionary", dictPath).Debug("Dictionary added")
	args = append(args, "--wordlist="+dictPath)

	// Add a rule file
	var rule string
	ok = false
	for _, r := range config.Rules {
		if v.job.Parameters["rules"] == r {
			rule = r
			ok = true
		}
	}
	if !ok {
		log.WithFields(log.Fields{
			"format": rule,
		}).Error("Could not find rule provided")
		return &johndictTasker{}, errors.New("Could not find rule provided")
	}
	args = append(args, "--rules="+rule)
	log.WithField("rules", rule).Debug("Added rule section")

	// Append config file arguments
	if config.Arguments != "" {
		args = append(args, config.Arguments)
	}

	// Take the hashes given and create a file
	hashFilePath := filepath.Join(v.wd, "hashes.txt")
	hashFile, err := os.Create(hashFilePath)
	if err != nil {
		log.WithFields(log.Fields{
			"file":  hashFilePath,
			"error": err.Error(),
		}).Error("Unable to create hash file")
		return &johndictTasker{}, err
	}
	log.WithField("hashfile", hashFilePath).Debug("Created hashfile")

	args = append(args, hashFilePath)

	hashFile.WriteString(v.job.Parameters["hashes"])

	var lines int64
	linescanner := bufio.NewScanner(hashFile)
	for linescanner.Scan() {
		lines++
	}

	v.job.TotalHashes = lines

	// Setup start and resume arguements
	v.args = args

	// Configure return values
	v.job.OutputTitles = []string{"Plaintext", "Hash"}

	return &v, nil
}

/*
	Gather a status of job information for this running task.  This will be used
	by the queue server every so often (5-30 seconds typically) to keep job
	information
*/
func (v *johndictTasker) Status() common.Job {
	log.WithField("task", v.job.UUID).Debug("Gathering task status")
	v.mux.Lock()
	defer v.mux.Unlock()

	// Run john --status command
	statusExec := exec.Command(config.BinPath, "--status="+v.job.UUID)
	statusExec.Dir = v.wd
	status, err := statusExec.CombinedOutput()
	if err != nil {
		v.job.Error = err.Error()
		log.WithField("Error", err.Error()).Debug("Error running john status command.")
		return v.job
	}

	log.WithField("StatusStdout", string(status)).Debug("Stdout status return of john call")

	timestamp := fmt.Sprintf("%d", time.Now().Unix())

	match := regStatusLine.FindStringSubmatch(string(status))
	log.WithField("StatusMatch", match).Debug("Regex match of john status call")

	if len(match) != 6 {
		// The ETA might not be printing so let's try form STDOUT
		ic, err := io.WriteString(v.stdinPipe, "nnn")
		if err != nil {
			log.Debug("Error writing to StdinPipe")
		}
		log.WithField("numBytes", ic).Debug("Bytes written to john's stdin")
		i := strings.LastIndex(v.stdout.String(), "\n")
		if i != -1 {
			match = regStatusLine.FindStringSubmatch(v.stdout.String()[i:])
		}
	}

	if len(match) == 6 {
		// Get # of cracked hashes
		crackedHashes, err := strconv.ParseInt(match[1], 10, 64)
		if err == nil {
			v.job.CrackedHashes = crackedHashes
		}

		// Get % complete
		progress, err := strconv.ParseFloat(match[3], 64)
		if err == nil {
			v.job.Progress = progress
		}

		// Get ETA
		eta, err := parseJohnETA(match[4])
		if err == nil {
			v.job.ETC = printTimeUntil(eta)
		}

		// Get guesses / second
		if v.job.PerformanceTitle == "" {
			// We need to set the units the first time
			v.job.PerformanceTitle = match[6]
		}

		var mag float64
		switch v.job.PerformanceTitle {
		case "C/s":
			mag = speedMagH[match[6]]
		case "KC/s":
			mag = speedMagK[match[6]]
		case "MC/s":
			mag = speedMagM[match[6]]
		case "GC/s":
			mag = speedMagG[match[6]]
		}

		// convert our string into a float
		speed, err := strconv.ParseFloat(match[5], 64)
		if err == nil {
			v.job.PerformanceData[timestamp] = fmt.Sprintf("%f", speed*mag)
			log.WithFields(log.Fields{
				"speed": speed,
				"mag":   mag,
			}).Debug("Speed calculated.")
		}

	} else {
		log.WithField("MatchCount", len(match)).Debug("Did not match enough items in the status")
	}

	// Now get any hashes we might have cracked. Because of how John works we will
	// need to read in all the hashes provied and then search the .pot file in this
	// working directory to find any cracked passwords.
	var hash2D [][]string

	// Read in hashes.txt file & pot file
	hashFile, err := ioutil.ReadFile(filepath.Join(v.wd, "hashes.txt"))
	potFile, err := ioutil.ReadFile(filepath.Join(v.wd, v.job.UUID+".pot"))
	potHashes := strings.Split(string(potFile), "\n")
	if err == nil {
		hashes := strings.Split(string(hashFile), "\n")
		for _, hash := range hashes {
			// Check for existence in potHashes
			for _, potHash := range potHashes {
				if strings.Contains(potHash, strings.ToLower(hash)) {
					// We have a hash match so let's add it to our output
					hashIndex := strings.Index(potHash, strings.ToLower(hash))
					var hashKeyPair []string
					hashKeyPair = append(hashKeyPair, potHash[hashIndex+1:])
					hashKeyPair = append(hashKeyPair, hash)

					hash2D = append(hash2D, hashKeyPair)
				}
			}
		}

		v.job.OutputData = hash2D
	}

	log.WithFields(log.Fields{
		"task":   v.job.UUID,
		"status": v.job.Status,
	}).Info("Ongoing task status")

	return v.job
}

/*
	Start or restart a job running under this tool.
*/
func (v *johndictTasker) Run() error {
	// Grab a Lock
	v.mux.Lock()
	defer v.mux.Unlock()

	// Check for the status of this job
	if common.IsDone(v.job.Status) {
		log.WithField("Status", v.job.Status).Debug("Unable to start johndict job")
		return errors.New("Job has already finished.")
	}

	// Check if this job is running
	if common.IsRunning(v.job.Status) {
		log.WithField("Status", v.job.Status).Debug("Johndict job is already running.")
		return nil
	}

	// Set commands for first start or restoring
	if common.IsNew(v.job.Status) {
		v.cmd = *exec.Command(config.BinPath, v.args...)
	} else {
		restoreArgs := []string{"--restore=" + v.job.UUID}
		v.cmd = *exec.Command(config.BinPath, restoreArgs...)
	}

	v.cmd.Dir = v.wd

	log.WithFields(log.Fields{
		"status": v.job.Status,
		"dir":    v.cmd.Dir,
	}).Debug("Setup exec.command")

	// Assign the Stderr, Stdout, and Stdin pipes
	var pipeError error
	v.stderrPipe, pipeError = v.cmd.StderrPipe()
	if pipeError != nil {
		return pipeError
	}
	v.stdoutPipe, pipeError = v.cmd.StdoutPipe()
	if pipeError != nil {
		return pipeError
	}
	v.stdinPipe, pipeError = v.cmd.StdinPipe()
	if pipeError != nil {
		return pipeError
	}

	v.stderr = bytes.NewBuffer([]byte(""))
	v.stdout = bytes.NewBuffer([]byte(""))

	// Start goroutine to copy data from stderr and stdout pipe
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
	log.WithField("Arguments", v.cmd.Args).Debug("Running the command")
	err := v.cmd.Start()
	if err != nil {
		v.job.Status = common.STATUS_FAILED
		log.WithField("Start Error", err.Error())
		return err
	}

	v.job.StartTime = time.Now()
	v.job.Status = common.STATUS_RUNNING

	// Goroutine to change status once the external executable quits
	go func() {
		v.cmd.Wait()

		// The exec command has finished running
		v.mux.Lock()
		v.job.Status = common.STATUS_DONE
		v.job.Progress = 100.00
		v.doneWaitChan <- struct{}{}
		v.mux.Unlock()
	}()

	return nil
}

/*
	Pause the job if possible, save the state, etc.
*/
func (v *johndictTasker) Pause() error {
	log.WithField("Task", v.job.UUID).Debug("Attempt to pause johndict job")

	// Update internal status
	v.Status()

	v.mux.Lock()

	// Kill the process after a SIGHUP
	v.cmd.Process.Signal(syscall.SIGHUP)
	v.cmd.Process.Kill()

	v.mux.Unlock()

	// Wait for the program to actually exit
	<-v.doneWaitChan

	// Change the status to paused
	v.mux.Lock()
	v.job.Status = common.STATUS_PAUSED
	v.mux.Unlock()

	log.WithField("Task", v.job.UUID).Debug("Task has been paused successfully.")

	return nil
}

/*
	Stop the job running under this tool, don't forget to cleanup and file system
	resources, etc.
*/
func (v *johndictTasker) Quit() common.Job {
	log.WithField("Task", v.job.UUID).Debug("Attempting to quit johndict task.")

	// Update the jobs status
	log.Debug("Getting status before quit")
	v.Status()

	v.mux.Lock()

	// Kill the process after a SIGHUP
	log.Debug("Sending SIGHUP before process kill")
	v.cmd.Process.Signal(syscall.SIGHUP)
	log.Debug("Sending kill signal to process")
	v.cmd.Process.Kill()

	v.mux.Unlock()

	// Wait for the program to actually exit
	log.Debug("Waiting on the process to finish")
	<-v.doneWaitChan

	// Change the status to paused
	log.Debug("Change status")
	v.mux.Lock()
	v.job.Status = common.STATUS_QUIT
	v.mux.Unlock()

	log.WithField("Task", v.job.UUID).Debug("Task has been quit successfully.")

	return v.job
}

/*
	Get the input, output, and error streams
*/
func (v *johndictTasker) IOE() (io.Writer, io.Reader, io.Reader) {
	return v.stdinPipe, v.stdoutPipe, v.stderrPipe
}

func parseJohnETA(eta string) (time.Time, error) {
	// First check for the format of the ETA
	if len(eta) == 8 {
		t := time.Now().UTC()
		parts := strings.Split(eta, ":")

		// Add hours
		sec, err := time.ParseDuration(parts[2] + "s")
		min, err := time.ParseDuration(parts[1] + "m")
		h, err := time.ParseDuration(parts[0] + "h")
		if err != nil {
			return time.Time{}, err
		}

		t = t.Add(sec)
		t = t.Add(min)
		t = t.Add(h)

		// Prepend the current YYYY-MM-DD
		year := strconv.Itoa(t.Year())
		month := strconv.Itoa(int(t.Month()))
		day := strconv.Itoa(t.Day())
		hour := strconv.Itoa(t.Hour())
		minute := strconv.Itoa(t.Minute())
		second := strconv.Itoa(t.Second())

		if int(time.Now().Month()) < 10 {
			month = "0" + month
		}

		if t.Minute() < 10 {
			minute = "0" + minute
		}

		if t.Hour() < 10 {
			hour = "0" + hour
		}

		if t.Second() < 10 {
			second = "0" + second
		}

		eta = year + "-" + month + "-" + day + " " + hour + ":" + minute + ":" + second
	} else if len(eta) == 16 {
		eta = eta + ":00"
	} else {
		// Neither correct size for provided so return an error
		return time.Time{}, errors.New("Time format provided was not supported.")
	}

	t, err := time.Parse("2006-01-02 15:04:05", eta)
	if err != nil {
		return time.Time{}, err
	}

	return t, nil
}

// Print the duration from now until ETA in a pretty fashion
func printTimeUntil(eta time.Time) string {
	// First let's get the duration
	d := eta.Sub(time.Now().UTC())

	// Need to decide to present in days, hours, minutes, or seconds
	switch {
	case d.Hours() < 0.016667:
		// Seconds
		return fmt.Sprintf("%.0f seconds", math.Floor(d.Seconds()))
	case d.Hours() < 1.000:
		// minutes
		return fmt.Sprintf("%.0f minutes, %d seconds", math.Floor(d.Minutes()), int64(d.Seconds())%60)
	case d.Hours() < 24.000:
		// hours
		return fmt.Sprintf("%.0f hours, %d minutes", math.Floor(d.Hours()), int64(math.Floor(d.Minutes()))%60)
	case d.Hours() >= 24.0000:
		// days
		return fmt.Sprintf("%.0f days, %d hours", math.Floor(d.Hours())/24, int64(d.Hours())%24)
	}

	return "error"
}
