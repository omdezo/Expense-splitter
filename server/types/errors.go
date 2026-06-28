package types

import "net/http"

// apiError is a JSON-serializable error response. Status is the HTTP status to
// send and is omitted from the body. The type is unexported so values can only
// be built through the constructors below; callers use the APIError alias.
type apiError struct {
	Status  int    `json:"-"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Error implements the error interface so an APIError can be returned as a plain
// error where convenient.
func (e *apiError) Error() string { return e.Message }

// APIError is the public handle for an application error. It is always a pointer
// (the underlying struct is unexported), so handlers write
// c.JSON(apiErr.Status, apiErr) and service signatures read (T, types.APIError)
// with no leading '*'.
type APIError = *apiError

func NewAPIError(status int, code, message string) APIError {
	return &apiError{Status: status, Code: code, Message: message}
}

func NewServerError() APIError {
	return NewAPIError(http.StatusInternalServerError, "internal_error", "internal server error")
}

func NewBadRequestError(message string) APIError {
	return NewAPIError(http.StatusBadRequest, "bad_request", message)
}

func NewUnauthorizedError(message string) APIError {
	return NewAPIError(http.StatusUnauthorized, "unauthorized", message)
}

func NewForbiddenError(message string) APIError {
	return NewAPIError(http.StatusForbidden, "forbidden", message)
}

func NewNotFoundError(message string) APIError {
	return NewAPIError(http.StatusNotFound, "not_found", message)
}

func NewConflictError(message string) APIError {
	return NewAPIError(http.StatusConflict, "conflict", message)
}

func NewServiceUnavailableError(message string) APIError {
	return NewAPIError(http.StatusServiceUnavailable, "service_unavailable", message)
}
