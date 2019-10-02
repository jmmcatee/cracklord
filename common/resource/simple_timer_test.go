package resource

import (
	"bytes"
	"errors"
	"io"
	"strconv"
	"time"

	"github.com/jmmcatee/cracklord/common"
)

type SimpleTimerTooler struct {
	uuid string
}

func (s *SimpleTimerTooler) Name() string {
	return "Simple Timer Tool"
}

func (s *SimpleTimerTooler) Type() string {
	return "Timer"
}

func (s *SimpleTimerTooler) Version() string {
	return "1.0.0"
}

func (s *SimpleTimerTooler) SetUUID(uuid string) {
	s.uuid = uuid
}

func (s *SimpleTimerTooler) UUID() string {
	return s.uuid
}

func (s *SimpleTimerTooler) Parameters() string {
	return "{timer:num}"
}

func (s *SimpleTimerTooler) Requirements() string {
	return common.RES_CPU
}

func (s *SimpleTimerTooler) NewTask(j common.Job) (common.Tasker, error) {
	if _, ok := j.Parameters["timer"]; !ok {
		return nil, errors.New("timer parameter not given!")
	}

	i, err := strconv.Atoi(j.Parameters["timer"])
	if err != nil {
		return nil, err
	}

	v := time.Duration(i) * time.Second

	return &SimpleTimer{
		j: j,
		d: v,
	}, nil
}

type SimpleTimer struct {
	s    time.Time
	d    time.Duration
	r    time.Duration
	t    *time.Timer
	kill chan bool
	j    common.Job
}

func (t *SimpleTimer) Done() {

}

func (t *SimpleTimer) Status() common.Job {
	if t.j.Status == common.STATUS_PAUSED {
		tmpData := make([]string, 1)
		tmpData[0] = strconv.Itoa(int(t.r))

		t.j.OutputData[0] = tmpData
	}

	if t.j.Status == common.STATUS_RUNNING {
		if t.r == 0 {
			tmpData := make([]string, 1)
			tmpData[0] = strconv.Itoa(int(t.d - time.Since(t.s)))
			t.j.OutputData[0] = tmpData
		} else {
			tmpData := make([]string, 1)
			tmpData[0] = strconv.Itoa(int(t.d - time.Since(t.s) - t.r))
			t.j.OutputData[0] = tmpData
		}

	}

	return t.j
}

func (t *SimpleTimer) Run() error {
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

func (t *SimpleTimer) Pause() error {
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

func (t *SimpleTimer) Quit() common.Job {
	t.t.Stop()
	t.kill <- true

	if t.j.Status != common.STATUS_DONE && t.j.Status != common.STATUS_FAILED && t.j.Status != common.STATUS_QUIT {
		t.j.Error = "Stopped by user"
		t.j.Status = common.STATUS_QUIT
		return t.j
	}

	return t.j
}

func (t *SimpleTimer) IOE() (io.Writer, io.Reader, io.Reader) {
	a := bytes.NewBuffer([]byte("Stdin not implemented"))
	b := bytes.NewBuffer([]byte("Stdout not implemneted"))
	c := bytes.NewBuffer([]byte("Stderr not implemented"))

	return a, b, c
}
