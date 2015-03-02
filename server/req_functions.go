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

// List Tools endpoint (GET - /tools)
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

// Get Tool Endpoint
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
