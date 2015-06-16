package common

import ()

const (
	STATUS_CREATED = "created"
	STATUS_RUNNING = "running"
	STATUS_PAUSED  = "paused"
	STATUS_DONE    = "done"
	STATUS_FAILED  = "failed"
	STATUS_QUIT    = "quit"

	RES_CPU = "cpu"
	RES_GPU = "gpu"
)

type RPCCall struct {
	Auth string
	Job  Job
}
