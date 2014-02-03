package resource

import (
	"bytes"
	"cl/common"
	"errors"
	"io"
	"strconv"
	"time"
)

type simpleTimerTooler struct {
	uuid string
}

func (s *simpleTimerTooler) Name() string {
	return "Simple Timer Tool"
}

func (s *simpleTimerTooler) Type() string {
	return "Timer"
}

func (s *simpleTimerTooler) Version() string {
	return "1.0.0"
}

func (s *simpleTimerTooler) SetUUID(uuid string) {
	s.uuid = uuid
}

func (s *simpleTimerTooler) UUID() string {
	return s.uuid
}

func (s *simpleTimerTooler) Parameters() string {
	return "{timer:num}"
}

func (s *simpleTimerTooler) Requirements() string {
	return common.RES_CPU
}

func (s *simpleTimerTooler) NewTask(j common.Job) common.Tasker {
	return &simpleTimer{
		j: j,
	}
}

type simpleTimer struct {
	s    time.Time
	d    time.Duration
	r    time.Duration
	t    *time.Timer
	kill chan bool
	j    common.Job
}

func (t *simpleTimer) Status() common.Job {
	if t.j.Status == common.STATUS_PAUSED {
		t.j.Output["TimeLeft"] = strconv.Itoa(int(t.r))
	}

	if t.j.Status == common.STATUS_RUNNING {
		if t.r == 0 {
			t.j.Output["TimeLeft"] = strconv.Itoa(int(t.d - time.Since(t.s)))
		} else {
			t.j.Output["TimeLeft"] = strconv.Itoa(int(t.d - time.Since(t.s) - t.r))
		}

	}

	return t.j
}

func (t *simpleTimer) Run() error {
	if t.j.Status == common.STATUS_FAILED || t.j.Status == common.STATUS_DONE {
		return errors.New("Cannot start task as its status is " + t.j.Status)
	}

	if t.j.Status == common.STATUS_RUNNING {
		return nil
	}

	if t.j.Status == common.STATUS_PAUSED {
		// Subtrack the time already run (r) from the set timer duration (d)
		t.t = time.NewTimer(t.d - t.r)
		t.j.Status = common.STATUS_RUNNING
	}

	i, err := strconv.Atoi(t.j.Parameters["timer"])
	if err != nil {
		return err
	}

	t.d = time.Duration(i) * time.Second
	t.t = time.NewTimer(t.d)
	t.s = time.Now()
	t.j.StartTime = t.s
	t.j.Status = common.STATUS_RUNNING
	t.kill = make(chan bool)

	go func() {
		select {
		case <-t.t.C:
			t.j.Status = common.STATUS_DONE
		case <-t.kill:
			return
		}
	}()

	return nil
}

func (t *simpleTimer) Pause() error {
	if t.j.Status == common.STATUS_RUNNING {
		t.t.Stop()

		t.r = t.r + t.s.Sub(time.Now())

		t.j.Status = common.STATUS_PAUSED

		return nil
	} else if t.j.Status == common.STATUS_PAUSED {
		return nil
	} else {
		return errors.New("Cannot pause task as status is " + t.j.Status)
	}

}

func (t *simpleTimer) Quit() common.Job {
	t.t.Stop()
	t.kill <- true

	if t.j.Status != common.STATUS_DONE || t.j.Status != common.STATUS_FAILED || t.j.Status != common.STATUS_QUIT {
		t.j.Error = "Stopped by user"
		t.j.Status = common.STATUS_QUIT
		return t.j
	}

	return t.j
}

func (t *simpleTimer) IOE() (io.Writer, io.Reader, io.Reader) {
	a := bytes.NewBuffer([]byte("Stdin not implemented"))
	b := bytes.NewBuffer([]byte("Stdout not implemneted"))
	c := bytes.NewBuffer([]byte("Stderr not implemented"))

	return a, b, c
}
