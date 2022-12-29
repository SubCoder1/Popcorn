package errors

import (
	"net/http"
	"strings"
)

// Standard for Error reponses to the client.
type ErrorResponse struct {
	Status  int         `json:"status"`
	Message string      `json:"message"`
	Details interface{} `json:"details"`
}

// Error is required by the error interface.
func (e ErrorResponse) Error() string {
	return e.Message
}

// Get the StatusCode of the error.
func (e ErrorResponse) StatusCode() int {
	return e.Status
}

// Replicates the New method of default errors package.
func New(err string) error {
	return ErrorResponse{
		Message: err,
	}
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
type validationError struct {
	Param   string `json:"param"`   // Parameter or Field
	Message string `json:"message"` // Issue in Field
}

// Captures multiple validation issues and sends it as a response in one go.
// Use-case of this would be bunch of validation issues caught in a form.
type ValidationErrorResponse struct {
	Response []validationError `json:"errors"`
}

// Scans through set of validation errors found by govalidator,
// Generates a slice of serializable validationErrorResponse.
func GenerateValidationErrorResponse(errs []error) ErrorResponse {
	// govalidator returns array of errors in -> Param:Message format
	// We split the error from ":"
	resp := []validationError{}
	for _, err := range errs {
		e := strings.Split(err.Error(), ":")
		resp = append(
			resp, validationError{
				Param:   e[0],
				Message: strings.TrimSpace(e[1]),
			},
		)
	}
	return ErrorResponse{
		Status:  http.StatusBadRequest,
		Message: "Data validation error",
		Details: ValidationErrorResponse{Response: resp},
	}
}
