package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"github.com/jmmcatee/cracklord/common"
	"net/http"
)

func APILogin(req *http.Request, auth common.Authenticator, ts common.TokenStore) (int, string) {
	// Grab the username and password submitted
	login := APILoginReq{}
	err := json.NewDecoder(req.Body).Decode(login)
	if err != nil {
		return 500, ErrorLogin
	}

	// Check login information
	user, err := auth.Login(login.Username, login.Password)
	if err != nil {
		return 500, ErrorLogin
	}

	seed := make([]byte, 256)
	token := sha256.New()

	rand.Read(seed)

	apitoken := APILoginResp{}

	apitoken.Token = base64.StdEncoding.EncodeToString(token.Sum(seed))

	resp, err := json.Marshal(apitoken)

	if err != nil {
		return 500, ErrorLogin
	}

	// Add token to the token store
	ts.AddToken(apitoken.Token, user)

	return 200, string(resp)
}

func APILogout(req *http.Request, ts common.TokenStore) (int, string) {
	// Grab the token to logout
	logout := APILogoutReq{}

	// Decode into it
	err := json.NewDecoder(req.Body).Decode(logout)
	if err != nil {
		return 500, ErrorLogin
	}

	// Logout the provided token
	ts.RemoveToken(logout.Token)

	return 200, Success
}

func APICrackTypes(req *http.Request, ts common.TokenStore, queue common.Queue) (int, string) {
	// Decode the json request
	typeReq := APICrackTypesReq{}
	err := json.NewDecoder(req.Body).Decode(typeReq)
	if err != nil {
		return 500, ErrorMalformedRequest
	}

	// Check that a valid token was provided
	if !ts.CheckToken(typeReq.Token) {
		return 500, ErrorBadToken
	}

	// Given a valid token return the various crack types from the Queue
	types := APICrackTypesResp{}
	types.Types = queue.Types()

	// Encode the response
	resp, err := json.Marshal(types)
	if err != nil {
		return 500, ErrorInternal
	}

	return 200, string(resp)
}

func APICrackTools(req *http.Request, ts common.TokenStore, queue common.Queue) (int, string) {
	// Decode the json request
	toolsReq := APICrackToolsReq{}
	err := json.NewDecoder(req.Body).Decode(toolsReq)
	if err != nil {
		return 500, ErrorMalformedRequest
	}

	// Check that a valid token was provided
	if !ts.CheckToken(toolsReq.Token) {
		return 500, ErrorBadToken
	}

	// Given a valid token return all the tools
	tools := APICrackToolsResp{}
	tools.Tools = queue.Tools()

	// Encode the response
	resp, err := json.Marshal(tools)
	if err != nil {
		return 500, ErrorInternal
	}

	return 200, string(resp)
}

func APICrackForm(req *http.Request, ts common.TokenStore, queue common.Queue) (int, string) {
	// Decode the json request
	jsonReq := APICrackFormReq{}
	err := json.NewDecoder(req.Body).Decode(jsonReq)
	if err != nil {
		return 500, ErrorMalformedRequest
	}

	// Check the token provided
	if !ts.CheckToken(jsonReq.Token) {
		return 500, ErrorBadToken
	}

	// Take the ToolID and get the form JSON object
	for toolUUID, tool := range queue.Tools() {
		if toolUUID == jsonReq.ToolID {
			resp := APICrackFormResp{Form: tool.Parameters}

			jsonResp, err := json.Marshal(resp)
			if err != nil {
				return 500, ErrorInternal
			}

			return 200, string(jsonResp)
		}
	}

	return 500, ErrorInternal
}

func APIJobNew(req *http.Request, ts common.TokenStore, queue common.Queue) (int, string) {
	// Decode the json request
	njReq := APIJobNewReq{}

	err := json.NewDecoder(req.Body).Decode(njReq)
	if err != nil {
		return 500, ErrorMalformedRequest
	}

	// Check that a valid token was provided
	if !ts.CheckToken(njReq.Token) {
		return 500, ErrorBadToken
	}

	// We need user information for this request so get it from the TokenStore
	user, err := ts.GetUser(njReq.Token)
	if err != nil {
		return 500, ErrorBadToken
	}

	// This request is not available to Read-Only users so check group membership
	var check bool
	for _, group := range user.Groups {
		check = check || group == common.StandardUser || group == common.Administrator
	}

	if !check {
		return 500, ErrorPermissionDenied
	}

	// Now we have the request decoded so build a common.Job
	job := common.NewJob(njReq.ToolID, njReq.Name, user.Username, njReq.Params)

	// Add the job to the queue
	err = queue.AddJob(job)
	if err != nil {
		return 500, ErrorJobAdd
	}

	// Everything worked great so build a response
	tempResp := APIJobNewResp{}
	tempResp.JobID = job.UUID

	resp, err := json.Marshal(tempResp)
	if err != nil {
		return 500, ErrorInternal
	}

	return 200, string(resp)
}

// TODO: Add in the ability to alert the higher level process to quit and shutdown
func APIShutdown(req *http.Request, ts common.TokenStore, queue common.Queue) (int, string) {
	// Decode the json request
	jsonReq := APIShutdownReq{}
	err := json.NewDecoder(req.Body).Decode(jsonReq)
	if err != nil {
		return 500, ErrorMalformedRequest
	}

	// Check the token
	if !ts.CheckToken(jsonReq.Token) {
		return 500, ErrorBadToken
	}

	// We need user information for this request so get it from the TokenStore
	user, err := ts.GetUser(jsonReq.Token)
	if err != nil {
		return 500, ErrorBadToken
	}

	// This request is only available to adminsitrators
	var check bool
	for _, group := range user.Groups {
		check = check || group == common.Administrator
	}
	if !check {
		return 500, ErrorPermissionDenied
	}

	// Quit the Queue
	queue.Quit()

	return 200, Success
}

func APIQueueList(req *http.Request, ts common.TokenStore, queue common.Queue) (int, string) {
	// Decode the json request
	jsonReq := APIQueueListReq{}
	err := json.NewDecoder(req.Body).Decode(jsonReq)
	if err != nil {
		return 500, ErrorMalformedRequest
	}

	// Check the token
	if !ts.CheckToken(jsonReq.Token) {
		return 500, ErrorBadToken
	}

	// Get the list of jobs
	jsonResp := APIQueueListResp{}
	jsonResp.Queue = queue.AllJobs()

	resp, err := json.Marshal(jsonResp)
	if err != nil {
		return 500, ErrorInternal
	}

	return 200, string(resp)
}

func APIQueueReorder(req *http.Request, ts common.TokenStore, queue common.Queue) (int, string) {
	// Decode the json request
	jsonReq := APIQueueReorderReq{}
	err := json.NewDecoder(req.Body).Decode(jsonReq)
	if err != nil {
		return 500, ErrorMalformedRequest
	}

	// Check the token
	if !ts.CheckToken(jsonReq.Token) {
		return 500, ErrorBadToken
	}

	// Check tokens as you must have write access to change the order of the queue
	user, err := ts.GetUser(jsonReq.Token)
	if err != nil {
		return 500, ErrorBadToken
	}

	var check bool
	for _, group := range user.Groups {
		check = check || group == common.StandardUser || group == common.Administrator
	}
	if !check {
		return 500, ErrorPermissionDenied
	}

	// Attempt to reorder the Queue
	errs := queue.StackReorder(jsonReq.JobOrder)
	for _, err := range errs {
		if err != nil {
			return 500, ErrorInternal
		}
	}

	// Queue should now be reordered
	return 200, Success
}

func APIJobInfo(req *http.Request, ts common.TokenStore, queue common.Queue) (int, string) {
	// Decode the json request
	jsonReq := APIJobInfoReq{}
	err := json.NewDecoder(req.Body).Decode(jsonReq)
	if err != nil {
		return 500, ErrorMalformedRequest
	}

	// Check the token provided
	if !ts.CheckToken(jsonReq.Token) {
		return 500, ErrorBadToken
	}

	// Get the job info
	job := queue.JobInfo(jsonReq.JobID)

	jsonResp := APIJobInfoResp{}
	jsonResp.Job = job

	resp, err := json.Marshal(jsonResp)
	if err != nil {
		return 500, ErrorInternal
	}

	return 200, string(resp)
}

func APIJobPause(req *http.Request, ts common.TokenStore, queue common.Queue) (int, string) {
	// Decode the json request
	jsonReq := APIJobPauseReq{}
	err := json.NewDecoder(req.Body).Decode(jsonReq)
	if err != nil {
		return 500, ErrorMalformedRequest
	}

	// Check the token provided
	if !ts.CheckToken(jsonReq.Token) {
		return 500, ErrorBadToken
	}

	// Check tokens as you must have write access to pause a job
	user, err := ts.GetUser(jsonReq.Token)
	if err != nil {
		return 500, ErrorBadToken
	}

	var check bool
	for _, group := range user.Groups {
		check = check || group == common.StandardUser || group == common.Administrator
	}
	if !check {
		return 500, ErrorPermissionDenied
	}

	// Pause the job
	err = queue.PauseJob(jsonReq.JobID)
	if err != nil {
		return 500, ErrorJobPause
	}

	return 200, Success
}

func APIJobQuit(req *http.Request, ts common.TokenStore, queue common.Queue) (int, string) {
	// Decode the json request
	jsonReq := APIJobQuitReq{}
	err := json.NewDecoder(req.Body).Decode(jsonReq)
	if err != nil {
		return 500, ErrorMalformedRequest
	}

	// Check the token provided
	if !ts.CheckToken(jsonReq.Token) {
		return 500, ErrorBadToken
	}

	// Check tokens as you must have write access to pause a job
	user, err := ts.GetUser(jsonReq.Token)
	if err != nil {
		return 500, ErrorBadToken
	}

	var check bool
	for _, group := range user.Groups {
		check = check || group == common.StandardUser || group == common.Administrator
	}
	if !check {
		return 500, ErrorPermissionDenied
	}

	// Pause the job
	err = queue.QuitJob(jsonReq.JobID)
	if err != nil {
		return 500, ErrorJobQuit
	}

	return 200, Success
}

func APIResourceList(req *http.Request, ts common.TokenStore, queue common.Queue) (int, string) {
	// Decode the json request
	jsonReq := APIResourceListReq{}
	err := json.NewDecoder(req.Body).Decode(jsonReq)
	if err != nil {
		return 500, ErrorMalformedRequest
	}

	// Check the token
	if !ts.CheckToken(jsonReq.Token) {
		return 500, ErrorBadToken
	}

	// Get the list of resources\
	resp := APIResourceListResp{}
	resp.Resources = queue.GetResources()

	// Encode the response
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		return 500, ErrorInternal
	}

	return 200, string(jsonResp)
}

func APIResourcePause(req *http.Request, ts common.TokenStore, queue common.Queue) (int, string) {
	// Decode the json request
	jsonReq := APIResourcePauseReq{}
	err := json.NewDecoder(req.Body).Decode(jsonReq)
	if err != nil {
		return 500, ErrorMalformedRequest
	}

	// Check the token provided
	if !ts.CheckToken(jsonReq.Token) {
		return 500, ErrorBadToken
	}

	// Check tokens as you must have admin access to manipulate resources
	user, err := ts.GetUser(jsonReq.Token)
	if err != nil {
		return 500, ErrorBadToken
	}

	var check bool
	for _, group := range user.Groups {
		check = check || group == common.Administrator
	}
	if !check {
		return 500, ErrorPermissionDenied
	}

	// We should be good to pause the resource
	err = queue.PauseResource(jsonReq.ResourceID)
	if err != nil {
		return 500, ErrorInternal
	}

	return 200, Success
}

func APIResourceQuit(req *http.Request, ts common.TokenStore, queue common.Queue) (int, string) {
	// Decode the json request
	jsonReq := APIResourceQuitReq{}
	err := json.NewDecoder(req.Body).Decode(jsonReq)
	if err != nil {
		return 500, ErrorMalformedRequest
	}

	// Check the token provided
	if !ts.CheckToken(jsonReq.Token) {
		return 500, ErrorBadToken
	}

	// Check tokens as you must have admin access to manipulate resources
	user, err := ts.GetUser(jsonReq.Token)
	if err != nil {
		return 500, ErrorBadToken
	}

	var check bool
	for _, group := range user.Groups {
		check = check || group == common.Administrator
	}
	if !check {
		return 500, ErrorPermissionDenied
	}

	// We should be good to pause the resource
	err = queue.RemoveResource(jsonReq.ResourceID)
	if err != nil {
		return 500, ErrorInternal
	}

	return 200, Success
}

func APIResourceAdd(req *http.Request, ts common.TokenStore, queue common.Queue) (int, string) {
	// Decode the json request
	jsonReq := APIResourceAddReq{}
	err := json.NewDecoder(req.Body).Decode(jsonReq)
	if err != nil {
		return 500, ErrorMalformedRequest
	}

	// Check the token provided
	if !ts.CheckToken(jsonReq.Token) {
		return 500, ErrorBadToken
	}

	// Check tokens as you must have admin access to manipulate resources
	user, err := ts.GetUser(jsonReq.Token)
	if err != nil {
		return 500, ErrorBadToken
	}

	var check bool
	for _, group := range user.Groups {
		check = check || group == common.Administrator
	}
	if !check {
		return 500, ErrorPermissionDenied
	}

	// Add the resource
	err = queue.AddResource(jsonReq.IPAddress, jsonReq.AuthToken)
	if err != nil {
		return 500, ErrorInternal
	}

	return 200, Success
}
