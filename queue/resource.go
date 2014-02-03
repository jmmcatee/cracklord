package queue

import (
	"cl/common"
	"net/rpc"
)

type ResourcePool map[string]Resource

type Resource struct {
	Client   *rpc.Client
	RPCCall  common.RPCCall
	Hardware map[string]bool
	Tools    map[string]common.Tool
	Paused   bool
}
