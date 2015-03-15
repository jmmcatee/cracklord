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
type APIJob struct {
	JobID         string    `json:jobid`
	Name          string    `json:name`
	JobStatus     string    `json:jobstatus`
	Owner         string    `json:owner`
	StartTime     time.Time `json:starttime`
	CrackedHashes int64     `json:crackedhashes`
	TotalHashes   int64     `json:totalhashes`
	Percentage    int       `json:percentage`
}

// Get Jobs structure
type GetJobsResp struct {
	Status  int      `json:status`
	Message string   `json:message`
	Jobs    []APIJob `json:jobs`
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

// Read Job resposne
type JobReadResp struct {
	Status           int               `json:status`
	Message          string            `json:message`
	Performance      map[string]string `json:performance`
	PerformanceTitle string            `json:performancetitle`
	Output           map[string]string `json:output`
	APIJob
}

// Update Job Request
type JobUpdateReq struct {
	Token  string `json:token`
	Action string `json:action`
}

// Update Job Response
type JobUpdateResp struct {
	Status  int    `json:status`
	Message string `json:message`
	Job     APIJob `json:job`
}

// Delete Job request
type JobDeleteReq struct {
	Token string `json:token`
}

// Delete Job response
type JobDeleteResp struct {
	Status  int    `json:status`
	Message string `json:message`
}

// Resource API Structure
type APIResource struct {
	ResourceID string   `json:resourceid`
	Status     string   `json:status`
	Tools      APITools `json:tools`
}

// Get Resources
type ResourceListResp struct {
	Status    int           `json:status`
	Message   string        `json:message`
	Resources []APIResource `json:resources`
}

// Create Resource structures
type ResourceCreateReq struct {
	Token string `json:token`
	Key   string `json:key`
	Host  string `json:host`
}

type ResourceCreateResp struct {
	Status  int    `json:status`
	Message string `json:message`
}

// Read Resource structures
type ResourceReadReq struct {
	Token string `json:token`
}

type ResourceReadResp struct {
	Status         int      `json:status`
	Message        string   `json:message`
	ResourceStatus string   `json:resource_status`
	Hardware       []string `json:hardware`
	Tools          APITools `json:tools`
}

// Update Resource structures
type ResourceUpdateReq struct {
	Token  string `json:token`
	Action string `json:action`
}

type ResourceUpdateResp struct {
	Status  int    `json:status`
	Message string `json:message`
}

// Delete Resource structures
type ResourceDeleteReq struct {
	Token string `json:token`
}

type ResourceDeleteResp struct {
	Status  int    `json:status`
	Message string `json:message`
}

// Queue Update structures
type QueueUpdateReq struct {
	Token    string   `json:token`
	JobOrder []string `json:joborder`
}

type QueueUpdateResp struct {
	Status  int    `json:status`
	Message string `json:message`
}
