package errors

import (
	"net/http"
	"strings"
)

// Standard for Error reponses to the client.
type ErrorResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

// Error is required by the error interface.
func (e ErrorResponse) Error() string {
	return e.Message
}

// StatusCode is required by routing.HTTPError interface.
func (e ErrorResponse) StatusCode() int {
	return e.Status
}

// InternalServerError creates a new error response representing an internal server error (HTTP 500)
func InternalServerError(msg string) ErrorResponse {
	if msg == "" {
		msg = "We encountered an error while processing your request."
	}
	return ErrorResponse{
		Status:  http.StatusInternalServerError,
		Message: msg,
	}
}

// NotFound creates a new error response representing a resource-not-found error (HTTP 404)
func NotFound(msg string) ErrorResponse {
	if msg == "" {
		msg = "The requested resource was not found."
	}
	return ErrorResponse{
		Status:  http.StatusNotFound,
		Message: msg,
	}
}

// Unauthorized creates a new error response representing an authentication/authorization failure (HTTP 401)
func Unauthorized(msg string) ErrorResponse {
	if msg == "" {
		msg = "You are not authenticated to perform the requested action."
	}
	return ErrorResponse{
		Status:  http.StatusUnauthorized,
		Message: msg,
	}
}

// Forbidden creates a new error response representing an authorization failure (HTTP 403)
func Forbidden(msg string) ErrorResponse {
	if msg == "" {
		msg = "You are not authorized to perform the requested action."
	}
	return ErrorResponse{
		Status:  http.StatusForbidden,
		Message: msg,
	}
}

// BadRequest creates a new error response representing a bad request (HTTP 400)
func BadRequest(msg string) ErrorResponse {
	if msg == "" {
		msg = "Your request is in a bad format."
	}
	return ErrorResponse{
		Status:  http.StatusBadRequest,
		Message: msg,
	}
}

// Standard for Validation-error responses to the client.
type validationErrorResponse struct {
	Param   string `json:"param"`   // Parameter or Field
	Message string `json:"message"` // Issue in Field
}

// Scans through set of validation errors found by govalidator,
// Generates a slice of serializable validationErrorResponse.
func GenerateValidationErrorResponse(errs []error) []validationErrorResponse {
	// govalidator returns array of errors -> Param:Message
	// We split the error from ":"
	resp := []validationErrorResponse{}
	for _, err := range errs {
		e := strings.Split(err.Error(), ":")
		resp = append(
			resp, validationErrorResponse{
				Param:   e[0],
				Message: strings.TrimSpace(e[1]),
			},
		)
	}
	return resp
}
