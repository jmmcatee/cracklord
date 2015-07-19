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
