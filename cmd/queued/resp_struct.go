package main

import ()

const (
	// Integer Status Codes
	RESP_CODE_OK           = 200
	RESP_CODE_CREATED      = 201
	RESP_CODE_NOCONTENT    = 204
	RESP_CODE_NOTMODIFIED  = 304
	RESP_CODE_BADREQ       = 400
	RESP_CODE_UNAUTHORIZED = 401
	RESP_CODE_FORBIDDEN    = 403
	RESP_CODE_NOTFOUND     = 404
	RESP_CODE_CONFLICT     = 409
	RESP_CODE_ERROR        = 500

	// Text Status Codes
	RESP_CODE_OK_T           = "OK"
	RESP_CODE_CREATED_T      = "Created"
	RESP_CODE_NOCONTENT_T    = "No Content"
	RESP_CODE_NOTMODIFIED_T  = "Not Modified"
	RESP_CODE_BADREQ_T       = "The system could not process your request, the expected data was incorrect."
	RESP_CODE_UNAUTHORIZED_T = "You are not authorized to perform that action."
	RESP_CODE_FORBIDDEN_T    = "You are not authorized to perform that action."
	RESP_CODE_NOTFOUND_T     = "Not Found"
	RESP_CODE_CONFLICT_T     = "Conflict"
	RESP_CODE_ERROR_T        = "An internal server error occured, please refer to the server log."
)

// // Response Code Interface
// type Response interface {
// 	Code() int
// 	Message() string
// }

// // OK Respose structure
// type OKResp struct{}

// func (c OKResp) Code() int {
// 	return RESP_CODE_OK
// }

// func (c OKResp) Message() string {
// 	return RESP_CODE_OK_T
// }

// // Created Respose structure
// type CreatedResp struct{}

// func (c CreatedResp) Code() int {
// 	return RESP_CODE_CREATED
// }

// func (c CreatedResp) Message() string {
// 	return RESP_CODE_CREATED_T
// }

// // No Content Respose structure
// type NoContentResp struct{}

// func (c NoContentResp) Code() int {
// 	return RESP_CODE_NOCONTENT
// }

// func (c NoContentResp) Message() string {
// 	return RESP_CODE_NOCONTENT_T
// }

// // Not Modified Response structure
// type NotModifiedResp struct{}

// func (c NotModifiedResp) Code() int {
// 	return RESP_CODE_NOTMODIFIED
// }

// func (c NotModifiedResp) Message() string {
// 	return RESP_CODE_NOTMODIFIED_T
// }

// // Bad Request Response structure
// type BadReqResp struct{}

// func (c BadReqResp) Code() int {
// 	return RESP_CODE_BADREQ
// }

// func (c BadReqResp) Message() string {
// 	return RESP_CODE_BADREQ_T
// }

// // Unauthorized Response structure
// type UnauthResp struct{}

// func (c UnauthResp) Code() int {
// 	return RESP_CODE_UNAUTHORIZED
// }

// func (c UnauthResp) Message() string {
// 	return RESP_CODE_UNAUTHORIZED_T
// }

// // Forbidden Response structure
// type ForbiddenResp struct{}

// func (c ForbiddenResp) Code() int {
// 	return RESP_CODE_FORBIDDEN
// }

// func (c ForbiddenResp) Message() string {
// 	return RESP_CODE_FORBIDDEN_T
// }

// // Not Found Response structure
// type NotFoundResp struct{}

// func (c NotFoundResp) Code() int {
// 	return RESP_CODE_NOTFOUND
// }

// func (c NotFoundResp) Message() string {
// 	return RESP_CODE_NOTFOUND_T
// }

// // Conflict Response structure
// type ConflictResp struct{}

// func (c ConflictResp) Code() int {
// 	return RESP_CODE_CONFLICT
// }

// func (c ConflictResp) Message() string {
// 	return RESP_CODE_CONFLICT_T
// }

// // Internal Error Response structure
// type IntErrorResp struct{}

// func (c IntErrorResp) Code() int {
// 	return RESP_CODE_ERROR
// }

// func (c IntErrorResp) Message() string {
// 	return RESP_CODE_ERROR_T
// }
