package common

import (
	"io"
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
