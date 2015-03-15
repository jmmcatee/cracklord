package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/jmmcatee/cracklord/common"
	"github.com/jmmcatee/cracklord/queue"
	"net/http"
)

// All handler functions are created as part of the base AppController. This is done to
// allow type safe dependency injection to all handler functions. This also make
// expandablility related to adding a database or other dependencies much easier
// for future development.
type AppController struct {
	T    TokenStore
	Auth Authenticator
	Q    queue.Queue
}

func (a *AppController) Router() *mux.Router {
	r := mux.NewRouter().StrictSlash(false)

	// Login and Logout
	r.Path("/api/login").Methods("POST").HandlerFunc(a.Login)
	r.Path("/api/logout").Methods("GET").HandlerFunc(a.Logout)

	// Tools endpoints
	r.Path("/api/tools").Methods("GET").HandlerFunc(a.ListTools)
	r.Path("/api/tools/{id}").Methods("GET").HandlerFunc(a.GetTool)

	// Jobs endpoints
	r.Path("/api/jobs").Methods("GET").HandlerFunc(a.GetJobs)
	r.Path("/api/jobs").Methods("POST").HandlerFunc(a.CreateJob)
	r.Path("/api/jobs/{id}").Methods("GET").HandlerFunc(a.ReadJob)
	r.Path("/api/jobs/{id}").Methods("PUT").HandlerFunc(a.UpdateJob)
	r.Path("/api/jobs/{id}").Methods("DELETE").HandlerFunc(a.DeleteJob)

	// Resource endpoints
	r.Path("/api/resources").Methods("GET").HandlerFunc(a.ListResources)
	r.Path("/api/resources").Methods("POST").HandlerFunc(a.CreateResource)
	r.Path("/api/resources/{id}").Methods("GET").HandlerFunc(a.ReadResource)
	r.Path("/api/resources/{id}").Methods("PUT").HandlerFunc(a.UpdateResource)
	r.Path("/api/resources/{id}").Methods("DELETE").HandlerFunc(a.DeleteResource)

	// Queue endpoints
	r.Path("/api/queue").Methods("PUT").HandlerFunc(a.UpdateQueue)

	return r
}

// Login Hander (POST - /api/login)
func (a *AppController) Login(rw http.ResponseWriter, r *http.Request) {
	// Decode the request and see if it is valid
	reqJSON := json.NewDecoder(r.Body)
	respJSON := json.NewEncoder(rw)

	var req = LoginReq{}
	var resp = LoginResp{}

	err := reqJSON.Decode(&req)
	if err != nil {
		// We had an error decoding the request to return an error
		resp.Status = RESP_CODE_BADREQ
		resp.Message = RESP_CODE_BADREQ_T
		resp.Token = ""

		// TODO: Eventually need to log this error
		rw.WriteHeader(RESP_CODE_BADREQ)
		respJSON.Encode(resp)
		return
	}

	// Verify the login
	user, err := a.Auth.Login(req.Username, req.Password)
	if err != nil {
		// Login failed so return error
		resp.Status = RESP_CODE_UNAUTHORIZED
		resp.Message = RESP_CODE_UNAUTHORIZED_T
		resp.Token = ""

		rw.WriteHeader(RESP_CODE_UNAUTHORIZED)
		respJSON.Encode(resp)
	}

	// Generate token
	seed := make([]byte, 256)
	bToken := sha256.New()

	rand.Read(seed)

	token := hex.EncodeToString(bToken.Sum(seed))

	// Add to the token store
	a.T.AddToken(token, user)

	// Return new information
	resp.Status = RESP_CODE_OK
	resp.Message = RESP_CODE_OK_T
	resp.Token = token

	rw.WriteHeader(RESP_CODE_OK)
	respJSON.Encode(resp)
}

// Logout endpoint (POST - /api/logout)
func (a *AppController) Logout(rw http.ResponseWriter, r *http.Request) {
	// Response and Request structures
	var resp = LogoutResp{}

	// Build the JSON Decoder
	respJSON := json.NewEncoder(rw)

	// Get the token from the Query string
	token := r.URL.Query().Get("token")

	a.T.RemoveToken(token)

	resp.Status = RESP_CODE_OK
	resp.Message = RESP_CODE_OK_T

	rw.WriteHeader(RESP_CODE_OK)
	respJSON.Encode(resp)
}

// List Tools endpoint (GET - /api/tools)
func (a *AppController) ListTools(rw http.ResponseWriter, r *http.Request) {
	// Resposne and Request structures
	var resp ToolsResp

	// Check the Token provided
	token := r.URL.Query().Get("token")

	// JSON Encoder and Decoder
	respJSON := json.NewEncoder(rw)

	if !a.T.CheckToken(token) {
		resp.Status = RESP_CODE_UNAUTHORIZED
		resp.Message = RESP_CODE_UNAUTHORIZED_T

		rw.WriteHeader(RESP_CODE_UNAUTHORIZED)
		respJSON.Encode(resp)
		return
	}

	// Get the tools list from the Queue
	var tmap APITools
	for _, t := range a.Q.Tools() {
		tmap[t.UUID] = APITool{Name: t.Name, Version: t.Version}
	}

	// Build response
	resp.Status = RESP_CODE_OK
	resp.Message = RESP_CODE_OK_T
	resp.Tools = tmap

	rw.WriteHeader(RESP_CODE_OK)
	respJSON.Encode(resp)
}

// Get Tool Endpoint (GET - /api/tools/{id})
func (a *AppController) GetTool(rw http.ResponseWriter, r *http.Request) {
	// Response and Request structures
	var resp ToolsGetResp

	// Check the token
	token := r.URL.Query().Get("token")

	// JSON Encoder and Decoder
	respJSON := json.NewEncoder(rw)

	if !a.T.CheckToken(token) {
		resp.Status = RESP_CODE_UNAUTHORIZED
		resp.Message = RESP_CODE_UNAUTHORIZED_T

		rw.WriteHeader(RESP_CODE_UNAUTHORIZED)
		respJSON.Encode(resp)
		return
	}

	// Get the tool ID
	uuid := mux.Vars(r)["id"]
	tool, ok := a.Q.Tools()[uuid]
	if !ok {
		// No tool found, return error
		resp.Status = RESP_CODE_NOTFOUND
		resp.Message = RESP_CODE_NOTFOUND_T

		rw.WriteHeader(RESP_CODE_NOTFOUND)
		respJSON.Encode(resp)
	}

	// We need to split the response from the tool into Form and Schema
	var form common.ToolJSONForm
	jsonBuf := bytes.NewBuffer([]byte(tool.Parameters))
	err := json.NewDecoder(jsonBuf).Decode(&form)
	if err != nil {
		resp.Status = RESP_CODE_ERROR
		resp.Message = RESP_CODE_ERROR_T

		rw.WriteHeader(RESP_CODE_ERROR)
		respJSON.Encode(resp)
		return
	}

	// We found the tools so return it in the resp structure
	resp.Status = RESP_CODE_OK
	resp.Message = RESP_CODE_OK_T
	resp.Name = tool.Name
	resp.Version = tool.Version
	resp.Form = form.Form
	resp.Schema = form.Schema

	rw.WriteHeader(RESP_CODE_OK)
	respJSON.Encode(resp)
}

// Get Job list (GET - /api/jobs)
func (a *AppController) GetJobs(rw http.ResponseWriter, r *http.Request) {
	// Response and Request structures
	var resp GetJobsResp

	// JSON Encoder and Decoder
	respJSON := json.NewEncoder(rw)

	// Check the token
	token := r.URL.Query().Get("token")
	if !a.T.CheckToken(token) {
		resp.Status = RESP_CODE_UNAUTHORIZED
		resp.Message = RESP_CODE_UNAUTHORIZED_T

		rw.WriteHeader(RESP_CODE_UNAUTHORIZED)
		respJSON.Encode(resp)
		return
	}

	// Get the list of jobs and populate a return structure
	for _, j := range a.Q.AllJobs() {
		var job APIJob

		job.JobID = j.UUID
		job.Name = j.Name
		job.JobStatus = j.Status
		job.Owner = j.Owner
		job.StartTime = j.StartTime
		job.CrackedHashes = j.CrackedHashes
		job.TotalHashes = j.TotalHashes
		job.Percentage = j.Percentage

		resp.Jobs = append(resp.Jobs, job)
	}

	// Return the results
	resp.Status = RESP_CODE_OK
	resp.Message = RESP_CODE_OK_T

	rw.WriteHeader(RESP_CODE_OK)
	respJSON.Encode(resp)
}

// Create a new job (POST - /api/job)
func (a *AppController) CreateJob(rw http.ResponseWriter, r *http.Request) {
	// Response and Request structures
	var req JobCreateReq
	var resp JobCreateResp

	// JSON Encoder and Decoder
	reqJSON := json.NewDecoder(r.Body)
	respJSON := json.NewEncoder(rw)

	// Decode the request
	err := reqJSON.Decode(&req)
	if err != nil {
		resp.Status = RESP_CODE_BADREQ
		resp.Message = RESP_CODE_BADREQ_T

		rw.WriteHeader(RESP_CODE_BADREQ)
		respJSON.Encode(resp)
		return
	}

	// Check Token
	if !a.T.CheckToken(req.Token) {
		resp.Status = RESP_CODE_UNAUTHORIZED
		resp.Message = RESP_CODE_UNAUTHORIZED_T

		rw.WriteHeader(RESP_CODE_UNAUTHORIZED)
		respJSON.Encode(resp)
		return
	}

	// Get the user
	user, _ := a.T.GetUser(req.Token) // Ignoring the error because we know the token is good

	// Build a job structure
	job := common.NewJob(req.ToolID, req.Name, user.Username, req.Params)

	err = a.Q.AddJob(job)
	if err != nil {
		resp.Status = RESP_CODE_BADREQ
		resp.Message = RESP_CODE_BADREQ_T

		rw.WriteHeader(RESP_CODE_BADREQ)
		respJSON.Encode(resp)
		return
	}

	// Job was created so populate the response structure and return
	resp.Status = RESP_CODE_OK
	resp.Message = RESP_CODE_OK_T
	resp.JobID = job.UUID

	rw.WriteHeader(RESP_CODE_OK)
	respJSON.Encode(resp)
}

// Read an individual Job (GET - /api/jobs/{id})
func (a *AppController) ReadJob(rw http.ResponseWriter, r *http.Request) {
	// Response and Request structures
	var resp JobReadResp

	// JSON Encoder and Decoder
	respJSON := json.NewEncoder(rw)

	// Get the token from the URI
	token := r.URL.Query().Get("token")

	// Check the token
	if !a.T.CheckToken(token) {
		resp.Status = RESP_CODE_UNAUTHORIZED
		resp.Message = RESP_CODE_UNAUTHORIZED_T

		rw.WriteHeader(RESP_CODE_UNAUTHORIZED)
		respJSON.Encode(resp)
		return
	}

	// Get the ID of the job we want
	jobid := mux.Vars(r)["id"]

	// Pull Job info from the Queue
	job := a.Q.JobInfo(jobid)

	// Build the response structure
	resp.Status = RESP_CODE_OK
	resp.Message = RESP_CODE_OK_T
	resp.JobID = job.UUID
	resp.Name = job.Name
	resp.JobStatus = job.Status
	resp.Owner = job.Owner
	resp.StartTime = job.StartTime
	resp.CrackedHashes = job.CrackedHashes
	resp.TotalHashes = job.TotalHashes
	resp.Percentage = job.Percentage
	resp.Performance = job.Performance
	resp.PerformanceTitle = job.PerformanceTitle
	resp.Output = job.Output

	rw.WriteHeader(RESP_CODE_OK)
	respJSON.Encode(resp)
}

// Update a job
func (a *AppController) UpdateJob(rw http.ResponseWriter, r *http.Request) {
	// Response and Request structures
	var req JobUpdateReq
	var resp JobUpdateResp

	// JSON Encoder and Decoder
	reqJSON := json.NewDecoder(r.Body)
	respJSON := json.NewEncoder(rw)

	// Decode the request
	err := reqJSON.Decode(&req)
	if err != nil {
		resp.Status = RESP_CODE_BADREQ
		resp.Message = RESP_CODE_BADREQ_T

		rw.WriteHeader(RESP_CODE_BADREQ)
		respJSON.Encode(resp)
		return
	}

	// Check Token
	if !a.T.CheckToken(req.Token) {
		resp.Status = RESP_CODE_UNAUTHORIZED
		resp.Message = RESP_CODE_UNAUTHORIZED_T

		rw.WriteHeader(RESP_CODE_UNAUTHORIZED)
		respJSON.Encode(resp)
		return
	}

	// Get the ID of the job we want
	jobid := mux.Vars(r)["id"]

	// Get the action requested
	switch req.Action {
	case "pause":
		// Pause the job
		err = a.Q.PauseJob(jobid)
		if err != nil {
			resp.Status = RESP_CODE_ERROR
			resp.Message = RESP_CODE_ERROR_T

			rw.WriteHeader(RESP_CODE_ERROR)
			respJSON.Encode(resp)
			return
		}
	case "stop":
		// Stop the job
		err = a.Q.QuitJob(jobid)
		if err != nil {
			resp.Status = RESP_CODE_ERROR
			resp.Message = RESP_CODE_ERROR_T

			rw.WriteHeader(RESP_CODE_ERROR)
			respJSON.Encode(resp)
			return
		}
	}

	// Now return everything is good and the job info
	j := a.Q.JobInfo(jobid)

	resp.Status = RESP_CODE_OK
	resp.Message = RESP_CODE_OK_T
	resp.Job.JobID = j.UUID
	resp.Job.Name = j.Name
	resp.Job.JobStatus = j.Status
	resp.Job.Owner = j.Owner
	resp.Job.StartTime = j.StartTime
	resp.Job.CrackedHashes = j.CrackedHashes
	resp.Job.TotalHashes = j.TotalHashes
	resp.Job.Percentage = j.Percentage

	rw.WriteHeader(RESP_CODE_OK)
	respJSON.Encode(resp)
}

func (a *AppController) DeleteJob(rw http.ResponseWriter, r *http.Request) {
	// Response and Request structures
	var req JobDeleteReq
	var resp JobDeleteResp

	// JSON Encoders and Decoders
	reqJSON := json.NewDecoder(r.Body)
	respJSON := json.NewEncoder(rw)

	// Decode the request
	err := reqJSON.Decode(&req)
	if err != nil {
		resp.Status = RESP_CODE_BADREQ
		resp.Message = RESP_CODE_BADREQ_T

		rw.WriteHeader(RESP_CODE_BADREQ)
		respJSON.Encode(resp)
		return
	}

	// Check Token
	if !a.T.CheckToken(req.Token) {
		resp.Status = RESP_CODE_UNAUTHORIZED
		resp.Message = RESP_CODE_UNAUTHORIZED_T

		rw.WriteHeader(RESP_CODE_UNAUTHORIZED)
		respJSON.Encode(resp)
		return
	}

	// Get the ID of the job we want
	jobid := mux.Vars(r)["id"]

	// Remove the job
	err = a.Q.RemoveJob(jobid)
	if err != nil {
		resp.Status = RESP_CODE_ERROR
		resp.Message = RESP_CODE_ERROR_T

		rw.WriteHeader(RESP_CODE_ERROR)
		respJSON.Encode(resp)
		return
	}

	// Job should now be removed, so return all OK
	resp.Status = RESP_CODE_OK
	resp.Message = RESP_CODE_OK_T

	rw.WriteHeader(RESP_CODE_OK)
	respJSON.Encode(resp)
}

func (a *AppController) ListResources(rw http.ResponseWriter, r *http.Request) {
	// Response and request structures
	var resp ResourceListResp

	// JSON encoders and decoders
	respJSON := json.NewEncoder(rw)

	// Get the token from the URI
	token := r.URL.Query().Get("token")

	// Check Token
	if !a.T.CheckToken(token) {
		resp.Status = RESP_CODE_UNAUTHORIZED
		resp.Message = RESP_CODE_UNAUTHORIZED_T

		rw.WriteHeader(RESP_CODE_UNAUTHORIZED)
		respJSON.Encode(resp)
		return
	}

	// Get resources and fill return structure
	for _, v := range a.Q.GetResources() {
		var res APIResource

		res.ResourceID = v.UUID
		if v.Paused {
			res.Status = "paused"
		} else {
			res.Status = "running"
		}

		for tID, tv := range v.Tools {
			var apitool APITool
			apitool.Name = tv.Name
			apitool.Version = tv.Version

			res.Tools[tID] = apitool
		}

		resp.Resources = append(resp.Resources, res)
	}

	// Return the structure
	resp.Status = RESP_CODE_OK
	resp.Message = RESP_CODE_OK_T

	rw.WriteHeader(RESP_CODE_OK)
	respJSON.Encode(resp)
}

func (a *AppController) CreateResource(rw http.ResponseWriter, r *http.Request) {
	// Response and request structures
	var req ResourceCreateReq
	var resp ResourceCreateResp

	// JSON encoders and decoders
	reqJSON := json.NewDecoder(r.Body)
	respJSON := json.NewEncoder(rw)

	// Decode the request
	err := reqJSON.Decode(&req)
	if err != nil {
		resp.Status = RESP_CODE_BADREQ
		resp.Message = RESP_CODE_BADREQ_T

		rw.WriteHeader(RESP_CODE_BADREQ)
		respJSON.Encode(resp)
		return
	}

	// Check Token
	if !a.T.CheckToken(req.Token) {
		resp.Status = RESP_CODE_UNAUTHORIZED
		resp.Message = RESP_CODE_UNAUTHORIZED_T

		rw.WriteHeader(RESP_CODE_UNAUTHORIZED)
		respJSON.Encode(resp)
		return
	}

	// Add resource
	err = a.Q.AddResource(req.Host, req.Key)
	if err != nil {
		resp.Status = RESP_CODE_ERROR
		resp.Message = RESP_CODE_ERROR_T

		rw.WriteHeader(RESP_CODE_ERROR)
		respJSON.Encode(resp)
		return
	}

	// Return the structure
	resp.Status = RESP_CODE_OK
	resp.Message = RESP_CODE_OK_T

	rw.WriteHeader(RESP_CODE_OK)
	respJSON.Encode(resp)
}

func (a *AppController) ReadResource(rw http.ResponseWriter, r *http.Request) {
	// Response and Request structures
	var req ResourceReadReq
	var resp ResourceReadResp

	// JSON encoders and decoders
	reqJSON := json.NewDecoder(r.Body)
	respJSON := json.NewEncoder(rw)

	// Decode the request
	err := reqJSON.Decode(&req)
	if err != nil {
		resp.Status = RESP_CODE_BADREQ
		resp.Message = RESP_CODE_BADREQ_T

		rw.WriteHeader(RESP_CODE_BADREQ)
		respJSON.Encode(resp)
		return
	}

	// Check Token
	if !a.T.CheckToken(req.Token) {
		resp.Status = RESP_CODE_UNAUTHORIZED
		resp.Message = RESP_CODE_UNAUTHORIZED_T

		rw.WriteHeader(RESP_CODE_UNAUTHORIZED)
		respJSON.Encode(resp)
		return
	}

	// Get the ID of the job we want
	resid := mux.Vars(r)["id"]

	// Get the resource by ID
	var found bool
	for _, v := range a.Q.GetResources() {
		// look for ID
		if v.UUID == resid {
			// Map the hardware
			for h, _ := range v.Hardware {
				resp.Hardware = append(resp.Hardware, h)
			}

			// Map resource status
			if v.Paused {
				resp.ResourceStatus = "paused"
			} else {
				resp.ResourceStatus = "running"
			}

			// Map the tools
			for _, tv := range v.Tools {
				var apitool APITool

				apitool.Name = tv.Name
				apitool.Version = tv.Version

				resp.Tools[tv.UUID] = apitool
			}

			// Note that we found a resource
			found = true
		}
	}

	// Check if we have a resource to return
	if !found {
		resp.Status = RESP_CODE_NOCONTENT
		resp.Message = RESP_CODE_NOCONTENT_T

		rw.WriteHeader(RESP_CODE_NOCONTENT)
		respJSON.Encode(resp)
		return
	}

	// Return the structure
	resp.Status = RESP_CODE_OK
	resp.Message = RESP_CODE_OK_T

	rw.WriteHeader(RESP_CODE_OK)
	respJSON.Encode(resp)
}

func (a *AppController) UpdateResource(rw http.ResponseWriter, r *http.Request) {
	// Response and Request structures
	var req ResourceUpdateReq
	var resp ResourceUpdateResp

	// JSON decoder and encoders
	reqJSON := json.NewDecoder(r.Body)
	respJSON := json.NewEncoder(rw)

	// Decode the request
	err := reqJSON.Decode(&req)
	if err != nil {
		resp.Status = RESP_CODE_BADREQ
		resp.Message = RESP_CODE_BADREQ_T

		rw.WriteHeader(RESP_CODE_BADREQ)
		respJSON.Encode(resp)
		return
	}

	// Check Token
	if !a.T.CheckToken(req.Token) {
		resp.Status = RESP_CODE_UNAUTHORIZED
		resp.Message = RESP_CODE_UNAUTHORIZED_T

		rw.WriteHeader(RESP_CODE_UNAUTHORIZED)
		respJSON.Encode(resp)
		return
	}

	// Get the ID of the job we want
	resid := mux.Vars(r)["id"]

	// Check the action for resume or pause
	switch req.Action {
	case "pause":
		err = a.Q.PauseResource(resid)
		if err != nil {
			resp.Status = RESP_CODE_ERROR
			resp.Message = RESP_CODE_ERROR_T

			rw.WriteHeader(RESP_CODE_ERROR)
			respJSON.Encode(resp)
			return
		}
	case "resume":
		err = a.Q.ResumeResource(resid)
		if err != nil {
			resp.Status = RESP_CODE_ERROR
			resp.Message = RESP_CODE_ERROR_T

			rw.WriteHeader(RESP_CODE_ERROR)
			respJSON.Encode(resp)
			return
		}
	}

	// No errors so we should be good
	resp.Status = RESP_CODE_OK
	resp.Message = RESP_CODE_OK_T

	rw.WriteHeader(RESP_CODE_OK)
	respJSON.Encode(resp)
}

func (a *AppController) DeleteResource(rw http.ResponseWriter, r *http.Request) {
	// Request and response structure
	var req ResourceDeleteReq
	var resp ResourceDeleteResp

	// JSON decoders and encoders
	reqJSON := json.NewDecoder(r.Body)
	respJSON := json.NewEncoder(rw)

	// Decode the request
	err := reqJSON.Decode(&req)
	if err != nil {
		resp.Status = RESP_CODE_BADREQ
		resp.Message = RESP_CODE_BADREQ_T

		rw.WriteHeader(RESP_CODE_BADREQ)
		respJSON.Encode(resp)
		return
	}

	// Check Token
	if !a.T.CheckToken(req.Token) {
		resp.Status = RESP_CODE_UNAUTHORIZED
		resp.Message = RESP_CODE_UNAUTHORIZED_T

		rw.WriteHeader(RESP_CODE_UNAUTHORIZED)
		respJSON.Encode(resp)
		return
	}

	// Get the ID of the job we want
	resid := mux.Vars(r)["id"]

	// Attempt to remove the resource
	err = a.Q.RemoveResource(resid)
	if err != nil {
		resp.Status = RESP_CODE_ERROR
		resp.Message = RESP_CODE_ERROR_T

		rw.WriteHeader(RESP_CODE_ERROR)
		respJSON.Encode(resp)
		return
	}

	// No errors so we should be good
	resp.Status = RESP_CODE_OK
	resp.Message = RESP_CODE_OK_T

	rw.WriteHeader(RESP_CODE_OK)
	respJSON.Encode(resp)
}

func (a *AppController) UpdateQueue(rw http.ResponseWriter, r *http.Request) {
	// Request and response structures
	var req QueueUpdateReq
	var resp QueueUpdateResp

	// JSON decoders and encoders
	reqJSON := json.NewDecoder(r.Body)
	respJSON := json.NewEncoder(rw)

	// Decode the request
	err := reqJSON.Decode(&req)
	if err != nil {
		resp.Status = RESP_CODE_BADREQ
		resp.Message = RESP_CODE_BADREQ_T

		rw.WriteHeader(RESP_CODE_BADREQ)
		respJSON.Encode(resp)
		return
	}

	// Check Token
	if !a.T.CheckToken(req.Token) {
		resp.Status = RESP_CODE_UNAUTHORIZED
		resp.Message = RESP_CODE_UNAUTHORIZED_T

		rw.WriteHeader(RESP_CODE_UNAUTHORIZED)
		respJSON.Encode(resp)
		return
	}

	// Reorder the queue
	errs := a.Q.StackReorder(req.JobOrder)
	if len(errs) != 0 {
		// TODO: Log erros
		resp.Status = RESP_CODE_ERROR
		resp.Message = RESP_CODE_ERROR_T

		rw.WriteHeader(RESP_CODE_ERROR)
		respJSON.Encode(resp)
		return
	}

	// No errors so we should be good
	resp.Status = RESP_CODE_OK
	resp.Message = RESP_CODE_OK_T

	rw.WriteHeader(RESP_CODE_OK)
	respJSON.Encode(resp)
}
