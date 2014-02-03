package queue

import (
	"github.com/jmmcatee/cracklord/common"
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
