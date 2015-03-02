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
}

// Logout Response Structure
type LogoutResp struct {
	Status  int    `json:status`
	Message string `json:message`
}

// Tool API structure
type APITool struct {
	Name    string `json:name`
	Version string `json:version`
}

type APITools map[string]APITool

// Tools List Response Structure
type ToolsResp struct {
	Status  int      `json:status`
	Message string   `json:message`
	Tools   APITools `json:tools`
}

// Get Tools structures
type ToolsGetResp struct {
	Status  int             `json:status`
	Message string          `json:message`
	Name    string          `json:name`
	Version string          `json:version`
	Form    json.RawMessage `json:form`
	Schema  json.RawMessage `json:schema`
}

// API Jobs structure
type APIJobs struct {
	JobID         string    `json:jobid`
	Name          string    `json:name`
	Status        string    `json:status`
	Owner         string    `json:owner`
	StartTime     time.Time `json:starttime`
	CrackedHashes int64     `json:crackedhashes`
	TotalHashes   int64     `json:totalhashes`
	Percentage    int       `json:percentage`
}

// Get Jobs structure
type GetJobsResp struct {
	Status  int       `json:status`
	Message string    `json:message`
	Jobs    []APIJobs `json:jobs`
}

// Create Jobs request
type JobCreateReq struct {
	Token  string            `json:token`
	ToolID string            `json:toolid`
	Name   string            `json:name`
	Params map[string]string `json:params`
}

// Create Job response
type JobCreateResp struct {
	Status  int    `json:status`
	Message string `json:message`
	JobID   string `json:jobid`
}
