package api

import (
	"github.com/jmmcatee/cracklord/common"
)

/*
 * Structures used for JSON calls to /login
 */
type APILoginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type APILoginResp struct {
	Token string `json:"token"`
}

/*
 * Structure used for JSON calls to /logout
 */
type APILogoutReq struct {
	Token string `json:"token"`
}

/*
 * Structures used for JSON calls to /crack/types
 */
type APICrackTypesReq struct {
	Token string `json:"token"`
}

type APICrackTypesResp struct {
	Types []string `json:"types"`
}

/*
 * Structures used for JSON calls to /crack/tools
 */
type APICrackToolsReq struct {
	Token string `json:"token"`
}

type APICrackToolsResp struct {
	Tools map[string]common.Tool `json:"tools"`
}

/*
 * Structures used for JSON calls to /crack/form
 */
type APICrackFormReq struct {
	Token string `json:"token"`
}

// This needs to be changed
type APICrackFormResp struct {
	Form string
}

/*
	Structures used for the JSON calls to /job/new
*/
type APIJobNewReq struct {
	Token  string            `json:"token"`
	Tool   string            `json:"tool"`
	Name   string            `json:"name"`
	Params map[string]string `json:"params"`
}

type APIJobNewResp struct {
	JobID string `json:"jobid"`
}

/*
	Stucture used for the JSON calls to /shutdown
*/
type APIShutdownReq struct {
	Token string `json:"token"`
}

/*
	Structures used for the JSON calls to /queue/list
*/
type APIQueueListReq struct {
	Token string `json:"token"`
}

type APIQueueListResp struct {
	Queue []common.Job
}

/*
	Structures used for the JSON calls to /queue/reorder
*/
type APIQueueReorderReq struct {
	Token    string   `json:"token"`
	JobOrder []string `json:"joborder"`
}

/*
	Structures used for the JSON calls to /job/info
*/
type APIJobInfoReq struct {
	Token string `json:"token"`
	JobID string `json:"jobid"`
}

type APIJobInfoResp struct {
	Job common.Job `json:"job"`
}

/*
	Structures used for the JSON calls to /job/pause
*/
type APIJobPauseReq struct {
	Token string `json:"token"`
	JobID string `json:"jobid"`
}

/*
	Structures used for the JSON calls to /job/quit
*/
type APIJobQuitReq struct {
	Token string `json:"token"`
	JobID string `json:"jobid"`
}

/*
	Structures used for the JSON calls to /resource/list
*/
type APIResourceListReq struct {
	Token string `json:"token"`
}

type APIResourceListResp struct {
	Resources []common.Resource
}

/*
	Structures used for the JSON calls to /resource/pause
*/
type APIResourcePauseReq struct {
	Token      string `json:"token"`
	ResourceID string `json:"resourceid"`
}

/*
	Structures used for the JSON calls to /resource/quit
*/
type APIResourceQuitReq struct {
	Token      string `json:"token"`
	ResourceID string `json:"resourceid"`
}

/*
	Structures used for the JSON calls to /resource/add
*/
type APIResourceAddReq struct {
	Token     string `json:"token"`
	IPAddress string `json:"ipaddress"`
	AuthToken string `json:"authtoken"`
}
