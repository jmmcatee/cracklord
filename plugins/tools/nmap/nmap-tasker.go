package nmap

import (
	"bytes"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/jmmcatee/cracklord/common"
	"io"
	"net"
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
var regPerformance *regexp.Regexp

func init() {
	var err error
	regHostsCompleted, err = regexp.Compile(`(\d+) hosts completed`)
	regTimeEstimate, err = regexp.Compile(`About (\d{1,3}\.\d\d)% done;.*\((.*)\)`)
	regPerformance, err = regexp.Compile(`Current sending rates: ([\d\.]*) packets / s`)

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

func newNmapTask(j common.Job) (common.Tasker, error) {
	t := nmapTasker{}
	t.waitChan = make(chan struct{}, 1)

	t.job = j

	// Build a working directory for this job
	t.wd = filepath.Join(config.WorkDir, t.job.UUID)
	err := os.Mkdir(t.wd, 0700)
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
	args = append(args, "-d", "--stats-every=10s")

	// Get the scan type from the job
	scantypekey, ok := t.job.Parameters["scantype"]
	if !ok {
		log.WithFields(log.Fields{
			"algoritm": t.job.Parameters["scantype"],
		}).Error("Could not find the scan type provided")
		return &nmapTasker{}, errors.New("Could not find the scan type provided.")
	}
	args = append(args, scanTypes[scantypekey])

	// Get the timing settings we should be using
	timingkey, ok := t.job.Parameters["timing"]
	if !ok {
		log.WithFields(log.Fields{
			"timing": t.job.Parameters["timing"],
		}).Error("Could not find the scan type provided")
		return &nmapTasker{}, errors.New("Could not find the scan type provided.")
	}
	args = append(args, timingSettings[timingkey])

	// Time to setup the ports
	portkey, ok := t.job.Parameters["ports"]
	if !ok {
		log.WithFields(log.Fields{
			"ports": t.job.Parameters["ports"],
		}).Error("Could not gather port definition from job parameters.")
		return &nmapTasker{}, errors.New("Could not gather port definition from job parameters.")
	}
	if portkey == "* Custom Port Listing" {
		customportdata, ok := t.job.Parameters["portscustom"]
		if !ok {
			log.WithFields(log.Fields{
				"timing": t.job.Parameters["portscustom"],
			}).Error("Could not find the custom port definitions.")
			return &nmapTasker{}, errors.New("Could not find the custom port definitions.")
		}
		args = append(args, "-p"+customportdata)
	} else {
		ports, ok := portSettings[portkey]
		if !ok {
			log.WithFields(log.Fields{
				"timing": t.job.Parameters["portscustom"],
			}).Error("Could not find the port definition requested.")
			return &nmapTasker{}, errors.New("Could not find the port definition requested.")
		}
		args = append(args, "-p"+ports)
	}

	serviceversion, ok := t.job.Parameters["serviceversion"]
	if ok {
		if serviceversion == "true" {
			args = append(args, "-sV")
		}
	}

	hostdiscovery, ok := t.job.Parameters["skiphostdiscovery"]
	if ok {
		if hostdiscovery == "true" {
			args = append(args, "-Pn")
		}
	}

	// Add our output files
	args = append(args, "-oG", filepath.Join(t.wd, "output.grep"))
	args = append(args, "-oX", filepath.Join(t.wd, "output.xml"))

	//Append config file arguments
	if config.Arguments != "" {
		args = append(args, config.Arguments)
	}

	// Take the target addresses given and create a file
	inFile, err := os.Create(filepath.Join(t.wd, "input.txt"))
	if err != nil {
		log.WithFields(log.Fields{
			"file":  inFile,
			"error": err.Error(),
		}).Error("Unable to create input target list file")
		return &nmapTasker{}, err
	}

	inFile.WriteString(t.job.Parameters["targets"])

	// Append that file to the arguments
	args = append(args, "-iL", filepath.Join(t.wd, "input.txt"))

	log.WithField("arguments", args).Debug("Arguments complete")

	t.job.PerformanceTitle = "Packets / sec"
	t.job.OutputTitles = []string{"IP Address", "Hostname", "Protocol", "Port", "Service"}
	t.job.TotalHashes, err = calcTotalTargets(t.job.Parameters["targets"])
	if err != nil {
		return &nmapTasker{}, err
	}

	t.start = append(t.start, args...)
	t.resume = append(t.resume, "--resume", filepath.Join(t.wd, "output.grep"))

	return &t, nil
}

func (v *nmapTasker) Status() common.Job {
	log.WithField("task", v.job.UUID).Debug("Gathering task details")
	v.mux.Lock()
	defer v.mux.Unlock()

	status := string(v.stdout.Bytes())

	//Time to gather the progress
	hostsDone := regHostsCompleted.FindStringSubmatch(status)
	if len(hostsDone) == 2 {
		prog, err := strconv.ParseInt(hostsDone[1], 10, 64)
		if err == nil {
			v.job.CrackedHashes = prog
			log.WithField("hostsfinished", v.job.CrackedHashes).Debug("Nmap job progress updated.")
		} else {
			log.WithField("error", err.Error()).Error("There was a problem converting progress to a number.")
		}
	} else {
		log.WithField("hostsdone", hostsDone).Debug("Did not match hosts done.")
	}

	timing := regTimeEstimate.FindStringSubmatch(status)
	if len(timing) == 3 {
		percent, err := strconv.ParseFloat(timing[1], 64)
		if err == nil {
			v.job.Progress = percent
			log.WithField("percent", percent).Debug("Nmap updated job progress")
		} else {
			log.Error("Unable to parse nmap percentage complete: " + err.Error())
		}
		v.job.ETC = timing[1] // Estimated time of completion
	} else {
		log.WithField("timing", timing).Debug("Did not match time estimate")
	}

	performance := regPerformance.FindStringSubmatch(status)
	if len(performance) == 2 {
		timestamp := fmt.Sprintf("%d", time.Now().Unix())
		log.WithFields(log.Fields{
			"timestamp": timestamp,
			"perfdata":  performance[1],
		}).Debug("Updating performance data.")
		v.job.PerformanceData[timestamp] = performance[1]
	} else {
		log.WithField("performance", performance).Debug("Did not match performance data")
	}

	v.job.Error = v.stderr.String()
	log.WithFields(log.Fields{
		"task":   v.job.UUID,
		"status": v.job.Status,
		"stdout": v.stdout,
		"stderr": v.stderr,
	}).Info("Ongoing nmap task status")

	v.stdout.Reset()

	return v.job
}

func (v *nmapTasker) Run() error {
	v.mux.Lock()
	defer v.mux.Unlock()
	// Check that we have not already finished this job
	done := v.job.Status == common.STATUS_DONE || v.job.Status == common.STATUS_QUIT || v.job.Status == common.STATUS_FAILED
	if done {
		log.WithField("Status", v.job.Status).Debug("Unable to start nmap job, it has already finished.")
		return errors.New("New nmap job already finished.")
	}

	// Check if this job is running
	if v.job.Status == common.STATUS_RUNNING {
		// Job already running so return no errors
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
	}).Debug("Setup exec.command for nmap tool")

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
	go v.onCmdComplete()

	return nil
}

func (v *nmapTasker) onCmdComplete() {
	// Listen on commmand wait and then send signal when finished
	// This will be read on the Status() function
	v.cmd.Wait()

	v.mux.Lock()
	defer v.mux.Unlock()

	v.job.Status = common.STATUS_DONE
	v.job.Progress = 100.00
	v.job.CrackedHashes = v.job.TotalHashes
	v.waitChan <- struct{}{}

	// Get the output results
	data, err := parseNmapXML(filepath.Join(v.wd, "output.xml"))
	if err != nil {
		log.WithField("error", err.Error()).Error("Unable to parse NMap output data.")
	} else {
		v.job.OutputData = nmapToCSV(data)
	}
}

// Pause the hashcat run
func (v *nmapTasker) Pause() error {
	log.WithField("task", v.job.UUID).Debug("Attempting to pause nmap task")

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
	log.WithField("task", v.job.UUID).Debug("Attempting to quit nmap task")

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

func calcTotalTargets(input string) (int64, error) {
	lines := strings.Split(input, "\n")
	total := 0

	for _, line := range lines {
		if strings.Contains(line, "-") {
			cnt, err := getRangeTargetCount(line)
			if err != nil {
				return -1, err
			}
			total += cnt
		} else if strings.Contains(line, "/") {
			cnt, err := getCIDRTargetCount(line)
			if err != nil {
				return -1, err
			}
			total += cnt
		} else {
			total++
		}
	}

	return int64(total), nil
}

func getCIDRTargetCount(cidr string) (int, error) {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return -1, err
	}

	ones, bits := ipnet.Mask.Size()
	zeros := uint(bits - ones)

	return 1 << zeros, nil
}

func getRangeTargetCount(ip string) (int, error) {
	octets := strings.Split(ip, ".")
	if len(octets) != 4 {
		return -1, errors.New("Address did not parse out to 4 octets")
	}

	count := 1
	for curoctet := 0; curoctet < 4; curoctet++ {
		rng := strings.Split(octets[curoctet], "-")
		if len(rng) == 2 {
			one, _ := strconv.Atoi(rng[0])
			two, _ := strconv.Atoi(rng[1])
			if one > two {
				count = count * (one - two + 1)
			} else {
				count = count * (two - one + 1)
			}
		}
	}

	return count, nil
}
