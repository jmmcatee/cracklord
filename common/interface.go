package common

import (
	"code.google.com/p/go-uuid/uuid"
	"encoding/json"
	"io"
	"time"
)

const (
	STATUS_CREATED = "Created"
	STATUS_RUNNING = "Running"
	STATUS_PAUSED  = "Paused"
	STATUS_DONE    = "Done"
	STATUS_FAILED  = "Failed"
	STATUS_QUIT    = "Quit"

	RES_CPU = "cpu"
	RES_GPU = "gpu"
)

type Tasker interface {
	Status() Job
	Run() error
	Pause() error
	Quit() Job
	IOE() (io.Writer, io.Reader, io.Reader)
}

type Tooler interface {
	Name() string
	Type() string
	Version() string
	UUID() string
	SetUUID(string)
	Parameters() string
	Requirements() string
	NewTask(Job) (Tasker, error)
}

type Job struct {
	UUID             string
	ToolUUID         string
	Name             string
	Status           string
	Error            string
	StartTime        time.Time
	Owner            string
	ResAssigned      string
	CrackedHashes    int64
	TotalHashes      int64
	Percentage       int
	Parameters       map[string]string
	Performance      map[string]string
	PerformanceTitle string
	Output           map[string]string
}

func NewJob(tooluuid string, name string, owner string, params map[string]string) Job {
	return Job{
		UUID:       uuid.New(),
		ToolUUID:   tooluuid,
		Name:       name,
		Status:     STATUS_CREATED,
		Owner:      owner,
		Parameters: params,
		Output:     make(map[string]string),
	}
}

type Tool struct {
	Name         string
	Type         string
	Version      string
	UUID         string
	Parameters   string
	Requirements string
}

// Compare two Tools to see if they are the same
func CompareTools(t1, t2 Tool) bool {
	if t1.Name != t2.Name {
		return false
	}

	if t1.Type != t2.Type {
		return false
	}

	if t1.Version != t2.Version {
		return false
	}

	if t1.Parameters != t2.Parameters {
		return false
	}

	if t1.Requirements != t2.Requirements {
		return false
	}

	return true
}

type ToolJSONForm struct {
	Form   json.RawMessage `json:form`
	Schema json.RawMessage `json.schema`
}

type RPCCall struct {
	Auth string
	Job  Job
}
