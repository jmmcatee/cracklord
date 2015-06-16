package common

import ()

type Resource struct {
	UUID     string
	Name     string
	Hardware map[string]bool
	Tools    map[string]Tool
	Status   string
	Address  string
}
