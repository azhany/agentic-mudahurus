// Package httpx holds shared HTTP concerns: a consistent JSON error envelope,
// middleware, and helpers. No handler leaks stack traces to clients (NFR security).
package httpx

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// ErrorBody is the consistent error shape returned to clients.
type ErrorBody struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

// Envelope wraps every error response.
type Envelope struct {
	Error ErrorBody `json:"error"`
}

// APIError is a domain error carrying an HTTP status + stable code.
type APIError struct {
	Status  int
	Code    string
	Message string
	Fields  map[string]string
}

func (e *APIError) Error() string { return e.Message }

// Constructors for common cases.
func BadRequest(msg string) *APIError      { return &APIError{http.StatusBadRequest, "bad_request", msg, nil} }
func Unauthorized(msg string) *APIError    { return &APIError{http.StatusUnauthorized, "unauthorized", msg, nil} }
func Forbidden(msg string) *APIError        { return &APIError{http.StatusForbidden, "forbidden", msg, nil} }
func NotFound(msg string) *APIError          { return &APIError{http.StatusNotFound, "not_found", msg, nil} }
func Conflict(msg string) *APIError           { return &APIError{http.StatusConflict, "conflict", msg, nil} }
func TooMany(msg string) *APIError             { return &APIError{http.StatusTooManyRequests, "rate_limited", msg, nil} }
func Internal(msg string) *APIError             { return &APIError{http.StatusInternalServerError, "internal", msg, nil} }

// Validation builds a 422 with per-field messages.
func Validation(fields map[string]string) *APIError {
	return &APIError{http.StatusUnprocessableEntity, "validation_failed", "validation failed", fields}
}

// ErrorHandler is the central Echo error handler producing the envelope.
func ErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}
	var ae *APIError
	switch e := err.(type) {
	case *APIError:
		ae = e
	case *echo.HTTPError:
		msg := http.StatusText(e.Code)
		if s, ok := e.Message.(string); ok {
			msg = s
		}
		ae = &APIError{Status: e.Code, Code: "http_error", Message: msg}
	default:
		// Never leak internal error text to clients.
		ae = Internal("an unexpected error occurred")
	}
	_ = c.JSON(ae.Status, Envelope{Error: ErrorBody{Code: ae.Code, Message: ae.Message, Fields: ae.Fields}})
}
