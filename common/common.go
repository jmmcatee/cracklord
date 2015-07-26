package common

import (
	"encoding/json"
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
	RES_NET = "net"
)

type RPCCall struct {
	Job Job
}

type JSONSchemaForm struct {
	Form   json.RawMessage `json:"form"`
	Schema json.RawMessage `json:"schema"`
}

// Function to determine if a status shows something is completed
func IsDone(status string) bool {
	switch status {
	case STATUS_DONE, STATUS_FAILED, STATUS_QUIT:
		return true
	default:
		return false
	}

}

// Function to determine if a status shwos something is running
func IsRunning(status string) bool {
	switch status {
	case STATUS_RUNNING:
		return true
	default:
		return false
	}
}

// Function to determine if a status shows something has failed
func IsFailed(status string) bool {
	switch status {
	case STATUS_FAILED:
		return true
	default:
		return false
	}
}

// Function to determine if a status shows something is newly created
func IsNew(status string) bool {
	switch status {
	case STATUS_CREATED:
		return true
	default:
		return false
	}
}

func StripQuotes(str string) string {
	if str == "" {
		return str
	}

	tmp := strings.TrimSpace(str)
	l := len(tmp)

	if l < 1 {
		return tmp
	}

	first := tmp[:1]
	last := tmp[l-1:]

	if first == "\"" && last == "\"" {
		return tmp[1 : l-1]
	}

	return tmp
}
