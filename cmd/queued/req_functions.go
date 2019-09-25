package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/jmmcatee/cracklord/common"
	"github.com/jmmcatee/cracklord/common/queue"
)

// AppController all handler functions are created as part of the base AppController. This is done to
// allow type safe dependency injection to all handler functions. This also make
// expandablility related to adding a database or other dependencies much easier
// for future development.
type AppController struct {
	T    TokenStore
	Auth Authenticator
	Q    *queue.Queue
	TLS  *tls.Config
}

// Returns the muxRouter
func (a *AppController) Router() *mux.Router {
	r := mux.NewRouter().StrictSlash(false)

	// Login and Logout
	r.Path("/api/login").Methods("POST").HandlerFunc(a.Login)
	r.Path("/api/logout").Methods("GET").HandlerFunc(a.Logout)

	// Tools endpoints
	r.Path("/api/tools").Methods("GET").HandlerFunc(a.ListTools)
	r.Path("/api/tools/{id}").Methods("GET").HandlerFunc(a.GetTool)

	// Resource Manager endpoints
	r.Path("/api/resourcemanagers").Methods("GET").HandlerFunc(a.ListResourceManagers)
	r.Path("/api/resourcemanagers/{id}").Methods("GET").HandlerFunc(a.GetResourceManager)

	// Resource endpoints
	r.Path("/api/resources").Methods("GET").HandlerFunc(a.ListResource)
	r.Path("/api/resources").Methods("POST").HandlerFunc(a.CreateResource)
	r.Path("/api/resources/{manager}/{id}").Methods("GET").HandlerFunc(a.ReadResource)
	r.Path("/api/resources/{id}").Methods("PUT").HandlerFunc(a.UpdateResource)
	r.Path("/api/resources/{id}").Methods("DELETE").HandlerFunc(a.DeleteResources)

	// Jobs endpoints
	r.Path("/api/jobs").Methods("GET").HandlerFunc(a.GetJobs)
	r.Path("/api/jobs").Methods("POST").HandlerFunc(a.CreateJob)
	r.Path("/api/jobs/{id}").Methods("GET").HandlerFunc(a.ReadJob)
	r.Path("/api/jobs/{id}").Methods("PUT").HandlerFunc(a.UpdateJob)
	r.Path("/api/jobs/{id}").Methods("DELETE").HandlerFunc(a.DeleteJob)

	// Queue endpoints
	r.Path("/api/queue").Methods("PUT").HandlerFunc(a.ReorderQueue)

	log.Debug("Application router handlers configured.")

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

		log.Error("Unable to decode login information provided.")
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

		log.WithField("username", req.Username).Warn("Login failed.")

		rw.WriteHeader(RESP_CODE_UNAUTHORIZED)
		respJSON.Encode(resp)

		return
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
	resp.Role = user.EffectiveRole()

	rw.WriteHeader(RESP_CODE_OK)
	respJSON.Encode(resp)
	log.WithField("username", req.Username).Info("User successfully logged in")
}

// Logout endpoint (POST - /api/logout)
func (a *AppController) Logout(rw http.ResponseWriter, r *http.Request) {
	// Response and Request structures
	var resp = LogoutResp{}

	// Build the JSON Decoder
	respJSON := json.NewEncoder(rw)

	// Get the authorization header
	token := r.Header.Get("AuthorizationToken")

	u, _ := a.T.GetUser(token)
	a.T.RemoveToken(token)

	resp.Status = RESP_CODE_OK
	resp.Message = RESP_CODE_OK_T

	rw.WriteHeader(RESP_CODE_OK)
	respJSON.Encode(resp)
	log.WithField("username", u.Username).Info("User successfully logged out.")
}

// List Tools endpoint (GET - /api/tools)
func (a *AppController) ListTools(rw http.ResponseWriter, r *http.Request) {
	// Resposne and Request structures
	var resp ToolsResp

	// JSON Encoder and Decoder
	respJSON := json.NewEncoder(rw)

	// Get the authorization header
	token := r.Header.Get("AuthorizationToken")

	if !a.T.CheckToken(token) {
		resp.Status = RESP_CODE_UNAUTHORIZED
		resp.Message = RESP_CODE_UNAUTHORIZED_T

		rw.WriteHeader(RESP_CODE_UNAUTHORIZED)
		respJSON.Encode(resp)
		log.WithField("token", token).Warn("An unknown user token attempted to list tools.")
		return
	}

	// Check for standard user level at least
	user, _ := a.T.GetUser(token)
	if !user.Allowed(StandardUser) {
		resp.Status = RESP_CODE_UNAUTHORIZED
		resp.Message = RESP_CODE_UNAUTHORIZED_T

		rw.WriteHeader(RESP_CODE_UNAUTHORIZED)
		respJSON.Encode(resp)
		log.WithField("user", user.Username).Warn("An unauthorized user token attempted to list tools.")
		return
	}

	// Get the tools list from the Queue
	for uuid, t := range a.Q.ActiveTools() {
		resp.Tools = append(resp.Tools, APITool{uuid, t.Name, t.Version})
		log.WithFields(log.Fields{
			"uuid": t.UUID,
			"name": t.Name,
			"ver":  t.Version,
		}).Debug("Gathered tool")
	}

	// Build response
	resp.Status = RESP_CODE_OK
	resp.Message = RESP_CODE_OK_T

	rw.WriteHeader(RESP_CODE_OK)
	respJSON.Encode(resp)
	log.Info("Provided a tool listing to API")
}

// Get Tool Endpoint (GET - /api/tools/{id})
func (a *AppController) GetTool(rw http.ResponseWriter, r *http.Request) {
	// Response and Request structures
	var resp ToolsGetResp

	// JSON Encoder and Decoder
	respJSON := json.NewEncoder(rw)

	// Get the authorization header
	token := r.Header.Get("AuthorizationToken")

	if !a.T.CheckToken(token) {
		resp.Status = RESP_CODE_UNAUTHORIZED
		resp.Message = RESP_CODE_UNAUTHORIZED_T

		rw.WriteHeader(RESP_CODE_UNAUTHORIZED)
		respJSON.Encode(resp)
		log.WithField("token", token).Warn("An unknown user token attempted to get tool details.")
		return
	}

	// Check for standard user level at least
	user, _ := a.T.GetUser(token)
	if !user.Allowed(StandardUser) {
		resp.Status = RESP_CODE_UNAUTHORIZED
		resp.Message = RESP_CODE_UNAUTHORIZED_T

		rw.WriteHeader(RESP_CODE_UNAUTHORIZED)
		respJSON.Encode(resp)
		log.WithField("user", user.Username).Warn("An unauthorized user token attempted to get tool details.")
		return
	}

	// Get the tool ID
	uuid := mux.Vars(r)["id"]
	tool, ok := a.Q.ActiveTools()[uuid]
	if !ok {
		// No tool found, return error
		resp.Status = RESP_CODE_NOTFOUND
		resp.Message = RESP_CODE_NOTFOUND_T

		rw.WriteHeader(RESP_CODE_NOTFOUND)
		respJSON.Encode(resp)
		return
	}

	// We need to split the response from the tool into Form and Schema
	var form common.JSONSchemaForm

	jsonBuf := bytes.NewBuffer([]byte(tool.Parameters))
	err := json.NewDecoder(jsonBuf).Decode(&form)
	if err != nil {
		log.WithField("error", err.Error()).Error("There was a problem parsing tool form schema JSON.")
		resp.Status = RESP_CODE_ERROR
		resp.Message = "There was an error parsing the tool form information: " + err.Error()

		rw.WriteHeader(RESP_CODE_ERROR)
		respJSON.Encode(resp)
		return
	}

	// We found the tools so return it in the resp structure
	resp.Status = RESP_CODE_OK
	resp.Message = RESP_CODE_OK_T
	resp.Tool.ID = tool.UUID
	resp.Tool.Name = tool.Name
	resp.Tool.Version = tool.Version
	resp.Tool.Form = &form.Form
	resp.Tool.Schema = &form.Schema

	rw.WriteHeader(RESP_CODE_OK)
	respJSON.Encode(resp)

	log.WithFields(log.Fields{
		"name": tool.Name,
		"ver":  tool.Version,
	}).Info("Detailed information on tool sent to API")
}

// List Resource Managers endpoint (GET - /api/resourcemanagers)
// This function will provide a list of all resource managers and their IDs to the API
// in the form of a javascript array of objects.
func (a *AppController) ListResourceManagers(rw http.ResponseWriter, r *http.Request) {
	// Resposne and Request structures
	var resp ResourceManagersResp

	// JSON Encoder and Decoder
	respJSON := json.NewEncoder(rw)

	// Get the authorization header
	token := r.Header.Get("AuthorizationToken")

	//Check to make sure our token is valid, and if not return an error to the API
	if !a.T.CheckToken(token) {
		resp.Status = RESP_CODE_UNAUTHORIZED
		resp.Message = RESP_CODE_UNAUTHORIZED_T

		rw.WriteHeader(RESP_CODE_UNAUTHORIZED)
		respJSON.Encode(resp)
		log.WithField("token", token).Warn("An unknown user token attempted to list resource managers.")
		return
	}

	// Check for the read only user level as this is just data gathering.
	user, _ := a.T.GetUser(token)
	if !user.Allowed(ReadOnly) {
		// If the user isn't allowed or the token isn't valid return an HTTP
		// Unauthorized to the user.
		resp.Status = RESP_CODE_UNAUTHORIZED
		resp.Message = RESP_CODE_UNAUTHORIZED_T

		//Write out the unauthorized response
		rw.WriteHeader(RESP_CODE_UNAUTHORIZED)
		respJSON.Encode(resp)
		log.WithField("user", user.Username).Warn("An unauthorized user token attempted to list resource managers.")
		return
	}

	// Get the map of all resource managers from the Queue
	for resmgrid, resmgrdata := range a.Q.AllResourceManagers() {
		resp.ResourceManagers = append(resp.ResourceManagers,
			APIResourceManager{
				ID:   resmgrid,
				Name: resmgrdata.DisplayName(),
			})
		log.WithFields(log.Fields{
			"id":   resmgrid,
			"name": resmgrdata.DisplayName(),
		}).Debug("Added resource manager to list")
	}

	// Build response of 200 for the API Status and Message portions
	resp.Status = RESP_CODE_OK
	resp.Message = RESP_CODE_OK_T

	//Write out the HTTP 200 header
	rw.WriteHeader(RESP_CODE_OK)
	// Write out our response to the response writer in JSON format
	respJSON.Encode(resp)

	//Log it for the end user
	log.Info("Provided a resource manager listing to API")
}

// Get the details on a single resource manager (GET /api/resourcemanagers/{id})
func (a *AppController) GetResourceManager(rw http.ResponseWriter, r *http.Request) {
	// Response and Request structures
	var resp ResourceManagerGetResp

	// JSON Encoder and Decoder
	respJSON := json.NewEncoder(rw)

	// Get the authorization header
	token := r.Header.Get("AuthorizationToken")

	// Check to make sure our user token is valid
	if !a.T.CheckToken(token) {
		resp.Status = RESP_CODE_UNAUTHORIZED
		resp.Message = RESP_CODE_UNAUTHORIZED_T

		rw.WriteHeader(RESP_CODE_UNAUTHORIZED)
		respJSON.Encode(resp)
		log.WithField("token", token).Warn("An unknown user token attempted to get tool details.")
		return
	}

	// Check for the read only level as this is just information we're returning
	user, _ := a.T.GetUser(token)
	if !user.Allowed(ReadOnly) {
		resp.Status = RESP_CODE_UNAUTHORIZED
		resp.Message = RESP_CODE_UNAUTHORIZED_T

		rw.WriteHeader(RESP_CODE_UNAUTHORIZED)
		respJSON.Encode(resp)
		log.WithField("user", user.Username).Warn("An unauthorized user token attempted to get tool details.")
		return
	}

	// Get the resource manager ID from the URL
	systemname := mux.Vars(r)["id"]

	// Get the resource manager object itself
	resmgr, ok := a.Q.GetResourceManager(systemname)
	if !ok {
		// The resource manager was not found, let's return that in proper HTTP
		resp.Status = RESP_CODE_NOTFOUND
		resp.Message = "That resource manager could not be found."

		rw.WriteHeader(RESP_CODE_NOTFOUND)
		respJSON.Encode(resp)
		return
	}

	form := json.RawMessage(resmgr.ParametersForm())
	schema := json.RawMessage(resmgr.ParametersSchema())

	// Now since everything seems ok, let's build up our response and send it off
	// to the API.
	resp.Status = RESP_CODE_OK
	resp.Message = RESP_CODE_OK_T
	//Resp.ResourceManager is of the type APIResourceManagerDetail
	resp.ResourceManager.ID = resmgr.SystemName()
	resp.ResourceManager.Name = resmgr.DisplayName()
	resp.ResourceManager.Description = resmgr.Description()
	resp.ResourceManager.Form = &form
	resp.ResourceManager.Schema = &schema

	// Write out the HTTP OK header
	rw.WriteHeader(RESP_CODE_OK)
	//Encode and write out our response
	err := respJSON.Encode(resp)
	if err != nil {
		log.WithField("error", err.Error()).Error("Unable to encode resource manager details.")
	}

	log.WithField("id", resmgr.SystemName()).Info("Detailed information on resource manager sent to API")
}

// Get Job list (GET - /api/jobs)
func (a *AppController) GetJobs(rw http.ResponseWriter, r *http.Request) {
	// Response and Request structures
	var resp GetJobsResp

	// JSON Encoder and Decoder
	respJSON := json.NewEncoder(rw)

	// Get the authorization header
	token := r.Header.Get("AuthorizationToken")

	if !a.T.CheckToken(token) {
		resp.Status = RESP_CODE_UNAUTHORIZED
		resp.Message = RESP_CODE_UNAUTHORIZED_T

		rw.WriteHeader(RESP_CODE_UNAUTHORIZED)
		respJSON.Encode(resp)
		log.WithField("token", token).Warn("An unknown user token attempted to get a job listing")
		return
	}

	// Get the list of jobs and populate a return structure
	for _, j := range a.Q.AllJobs() {
		var job APIJob

		job.ID = j.UUID
		job.Name = j.Name
		job.Status = j.Status
		job.ResourceID = j.ResAssigned
		job.Owner = j.Owner
		job.StartTime = j.StartTime
		job.ETC = j.ETC
		job.CrackedHashes = j.CrackedHashes
		job.TotalHashes = j.TotalHashes
		job.Progress = j.Progress
		job.ToolID = j.ToolUUID

		resp.Jobs = append(resp.Jobs, job)
		log.WithFields(log.Fields{
			"uuid":   j.UUID,
			"name":   j.Name,
			"status": j.Status,
		}).Debug("Gathered job for query listing.")
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

	// Get the authorization header
	token := r.Header.Get("AuthorizationToken")

	if !a.T.CheckToken(token) {
		resp.Status = RESP_CODE_UNAUTHORIZED
		resp.Message = RESP_CODE_UNAUTHORIZED_T

		rw.WriteHeader(RESP_CODE_UNAUTHORIZED)
		respJSON.Encode(resp)
		log.Warn("An unknown token attempted to create a job.")
		return
	}

	// Check for standard user level at least
	user, _ := a.T.GetUser(token)
	if !user.Allowed(StandardUser) {
		resp.Status = RESP_CODE_UNAUTHORIZED
		resp.Message = RESP_CODE_UNAUTHORIZED_T

		rw.WriteHeader(RESP_CODE_UNAUTHORIZED)
		respJSON.Encode(resp)
		log.WithField("user", user.Username).Warn("An unauthorized user attempted to create a job.")
		return
	}

	// Decode the request
	err := reqJSON.Decode(&req)
	if err != nil {
		log.WithField("err", err).Error("Error parsing the request.")
		resp.Status = RESP_CODE_BADREQ
		resp.Message = RESP_CODE_BADREQ_T

		rw.WriteHeader(RESP_CODE_BADREQ)
		respJSON.Encode(resp)
		return
	}

	// Some types might not be strings so let's build a map for the params input
	params := map[string]string{}
	for key, value := range req.Params {
		switch v := value.(type) {
		case string:
			params[key] = v
		case bool:
			params[key] = strconv.FormatBool(v)
		case int:
			params[key] = strconv.Itoa(v)
		case float64:
			params[key] = strconv.FormatFloat(v, 'g', -1, 64)
		case float32:
			params[key] = strconv.FormatFloat(float64(v), 'g', -1, 32)

		}
	}

	// Build a job structure
	job := common.NewJob(req.ToolID, req.Name, user.Username, params)

	// Log the new job structure and parameters used for job creation
	logNewJob := log.WithFields(log.Fields{
		"jobID":     job.UUID,
		"jobName":   job.Name,
		"jobParams": common.CleanJobParamsForLogging(&job),
	})

	err = a.Q.AddJob(job)
	if err != nil {
		log.Error(err)
		resp.Status = RESP_CODE_BADREQ
		resp.Message = "An error occured when trying to create the job: " + err.Error()

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

	logNewJob.Debug("New job submitted via the API")
	log.WithFields(log.Fields{
		"uuid": job.UUID,
		"name": job.Name,
	}).Info("New job created.")
}

// Read an individual Job (GET - /api/jobs/{id})
func (a *AppController) ReadJob(rw http.ResponseWriter, r *http.Request) {
	// Response and Request structures
	var resp JobReadResp

	// JSON Encoder and Decoder
	respJSON := json.NewEncoder(rw)

	// Get the authorization header
	token := r.Header.Get("AuthorizationToken")

	if !a.T.CheckToken(token) {
		resp.Status = RESP_CODE_UNAUTHORIZED
		resp.Message = RESP_CODE_UNAUTHORIZED_T

		rw.WriteHeader(RESP_CODE_UNAUTHORIZED)
		respJSON.Encode(resp)

		log.WithField("token", token).Warn("An unknown user token attempted to read job data.")

		return
	}

	// Get the ID of the job we want
	jobid := mux.Vars(r)["id"]

	// Pull Job info from the Queue
	job := a.Q.JobInfo(jobid)
	logJob := log.WithFields(log.Fields{
		"jobID":     job.UUID,
		"jobName":   job.Name,
		"jobParams": common.CleanJobParamsForLogging(&job),
	})

	// Build the response structure
	resp.Status = RESP_CODE_OK
	resp.Message = RESP_CODE_OK_T
	resp.Job.ID = job.UUID
	resp.Job.Name = job.Name
	resp.Job.Status = job.Status
	resp.Job.ResourceID = job.ResAssigned
	resp.Job.Owner = job.Owner
	resp.Job.StartTime = job.StartTime
	resp.Job.ETC = job.ETC
	resp.Job.CrackedHashes = job.CrackedHashes
	resp.Job.TotalHashes = job.TotalHashes
	resp.Job.Progress = job.Progress
	resp.Job.Params = common.CleanJobParamsForLogging(&job)
	resp.Job.ToolID = job.ToolUUID
	resp.Job.PerformanceTitle = job.PerformanceTitle
	resp.Job.PerformanceData = job.PerformanceData
	resp.Job.OutputTitles = job.OutputTitles
	resp.Job.OutputData = job.OutputData

	rw.WriteHeader(RESP_CODE_OK)
	respJSON.Encode(resp)

	logJob.Debug("Job data pulled from Queue for API call")
	log.WithFields(log.Fields{
		"uuid": job.UUID,
		"name": job.Name,
	}).Info("Job detailed information gathered.")
}

// Update a job
func (a *AppController) UpdateJob(rw http.ResponseWriter, r *http.Request) {
	// Response and Request structures
	var req JobUpdateReq
	var resp JobUpdateResp

	// JSON Encoder and Decoder
	reqJSON := json.NewDecoder(r.Body)
	respJSON := json.NewEncoder(rw)

	// Get the authorization header
	token := r.Header.Get("AuthorizationToken")

	if !a.T.CheckToken(token) {
		resp.Status = RESP_CODE_UNAUTHORIZED
		resp.Message = RESP_CODE_UNAUTHORIZED_T

		rw.WriteHeader(RESP_CODE_UNAUTHORIZED)
		respJSON.Encode(resp)

		log.WithField("token", token).Warn("An unknown user token attempted to update job data.")

		return
	}

	// Check for standard user level at least
	user, _ := a.T.GetUser(token)
	if !user.Allowed(StandardUser) {
		resp.Status = RESP_CODE_UNAUTHORIZED
		resp.Message = RESP_CODE_UNAUTHORIZED_T

		rw.WriteHeader(RESP_CODE_UNAUTHORIZED)
		respJSON.Encode(resp)

		log.WithField("user", user).Warn("An unauthorized user attempted to update job data.")

		return
	}

	// Decode the request
	err := reqJSON.Decode(&req)
	if err != nil {
		resp.Status = RESP_CODE_BADREQ
		resp.Message = RESP_CODE_BADREQ_T

		rw.WriteHeader(RESP_CODE_BADREQ)
		respJSON.Encode(resp)

		log.Error("An error occured while trying to decode updated job data.")

		return
	}

	// Get the ID of the job we want
	jobid := mux.Vars(r)["id"]

	// Get the action requested
	switch req.Status {
	case "pause":
		// Pause the job
		err = a.Q.PauseJob(jobid)
		if err != nil {
			resp.Status = RESP_CODE_ERROR
			resp.Message = "Unable to pause the job: " + err.Error()

			rw.WriteHeader(RESP_CODE_ERROR)
			respJSON.Encode(resp)
			return
		}
	case "quit":
		// Stop the job
		err = a.Q.QuitJob(jobid)
		if err != nil {
			resp.Status = RESP_CODE_ERROR
			resp.Message = "Unable to stop the job: " + err.Error()

			rw.WriteHeader(RESP_CODE_ERROR)
			respJSON.Encode(resp)
			return
		}
	}

	// Now return everything is good and the job info
	j := a.Q.JobInfo(jobid)

	resp.Status = RESP_CODE_OK
	resp.Message = RESP_CODE_OK_T
	resp.Job.ID = j.UUID
	resp.Job.Name = j.Name
	resp.Job.Status = j.Status
	resp.Job.ResourceID = j.ResAssigned
	resp.Job.Owner = j.Owner
	resp.Job.StartTime = j.StartTime
	resp.Job.ETC = j.ETC
	resp.Job.CrackedHashes = j.CrackedHashes
	resp.Job.TotalHashes = j.TotalHashes
	resp.Job.Progress = j.Progress
	resp.Job.ToolID = j.ToolUUID

	rw.WriteHeader(RESP_CODE_OK)
	respJSON.Encode(resp)

	log.WithFields(log.Fields{
		"uuid":   j.UUID,
		"name":   j.Name,
		"status": j.Status,
	}).Info("Job information updated.")
}

func (a *AppController) DeleteJob(rw http.ResponseWriter, r *http.Request) {
	// Response and Request structures
	var resp JobDeleteResp

	// JSON Encoders and Decoders
	respJSON := json.NewEncoder(rw)

	// Get the authorization header
	token := r.Header.Get("AuthorizationToken")

	if !a.T.CheckToken(token) {
		resp.Status = RESP_CODE_UNAUTHORIZED
		resp.Message = RESP_CODE_UNAUTHORIZED_T

		rw.WriteHeader(RESP_CODE_UNAUTHORIZED)
		respJSON.Encode(resp)

		log.WithField("token", token).Warn("An unknown user token attempted to delete a job.")

		return
	}

	// Check for standard user level at least
	user, _ := a.T.GetUser(token)
	if !user.Allowed(StandardUser) {
		resp.Status = RESP_CODE_UNAUTHORIZED
		resp.Message = RESP_CODE_UNAUTHORIZED_T

		rw.WriteHeader(RESP_CODE_UNAUTHORIZED)
		respJSON.Encode(resp)

		log.WithField("username", user.Username).Warn("An unauthorized user attempted to delete a job.")

		return
	}

	// Get the ID of the job we want
	jobid := mux.Vars(r)["id"]

	// Remove the job
	err := a.Q.RemoveJob(jobid)
	if err != nil {
		resp.Status = RESP_CODE_ERROR
		resp.Message = "An error occured while trying to delete a job: " + err.Error()

		rw.WriteHeader(RESP_CODE_ERROR)
		respJSON.Encode(resp)

		log.WithFields(log.Fields{
			"jobid": jobid,
			"error": err.Error(),
		}).Error("An error occured while trying to delete a job.")

		return
	}

	// Job should now be removed, so return all OK
	resp.Status = RESP_CODE_OK
	resp.Message = RESP_CODE_OK_T

	rw.WriteHeader(RESP_CODE_OK)
	respJSON.Encode(resp)

	log.WithFields(log.Fields{
		"jobid": jobid,
	}).Info("Job deleted.")
}

// List Resource API function
func (a *AppController) ListResource(rw http.ResponseWriter, r *http.Request) {
	// Response and Request structure
	var resp ResListResp

	// JSON Encoders and Decoders
	respJSON := json.NewEncoder(rw)

	// Get the authorization header
	token := r.Header.Get("AuthorizationToken")

	if !a.T.CheckToken(token) {
		resp.Status = RESP_CODE_UNAUTHORIZED
		resp.Message = RESP_CODE_UNAUTHORIZED_T

		rw.WriteHeader(RESP_CODE_UNAUTHORIZED)
		respJSON.Encode(resp)

		log.WithField("token", token).Warn("An unknown user token attempted to list resources.")

		return
	}

	// Check for standard user level at least
	user, _ := a.T.GetUser(token)
	if !user.Allowed(StandardUser) {
		resp.Status = RESP_CODE_UNAUTHORIZED
		resp.Message = RESP_CODE_UNAUTHORIZED_T

		rw.WriteHeader(RESP_CODE_UNAUTHORIZED)
		respJSON.Encode(resp)

		log.WithField("username", user.Username).Warn("An unauthorized user attempted to list resources.")

		return
	}

	// First we need to loop through all resource managers
	for managerid, manager := range a.Q.AllResourceManagers() {
		//Then  we need to loop through all resources controlled by the manager
		for _, resourceid := range manager.GetManagedResources() {
			resource, params, err := manager.GetResource(resourceid)

			if err != nil {
				log.WithField("resourceid", resourceid).Error("Unable to find resource in queue when provided by manager while gathering API resource list.")
				continue
			}

			var outresource APIResource
			outresource.Manager = managerid
			outresource.ID = resourceid
			outresource.Name = resource.Name
			outresource.Status = resource.Status
			outresource.Address = resource.Address
			outresource.Params = params

			for _, t := range resource.Tools {
				outresource.Tools = append(outresource.Tools, APITool{t.UUID, t.Name, t.Version})
			}

			resp.Resources = append(resp.Resources, outresource)

			log.WithFields(log.Fields{
				"id":      resourceid,
				"name":    resource.Name,
				"addr":    resource.Address,
				"manager": managerid,
			}).Debug("Gathered resource information.")
		}
	}

	// Job should now be removed, so return all OK
	resp.Status = RESP_CODE_OK
	resp.Message = RESP_CODE_OK_T

	rw.WriteHeader(RESP_CODE_OK)
	respJSON.Encode(resp)

	log.Info("Listing of resources provided to API.")
}

func (a *AppController) CreateResource(rw http.ResponseWriter, r *http.Request) {
	// Response and Request structures
	var req ResCreateReq
	var resp ResCreateResp

	// JSON Encoders and Decoders
	reqJSON := json.NewDecoder(r.Body)
	respJSON := json.NewEncoder(rw)

	// Get the authorization header
	token := r.Header.Get("AuthorizationToken")

	if !a.T.CheckToken(token) {
		resp.Status = RESP_CODE_UNAUTHORIZED
		resp.Message = RESP_CODE_UNAUTHORIZED_T

		rw.WriteHeader(RESP_CODE_UNAUTHORIZED)
		respJSON.Encode(resp)

		log.WithField("token", token).Warn("An unknown user token attempted to connect to a resource.")

		return
	}

	// Check for Administrators user level at least
	user, _ := a.T.GetUser(token)
	if !user.Allowed(Administrator) {
		resp.Status = RESP_CODE_UNAUTHORIZED
		resp.Message = RESP_CODE_UNAUTHORIZED_T

		rw.WriteHeader(RESP_CODE_UNAUTHORIZED)
		respJSON.Encode(resp)

		log.WithField("username", user.Username).Warn("An unauthorized user attempted to connect to a resource.")

		return
	}

	// Decode the request
	err := reqJSON.Decode(&req)
	if err != nil {
		resp.Status = RESP_CODE_BADREQ
		resp.Message = RESP_CODE_BADREQ_T

		rw.WriteHeader(RESP_CODE_BADREQ)
		respJSON.Encode(resp)

		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("An error occured while trying to decode resource creation information.")

		return
	}

	//First we need to get the appropriate resource manager
	manager, ok := a.Q.GetResourceManager(req.Manager)
	//If that resource manager doesn't exist, return a not found error
	if !ok {
		resp.Status = RESP_CODE_NOTFOUND
		resp.Message = "That resource manager does not exist."

		rw.WriteHeader(RESP_CODE_NOTFOUND)
		respJSON.Encode(resp)

		log.WithFields(log.Fields{
			"manager": req.Manager,
		}).Warn("Unable to find requested resource manager.")

		return
	}

	// Now let's try and add the resource itself.
	err = manager.AddResource(req.Params)

	// If there was an error returned by the resource manager, let's go ahead and return an error to the user.
	if err != nil {
		resp.Status = RESP_CODE_ERROR
		resp.Message = "An error occured when trying to add the resource: " + err.Error()

		rw.WriteHeader(RESP_CODE_ERROR)
		respJSON.Encode(resp)

		log.WithFields(log.Fields{
			"error":   err.Error(),
			"manager": req.Manager,
		}).Error("An error occured adding a resource.")

		return
	}

	// At this point, the resource should be added, we can return success.
	resp.Status = RESP_CODE_OK
	resp.Message = RESP_CODE_OK_T

	rw.WriteHeader(RESP_CODE_OK)
	respJSON.Encode(resp)

	log.WithFields(log.Fields{
		"manager": req.Manager,
	}).Info("Resource successfully added.")
}

func (a *AppController) ReadResource(rw http.ResponseWriter, r *http.Request) {
	// Response and Request structures
	var resp ResReadResp

	// JSON Encoder and Decoder
	respJSON := json.NewEncoder(rw)

	// Get the authorization header
	token := r.Header.Get("AuthorizationToken")

	if !a.T.CheckToken(token) {
		resp.Status = RESP_CODE_UNAUTHORIZED
		resp.Message = RESP_CODE_UNAUTHORIZED_T

		rw.WriteHeader(RESP_CODE_UNAUTHORIZED)
		respJSON.Encode(resp)

		log.WithField("token", token).Warn("An unknown user token attempted to get resource information.")

		return
	}

	// Check for standard user level at least
	user, _ := a.T.GetUser(token)
	if !user.Allowed(StandardUser) {
		resp.Status = RESP_CODE_UNAUTHORIZED
		resp.Message = RESP_CODE_UNAUTHORIZED_T

		rw.WriteHeader(RESP_CODE_UNAUTHORIZED)
		respJSON.Encode(resp)

		log.WithField("username", user.Username).Warn("An unauthorized user attempted to get resource information.")

		return
	}

	// Get the resource ID and manager name from URL
	resID := mux.Vars(r)["id"]
	managerName := mux.Vars(r)["manager"]

	// Get the resource manager as defined in the URL
	manager, ok := a.Q.GetResourceManager(managerName)
	if !ok {
		resp.Status = RESP_CODE_NOTFOUND
		resp.Message = "The requested resource manager was not found."

		rw.WriteHeader(RESP_CODE_NOTFOUND)
		respJSON.Encode(resp)

		log.WithField("resource", resID).Warn("Resource manager details could not be found.")
	}

	// Get the resource
	resource, params, err := manager.GetResource(resID)
	if err != nil {
		resp.Status = RESP_CODE_NOTFOUND
		resp.Message = "The requested resource was not found."

		rw.WriteHeader(RESP_CODE_NOTFOUND)
		respJSON.Encode(resp)

		log.WithField("resource", resID).Warn("Resource details were requested and could not be found.")
	}

	// Found the resource so set it to the response
	resp.Resource.ID = resID
	resp.Resource.Name = resource.Name
	resp.Resource.Address = resource.Address
	resp.Resource.Status = resource.Status
	resp.Resource.Params = params
	resp.Resource.Manager = manager.SystemName()

	log.WithFields(log.Fields{
		"uuid":    resID,
		"name":    resource.Name,
		"addr":    resource.Address,
		"manager": manager.SystemName(),
	}).Debug("Gathered resource information.")

	for _, t := range resource.Tools {
		resp.Resource.Tools = append(resp.Resource.Tools, APITool{t.UUID, t.Name, t.Version})
		log.WithFields(log.Fields{
			"uuid": t.UUID,
			"name": t.Name,
			"ver":  t.Version,
		}).Debug("Tool configured on resource gathered.")
	}

	// TODO (mcatee): Add a check for no found resource and return correct status codes

	// Build good response
	resp.Status = RESP_CODE_OK
	resp.Message = RESP_CODE_OK_T

	rw.WriteHeader(RESP_CODE_OK)
	respJSON.Encode(resp)

	log.WithField("name", resp.Resource.Name).Info("Information gathered on resource.")
}

func (a *AppController) UpdateResource(rw http.ResponseWriter, r *http.Request) {
	// Response and Request structures
	var req ResUpdateReq
	var resp ResUpdateResp

	// JSON Encoder and Decoder
	reqJSON := json.NewDecoder(r.Body)
	respJSON := json.NewEncoder(rw)

	// Get the authorization header
	token := r.Header.Get("AuthorizationToken")

	if !a.T.CheckToken(token) {
		resp.Status = RESP_CODE_UNAUTHORIZED
		resp.Message = RESP_CODE_UNAUTHORIZED_T

		rw.WriteHeader(RESP_CODE_UNAUTHORIZED)
		respJSON.Encode(resp)

		log.WithField("token", token).Warn("An unknown user token attempted to update resource information.")

		return
	}

	// Check for Administrator user level at least
	user, _ := a.T.GetUser(token)
	if !user.Allowed(Administrator) {
		resp.Status = RESP_CODE_UNAUTHORIZED
		resp.Message = RESP_CODE_UNAUTHORIZED_T

		rw.WriteHeader(RESP_CODE_UNAUTHORIZED)
		respJSON.Encode(resp)

		log.WithField("user", user.Username).Warn("An unauthorized user attempted to update resource information.")

		return
	}

	// Decode the request
	err := reqJSON.Decode(&req)
	if err != nil {
		resp.Status = RESP_CODE_BADREQ
		resp.Message = RESP_CODE_BADREQ_T

		rw.WriteHeader(RESP_CODE_BADREQ)
		respJSON.Encode(resp)

		log.WithField("error", err.Error()).Error("An error occured while trying to decode resource update data.")

		return
	}

	// Get the resource ID
	resID := mux.Vars(r)["id"]
	managerName := req.Manager

	// Get the manager for the resource
	manager, manok := a.Q.GetResourceManager(managerName)

	//If that resource manager doesn't exist, return a not found error
	if !manok {
		resp.Status = RESP_CODE_NOTFOUND
		resp.Message = "That resource manager does not exist."

		rw.WriteHeader(RESP_CODE_NOTFOUND)
		respJSON.Encode(resp)

		log.WithFields(log.Fields{
			"manager":  managerName,
			"resource": resID,
		}).Warn("Unable to find requested manager to update resource.")

		return
	}

	switch req.Status {
	case common.STATUS_QUIT:
		log.WithFields(log.Fields{
			"manager":  manager.SystemName(),
			"resource": resID,
			"status":   req.Status,
		}).Info("Quiting resource status.")

		// Quit the resource
		err = manager.DeleteResource(resID)
		if err != nil {
			resp.Status = RESP_CODE_ERROR
			resp.Message = "An error occured while trying to quit that resource: " + err.Error()

			rw.WriteHeader(RESP_CODE_ERROR)
			respJSON.Encode(resp)

			log.WithFields(log.Fields{
				"manager":  manager.SystemName(),
				"error":    err.Error(),
				"resource": resID,
			}).Error("An error occured while trying to quit a resource.")

			return
		}
	case common.STATUS_PAUSED, common.STATUS_RUNNING:
		log.WithFields(log.Fields{
			"manager":  manager.SystemName(),
			"resource": resID,
			"status":   req.Status,
		}).Info("Updating resource status.")

		// Pause or resume the resource
		err = manager.UpdateResource(resID, req.Status, req.Params)
		if err != nil {
			resp.Status = RESP_CODE_ERROR
			resp.Message = "An error occured while trying to update that resource: " + err.Error()

			rw.WriteHeader(RESP_CODE_ERROR)
			respJSON.Encode(resp)

			log.WithFields(log.Fields{
				"manager":  manager.SystemName(),
				"error":    err.Error(),
				"resource": resID,
			}).Error("An error occured while trying to update a resource.")

			return
		}
	}

	// Build good response because we were able to get here
	resp.Status = RESP_CODE_OK
	resp.Message = RESP_CODE_OK_T

	rw.WriteHeader(RESP_CODE_OK)
	respJSON.Encode(resp)

	log.WithFields(log.Fields{
		"resource": resID,
		"status":   req.Status,
	}).Info("Resource updated.")
}

func (a *AppController) DeleteResources(rw http.ResponseWriter, r *http.Request) {
	// Response and Request structures
	var resp ResDeleteResp
	var req ResDeleteReq

	// JSON Encoder and Decoder
	respJSON := json.NewEncoder(rw)
	reqJSON := json.NewDecoder(r.Body)

	// Get the authorization header
	token := r.Header.Get("AuthorizationToken")

	if !a.T.CheckToken(token) {
		resp.Status = RESP_CODE_UNAUTHORIZED
		resp.Message = RESP_CODE_UNAUTHORIZED_T

		rw.WriteHeader(RESP_CODE_UNAUTHORIZED)
		respJSON.Encode(resp)

		log.WithField("token", token).Warn("An unknown user token attempted to delete a resource.")

		return
	}

	// Check for Administrator user level at least
	user, _ := a.T.GetUser(token)
	if !user.Allowed(Administrator) {
		resp.Status = RESP_CODE_UNAUTHORIZED
		resp.Message = RESP_CODE_UNAUTHORIZED_T

		rw.WriteHeader(RESP_CODE_UNAUTHORIZED)
		respJSON.Encode(resp)

		log.WithField("username", user.Username).Warn("An unauthorized user attempted to delete a resource.")

		return
	}

	// Decode the request
	err := reqJSON.Decode(&req)
	if err != nil {
		resp.Status = RESP_CODE_BADREQ
		resp.Message = RESP_CODE_BADREQ_T

		rw.WriteHeader(RESP_CODE_BADREQ)
		respJSON.Encode(resp)

		log.WithField("error", err.Error()).Error("An error occured while trying to decode resource delete data.")

		return
	}

	// Get the resource ID
	resID := mux.Vars(r)["id"]
	managerName := req.Manager

	// Get the manager for the resource
	manager, manok := a.Q.GetResourceManager(managerName)

	//If that resource manager doesn't exist, return a not found error
	if !manok {
		resp.Status = RESP_CODE_NOTFOUND
		resp.Message = "That resource manager does not exist."

		rw.WriteHeader(RESP_CODE_NOTFOUND)
		respJSON.Encode(resp)

		log.WithFields(log.Fields{
			"manager":  managerName,
			"resource": resID,
		}).Warn("Unable to find requested manager to update resource.")

		return
	}

	// Get the resource
	resource, _, err := manager.GetResource(resID)

	// If that resource doesn't exist, let's throw an error
	if err != nil {
		resp.Status = RESP_CODE_NOTFOUND
		resp.Message = "That resource does not exist."

		rw.WriteHeader(RESP_CODE_NOTFOUND)
		respJSON.Encode(resp)

		log.WithFields(log.Fields{
			"manager": manager.SystemName(),
			"addr":    resource.Address,
			"name":    resource.Name,
		}).Warn("Unable to find requested resource to update.")

		return
	}

	// Remove the resource
	err = manager.DeleteResource(resID)
	if err != nil {
		resp.Status = RESP_CODE_ERROR
		resp.Message = "An error occured while trying to delete that resource: " + err.Error()

		rw.WriteHeader(RESP_CODE_ERROR)
		respJSON.Encode(resp)

		log.WithFields(log.Fields{
			"error":    err.Error(),
			"resource": resID,
		}).Error("An error occured while trying to delete a resource.")

		return
	}

	// Build good response
	resp.Status = RESP_CODE_OK
	resp.Message = RESP_CODE_OK_T

	rw.WriteHeader(RESP_CODE_OK)
	respJSON.Encode(resp)

	log.WithField("resource", resID).Info("Resource disconnected.")
}

/*
	Handler for the PUT /api/queue function in our API that is used, for now to
	handle updates to the order of jobs in the queue.
*/
func (a *AppController) ReorderQueue(rw http.ResponseWriter, r *http.Request) {
	// Structurs to hold our request and response from Negroni, see api_struct.go
	var req QueueUpdateReq
	var resp QueueUpdateResp

	// A decoder to take the JSON information passed by the API and return it
	reqJSON := json.NewDecoder(r.Body)
	// An encoder to take our response and give it back to the user
	respJSON := json.NewEncoder(rw)

	// First, we handle authentication through the header
	token := r.Header.Get("AuthorizationToken")
	if !a.T.CheckToken(token) {
		//If the token is unknown, send back an unauthenticated message
		resp.Status = RESP_CODE_UNAUTHORIZED
		resp.Message = RESP_CODE_UNAUTHORIZED_T

		rw.WriteHeader(RESP_CODE_UNAUTHORIZED)
		respJSON.Encode(resp)

		log.WithField("token", token).Warn("An unknown user token attempted to reorder the queue.")

		return
	}

	// Let's then check to make sure the user has the right group, in this case standard
	user, _ := a.T.GetUser(token)
	if !user.Allowed(StandardUser) {
		//If not, send back the proper response.
		resp.Status = RESP_CODE_UNAUTHORIZED
		resp.Message = RESP_CODE_UNAUTHORIZED_T

		rw.WriteHeader(RESP_CODE_UNAUTHORIZED)
		respJSON.Encode(resp)

		log.WithField("username", user.Username).Warn("An unauthorized user attempted to reorder the queue.")

		return
	}

	// Decode the request data that we recieved into our struct
	err := reqJSON.Decode(&req)
	if err != nil {
		// If there is an error, let the API know via HTTP
		resp.Status = RESP_CODE_BADREQ
		resp.Message = RESP_CODE_BADREQ_T

		rw.WriteHeader(RESP_CODE_BADREQ)
		respJSON.Encode(resp)

		log.WithField("error", err.Error()).Error("An error occured while trying to decode queue update data.")

		return
	}

	// Let's try and actually reorder the stack
	err = a.Q.StackReorder(req.JobOrder)
	if err != nil {
		//If there was an error, send the code to the API
		resp.Status = RESP_CODE_ERROR
		resp.Message = err.Error()

		rw.WriteHeader(RESP_CODE_ERROR)
		respJSON.Encode(resp)

		log.WithField("error", err.Error()).Error("An error occured while trying to update the queue order.")

		return
	}

	// Finally, we did it successfully!
	log.Info("Queue reodered successfully")
}
