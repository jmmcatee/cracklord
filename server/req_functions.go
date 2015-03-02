package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/gorilla/mux"
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
	respJSON.Encode(resp)
}

// Logout endpoint (POST - /api/logout)
func (a *AppController) Logout(rw http.ResponseWriter, r *http.Request) {
	// Response and Request structures
	var req = LogoutReq{}
	var resp = LogoutResp{}

	// Build the JSON Decoder
	respJSON := json.NewEncoder(rw)

	// Get the token from the Query string
	req.Token = r.URL.Query().Get("token")

	a.T.RemoveToken(req.Token)

	resp.Status = RESP_CODE_OK
	resp.Message = RESP_CODE_OK_T

	respJSON.Encode(resp)
}
