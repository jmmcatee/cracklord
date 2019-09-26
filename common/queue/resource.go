package queue

import (
	"net/rpc"

	"github.com/jmmcatee/cracklord/common"
)

type ResourcePool map[string]Resource

type Resource struct {
	Client   *rpc.Client
	Name     string
	Address  string
	Hardware map[string]bool
	Tools    map[string]common.Tool
	Status   string // Can be running, paused, quit
}

func NewResourcePool() ResourcePool {
	return make(map[string]Resource)
}

func NewResource() Resource {
	return Resource{
		Hardware: make(map[string]bool),
		Tools:    make(map[string]common.Tool),
	}
}
