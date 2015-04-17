package exampleplugin

import (
	log "github.com/Sirupsen/logrus"
	"github.com/jmmcatee/cracklord/common"
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
}

/*
	Gather a status of job information for this running task.  This will be used
	by the queue server every so often (5-30 seconds typically) to keep job
	information
*/
func (v *exampleTasker) Status() common.Job {
}

/*
	Start or restart a job running under this tool.
*/
func (v *exampleTasker) Run() error {
}

/*
	Pause the job if possible, save the state, etc.
*/
func (v *exampleTasker) Pause() error {
}

/*
	Stop the job running under this tool, don't forget to cleanup and file system
	resources, etc.
*/
func (v *exampleTasker) Quit() common.Job {
}

/*
	Get the input, output, and error streams
*/
func (v *hascatTasker) IOE() (io.Writer, io.Reader, io.Reader) {
	return v.stdinPipe, v.stdoutPipe, v.stderrPipe
}
