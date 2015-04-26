package common

import (
	"encoding/json"
)

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
	Form   json.RawMessage `json:"form"`
	Schema json.RawMessage `json:"schema"`
}
