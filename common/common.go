package common

import (
	"strings"
)

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
	Job Job
}

func StripQuotes(str string) string {
	tmp   := strings.TrimSpace(str)
	l     := len(tmp)
	first := tmp[:1]
	last  := tmp[l-1:]

	if first == "\"" && last == "\"" {
		return tmp[1:l-1]
	}
	return tmp
}