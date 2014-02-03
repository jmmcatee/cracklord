package resource

import (
	"bytes"
	"cl/common"
	"errors"
	"io"
)

type simpleFailerTooler struct {
	uuid string
}

func (s *simpleFailerTooler) Name() string {
	return "Simple Failure Tool"
}

func (s *simpleFailerTooler) Type() string {
	return "Failure"
}

func (s *simpleFailerTooler) Version() string {
	return "1.0.0"
}

func (s *simpleFailerTooler) SetUUID(uuid string) {
	s.uuid = uuid
}

func (s *simpleFailerTooler) UUID() string {
	return s.uuid
}

func (s *simpleFailerTooler) Parameters() string {
	return "{timer:num}"
}

func (s *simpleFailerTooler) Requirements() string {
	return common.RES_CPU
}

func (s *simpleFailerTooler) NewTask(j common.Job) common.Tasker {
	return &simpleFailureTask{
		j: j,
	}
}

type simpleFailureTask struct {
	failFunc string
	j        common.Job
}

func (t *simpleFailureTask) Status() common.Job {
	return t.j
}

func (t *simpleFailureTask) Run() error {
	var ok bool
	t.failFunc, ok = t.j.Parameters["failFunc"]
	if !ok {
		return errors.New("failFunc was not provided.")
	}

	if t.j.Status == common.STATUS_FAILED || t.j.Status == common.STATUS_DONE {
		return errors.New("Cannot start task as its status is " + t.j.Status)
	}

	if t.j.Status == common.STATUS_RUNNING {
		return nil
	}

	if t.j.Status == common.STATUS_PAUSED {
		if t.failFunc == "RunAfterPause" {
			return errors.New("Expected on Run")
		}

		t.j.Status = common.STATUS_RUNNING
	}

	t.j.Status = common.STATUS_RUNNING

	if t.failFunc == "Run" {
		return errors.New("Expected on Run")
	}

	return nil
}

func (t *simpleFailureTask) Pause() error {
	var ok bool
	t.failFunc, ok = t.j.Parameters["failFunc"]
	if !ok {
		return errors.New("failFunc was not provided.")
	}

	if t.failFunc == "Pause" {
		return errors.New("Expected on Pause")
	}

	if t.j.Status == common.STATUS_RUNNING {
		t.j.Status = common.STATUS_PAUSED

		return nil
	} else if t.j.Status == common.STATUS_PAUSED {
		return nil
	} else {
		return errors.New("Cannot pause task as status is " + t.j.Status)
	}
}

func (t *simpleFailureTask) Quit() common.Job {
	if t.j.Status != common.STATUS_DONE || t.j.Status != common.STATUS_FAILED || t.j.Status != common.STATUS_QUIT {
		t.failFunc, _ = t.j.Parameters["failFunc"]

		if t.failFunc == "Quit" {
			t.j.Error = "Stopped by user"
		}

		t.j.Status = common.STATUS_QUIT
		return t.j
	}

	return t.j
}

func (t *simpleFailureTask) IOE() (io.Writer, io.Reader, io.Reader) {
	a := bytes.NewBuffer([]byte("Stdin not implemented"))
	b := bytes.NewBuffer([]byte("Stdout not implemneted"))
	c := bytes.NewBuffer([]byte("Stderr not implemented"))

	return a, b, c
}
