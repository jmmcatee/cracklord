package main

import ()

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

// Logout Request Structure
type LogoutReq struct {
	Token
}

// Logout Response Structure
type LogoutResp struct {
	Status  int    `json:status`
	Message string `json:message`
}
