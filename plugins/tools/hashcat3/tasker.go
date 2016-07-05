package hashcat3

import (
	"bytes"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/jmmcatee/cracklord/common"
	"io"
	"os/exec"
	"sync"
	"time"
)

// Tasker is the structure that implements the Tasker inteface
type Tasker struct {
	job        common.Job
	wd         string
	exec       exec.Cmd
	start      []string
	resume     []string
	stderr     *bytes.Buffer
	stderrCp   bool
	stdout     *bytes.Buffer
	stdoutCp   bool
	stderrPipe io.ReadCloser
	stdoutPipe io.ReadCloser
	stdinPipe  io.WriteCloser

	waitChan chan struct{}

	mux sync.Mutex
}

// Status returns the common.Job option of the Tasker
func (t *Tasker) Status() common.Job {
	log.WithField("task", t.job.UUID).Debug("Status call for hashcat3 Tasker")

	t.mux.Lock()
	defer t.mux.Unlock()

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

	if !t.stderrCp {
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

	status := ParseMachineOutput(t.stdout.String())

	if t.job.PerformanceTitle == "" {
		t.job.PerformanceTitle = "MH/s"
	}
	t.job.Progress = status.Progress
	t.job.ETC = status.EstimateTime

	var totalSpeed float64
	for i := range status.Speed {
		totalSpeed += status.Speed[i]
	}
	t.job.PerformanceData[fmt.Sprintf("%d", time.Now().Unix())] = fmt.Sprintf("%f", totalSpeed)

	t.job.CrackedHashes = status.RecoveredHashes
	t.job.TotalHashes = status.TotalHashes
	t.job.Error = t.stderr.String()

	// Parse hash output

	return t.job
}
