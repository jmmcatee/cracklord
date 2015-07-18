package exampleplugin

import (
	"bytes"
	log "github.com/Sirupsen/logrus"
	"github.com/jmmcatee/cracklord/common"
	"io"
	"os/exec"
	"sync"
)

/*
	Function that runs on the setup of this package once and only once, useful
	for one time setup of things
*/
func init() {
}

/*
	Struct to hold our interface functions and some basic data that you would
	probably want
*/
type exampleTasker struct {
	job        common.Job
	cmd        exec.Cmd
	stderr     *bytes.Buffer
	stdout     *bytes.Buffer
	stderrPipe io.ReadCloser
	stdoutPipe io.ReadCloser
	stdinPipe  io.WriteCloser

	mux  sync.Mutex
	done bool
}

/*
	Create a new tasker for this tool.  As noted in the "tooler", the "tasker"
	is used to actually run individual jobs as tasks on the individual resource
	server
*/
func newExampleTask(j common.Job) (common.Tasker, error) {
	//You can also log.Fatal (which will kill the resourceserver, so avoid), log.Error, log.Warn, and log.Info
	log.Debug("Starting up a new example task plugin.")

	e := exampleTasker{}

	return &e, nil
}

/*
	Gather a status of job information for this running task.  This will be used
	by the queue server every so often (5-30 seconds typically) to keep job
	information
*/
func (v *exampleTasker) Status() common.Job {
	return v.job
}

/*
	Start or restart a job running under this tool.
*/
func (v *exampleTasker) Run() error {
	return nil
}

/*
	Pause the job if possible, save the state, etc.
*/
func (v *exampleTasker) Pause() error {
	return nil
}

/*
	Stop the job running under this tool, don't forget to cleanup and file system
	resources, etc.
*/
func (v *exampleTasker) Quit() common.Job {
	return v.job
}

/*
	Get the input, output, and error streams
*/
func (v *exampleTasker) IOE() (io.Writer, io.Reader, io.Reader) {
	return v.stdinPipe, v.stdoutPipe, v.stderrPipe
}
