package common

import ()

type Resource struct {
	UUID     string
	Name     string
	Hardware map[string]bool
	Tools    map[string]Tool
	Paused   bool
	Address  string
}

type Queue interface {
	Tools() map[string]Tool
	Types() []string

	PauseQueue() []error
	ResumeQueue()
	Quit() []Job
	StackReorder(uuids []string) []error

	AddResource(addr string, auth string) error
	GetResources() []Resource
	PauseResource(resUUID string) error
	ResumeResource(resUUID string) error
	RemoveResource(resUUID string) error

	AllJobs() []Job
	JobInfo(jobUUID string) Job
	AddJob(j Job) error
	PauseJob(jobuuid string) error
	QuitJob(jobuuid string) error
}
