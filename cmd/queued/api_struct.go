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
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

type APIToolDetail struct {
	ID      string           `json:"id"`
	Name    string           `json:"name"`
	Version string           `json:"version"`
	Form    *json.RawMessage `json:"form"`
	Schema  *json.RawMessage `json:"schema"`
}

// Tools List Response Structure
type ToolsResp struct {
	Status  int       `json:"status"`
	Message string    `json:"message"`
	Tools   []APITool `json:"tools"`
}

// Get Tools structures
type ToolsGetResp struct {
	Status  int           `json:"status"`
	Message string        `json:"message"`
	Tool    APIToolDetail `json:"tool"`
}

// Resource Manager API structure
type APIResourceManager struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type APIResourceManagerDetail struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Form        string `json:"form"`
	Schema      string `json:"schema"`
}

// Tools List Response Structure
type ResourceManagersResp struct {
	Status           int                  `json:"status"`
	Message          string               `json:"message"`
	ResourceManagers []APIResourceManager `json:"tools"`
}

type ResourceManagerGetResp struct {
	Status          int                      `json:"status"`
	Message         string                   `json:"message"`
	ResourceManager APIResourceManagerDetail `json:"resourcemanager"`
}

// API Jobs structure
type APIJob struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Status        string    `json:"status"`
	ResourceID    string    `json:"resourceid"`
	Owner         string    `json:"owner"`
	StartTime     time.Time `json:"starttime"`
	ETC           string    `json:"etc"`
	CrackedHashes int64     `json:"crackedhashes"`
	TotalHashes   int64     `json:"totalhashes"`
	Progress      float64   `json:"progress"`
	ToolID        string    `json:"toolid"`
}

type APIJobDetail struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Status           string            `json:"status"`
	ResourceID       string            `json:"resourceid"`
	Owner            string            `json:"owner"`
	StartTime        time.Time         `json:"starttime"`
	ETC              string            `json:"etc"`
	CrackedHashes    int64             `json:"crackedhashes"`
	TotalHashes      int64             `json:"totalhashes"`
	Progress         float64           `json:"progress"`
	Params           map[string]string `json:"params"`
	ToolID           string            `json:"toolid"`
	PerformanceTitle string            `json:"performancetitle"`
	PerformanceData  map[string]string `json:"performancedata"`
	OutputTitles     []string          `json:"outputtitles"`
	OutputData       [][]string        `json:"outputdata"`
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
	Status  int          `json:"status"`
	Message string       `json:"message"`
	Job     APIJobDetail `json:"job"`
}

// Update Job Request
type JobUpdateReq struct {
	APIJob
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
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Address string            `json:"address"`
	Manager string            `json:"manager"`
	Params  map[string]string `json:"params"`
	Status  string            `json:"status"`
	Tools   []APITool         `json:"tools"`
}

// List resource structs
type ResListResp struct {
	Status    int           `json:"status"`
	Message   string        `json:"message"`
	Resources []APIResource `json:"resources"`
}

// Create resource structs
type ResCreateReq struct {
	Manager string            `json:"manager"`
	Params  map[string]string `json:"params"`
}

type ResCreateResp struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

// Read a resource struct
type ResReadResp struct {
	Status   int         `json:"status"`
	Message  string      `json:"message"`
	Resource APIResource `json:"resource"`
}

// Update a resource struct
type ResUpdateReq struct {
	ID      string            `json:"id"`
	Manager string            `json:"manager"`
	Params  map[string]string `json:"params"`
	Status  string            `json:"status"`
	Tools   []APITool         `json:"tools"`
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

type QueueUpdateReq struct {
	JobOrder []string `json:"joborder"`
}

type QueueUpdateResp struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}
