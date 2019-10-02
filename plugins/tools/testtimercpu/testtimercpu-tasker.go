package testtimercpu

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/jmmcatee/cracklord/common"
)

type testPipe struct {
	R *io.PipeReader
	W *io.PipeWriter
}

type testTimerCPUTasker struct {
	job       common.Job
	success   bool
	stop      chan bool
	stderrBuf *bytes.Buffer
	stdoutBuf *bytes.Buffer
	stderr    testPipe
	stdout    testPipe
	stdin     testPipe
}

func newTestTimerTask(j common.Job) (common.Tasker, error) {
	log.Debug("Starting up a new example task plugin.")
	t := testTimerCPUTasker{}

	t.job = j
	t.job.CrackedHashes = 0
	t.job.PerformanceTitle = "Time data"

	var err error
	t.job.TotalHashes, err = strconv.ParseInt(j.Parameters["seconds"], 10, 0)
	if err != nil {
		return &t, errors.New("Unable to parse seconds.")
	}

	if j.Parameters["result"] == "Success" {
		t.success = true
	} else {
		t.success = false
	}

	t.stderr.R, t.stderr.W = io.Pipe()
	t.stdout.R, t.stdout.W = io.Pipe()
	t.stdin.R, t.stdin.W = io.Pipe()

	return &t, nil
}

func (t *testTimerCPUTasker) Status() common.Job {
	timestamp := fmt.Sprintf("%d", time.Now().Unix())

	t.job.PerformanceData[timestamp] = fmt.Sprintf("%d", t.job.CrackedHashes)
	t.job.Progress = float64(t.job.CrackedHashes) / float64(t.job.TotalHashes) * 100.0
	t.job.ETC = fmt.Sprintf("%d seconds", t.job.TotalHashes-t.job.CrackedHashes)

	log.WithFields(log.Fields{
		"cur": t.job.CrackedHashes,
		"max": t.job.TotalHashes,
	}).Debug("Test timer status.")

	return t.job
}

func (t *testTimerCPUTasker) Run() error {
	t.stop = make(chan bool)
	t.job.Status = common.STATUS_RUNNING

	go func() {
		for ; t.job.CrackedHashes < t.job.TotalHashes; t.job.CrackedHashes++ {
			select {
			case <-t.stop:
				return
			case <-time.After(time.Second):
			}
		}
		if t.success {
			t.job.Status = common.STATUS_DONE
		} else {
			t.job.Status = common.STATUS_FAILED
		}
		log.WithField("status", t.job.Status).Info("Timer ended")
	}()
	return nil
}

func (t *testTimerCPUTasker) Pause() error {
	log.Debug("Pausing test timer job.")
	t.stop <- true
	t.job.Status = common.STATUS_PAUSED
	return nil
}

func (t *testTimerCPUTasker) Quit() common.Job {
	log.Debug("Quitting test timer job.")
	t.stop <- true
	t.job.Status = common.STATUS_QUIT
	return t.job
}

func (t *testTimerCPUTasker) Done() {

}

func (t *testTimerCPUTasker) IOE() (io.Writer, io.Reader, io.Reader) {
	return t.stdin.W, t.stderr.R, t.stdout.R
}
