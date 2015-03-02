package main

import (
	"encoding/json"
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
