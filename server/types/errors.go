package types

import "net/http"

type apiError struct {
	Status  int    `json:"-"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *apiError) Error() string { return e.Message }

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
