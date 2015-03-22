package queue

import (
	"github.com/jmmcatee/cracklord/common"
	"net/rpc"
)

type ResourcePool map[string]Resource

type Resource struct {
	Client   *rpc.Client
	Name     string
	Address  string
	RPCCall  common.RPCCall
	Hardware map[string]bool
	Tools    map[string]common.Tool
	Paused   bool
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
