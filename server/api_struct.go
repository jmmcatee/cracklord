package main

import (
	"encoding/json"
	"time"
)

// Login Request Structure
type LoginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Login Response Structure
type LoginResp struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Token   string `json:"token"`
	Role    string `json:"role"`
}

// Logout Response Structure
type LogoutResp struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

// Tool API structure
type APITool struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type APITools map[string]APITool

// Tools List Response Structure
type ToolsResp struct {
	Status  int      `json:"status"`
	Message string   `json:"message"`
	Tools   APITools `json:"tools"`
}

// Get Tools structures
type ToolsGetResp struct {
	Status  int              `json:"status"`
	Message string           `json:"message"`
	Name    string           `json:"name"`
	Version string           `json:"version"`
	Form    *json.RawMessage `json:"form"`
	Schema  *json.RawMessage `json:"schema"`
}

// API Jobs structure
type APIJob struct {
	JobID         string    `json:"jobid"`
	Name          string    `json:"name"`
	JobStatus     string    `json:"jobstatus"`
	Owner         string    `json:"owner"`
	StartTime     time.Time `json:"starttime"`
	CrackedHashes int64     `json:"crackedhashes"`
	TotalHashes   int64     `json:"totalhashes"`
	Percentage    int       `json:"percentage"`
}

// Get Jobs structure
type GetJobsResp struct {
	Status  int      `json:"status"`
	Message string   `json:"message"`
	Jobs    []APIJob `json:"jobs"`
}

// Create Jobs request
type JobCreateReq struct {
	ToolID string            `json:"toolid"`
	Name   string            `json:"name"`
	Params map[string]string `json:"params"`
}

// Create Job response
type JobCreateResp struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	JobID   string `json:"jobid"`
}

// Read Job resposne
type JobReadResp struct {
	Status           int               `json:"status"`
	Message          string            `json:"message"`
	Performance      map[string]string `json:"performance"`
	PerformanceTitle string            `json:"performancetitle"`
	Output           map[string]string `json:"output"`
	APIJob
}

// Update Job Request
type JobUpdateReq struct {
	Action string `json:"action"`
}

// Update Job Response
type JobUpdateResp struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Job     APIJob `json:"job"`
}

// Delete Job response
type JobDeleteResp struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

// Resource API structure
type APIResource struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Address string   `json:"address"`
	Status  string   `json:"status"`
	Tools   APITools `json:"tools"`
}

// List resource structs
type ResListResp struct {
	Status    int           `json:"status"`
	Message   string        `json:"message"`
	Resources []APIResource `json:"resources"`
}

// Create resource structs
type ResCreateReq struct {
	Key     string `json:"key"`
	Name    string `json:"name"`
	Address string `json:"address"`
}

type ResCreateResp struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

// Read a resource struct
type ResReadReq struct {
	Token string `json:"token"`
}

type ResReadResp struct {
	Status   int         `json:"status"`
	Message  string      `json:"message"`
	Resource APIResource `json:"resource"`
}

// Update a resource struct
type ResUpdateReq struct {
	Status string `json:"status"`
}

type ResUpdateResp struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

// Delete a resource struct
type ResDeleteResp struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}
