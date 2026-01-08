package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// Error codes
const (
	CodeUnknown           = 0
	CodeInvalidArgument   = 400
	CodeUnauthenticated   = 401
	CodePermissionDenied  = 403
	CodeNotFound          = 404
	CodeConflict          = 409
	CodeTooManyRequests   = 429
	CodeInternal          = 500
	CodeUnavailable       = 503
	CodeTimeout           = 504

	// Business error codes (1000+)
	CodeInsufficientBalance = 1001
	CodeQuotaExceeded       = 1002
	CodeProviderError       = 1003
	CodeModelNotSupported   = 1004
	CodeRateLimited         = 1005
	CodeStreamError         = 1006
)

// Error represents a structured error
type Error struct {
	Code    int                    `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
	Cause   error                  `json:"-"`
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns the underlying error
func (e *Error) Unwrap() error {
	return e.Cause
}

// WithDetails adds details to the error
func (e *Error) WithDetails(key string, value interface{}) *Error {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// WithCause adds a cause to the error
func (e *Error) WithCause(cause error) *Error {
	e.Cause = cause
	return e
}

// HTTPStatus returns the HTTP status code for this error
func (e *Error) HTTPStatus() int {
	switch e.Code {
	case CodeInvalidArgument:
		return http.StatusBadRequest
	case CodeUnauthenticated:
		return http.StatusUnauthorized
	case CodePermissionDenied:
		return http.StatusForbidden
	case CodeNotFound:
		return http.StatusNotFound
	case CodeConflict:
		return http.StatusConflict
	case CodeTooManyRequests, CodeRateLimited:
		return http.StatusTooManyRequests
	case CodeInternal:
		return http.StatusInternalServerError
	case CodeUnavailable:
		return http.StatusServiceUnavailable
	case CodeTimeout:
		return http.StatusGatewayTimeout
	case CodeInsufficientBalance, CodeQuotaExceeded:
		return http.StatusPaymentRequired
	case CodeProviderError, CodeModelNotSupported, CodeStreamError:
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}

// New creates a new error
func New(code int, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// Newf creates a new error with formatted message
func Newf(code int, format string, args ...interface{}) *Error {
	return &Error{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	}
}

// Wrap wraps an error with a message
func Wrap(err error, message string) *Error {
	if err == nil {
		return nil
	}
	if e, ok := err.(*Error); ok {
		return &Error{
			Code:    e.Code,
			Message: message,
			Cause:   err,
		}
	}
	return &Error{
		Code:    CodeInternal,
		Message: message,
		Cause:   err,
	}
}

// Wrapf wraps an error with a formatted message
func Wrapf(err error, format string, args ...interface{}) *Error {
	return Wrap(err, fmt.Sprintf(format, args...))
}

// FromError extracts an Error from an error
func FromError(err error) *Error {
	if err == nil {
		return nil
	}
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return &Error{
		Code:    CodeUnknown,
		Message: err.Error(),
		Cause:   err,
	}
}

// Is checks if an error matches a code
func Is(err error, code int) bool {
	if err == nil {
		return false
	}
	e := FromError(err)
	return e.Code == code
}

// Common errors

// ErrInvalidArgument returns an invalid argument error
func ErrInvalidArgument(message string) *Error {
	return New(CodeInvalidArgument, message)
}

// ErrUnauthenticated returns an unauthenticated error
func ErrUnauthenticated(message string) *Error {
	return New(CodeUnauthenticated, message)
}

// ErrPermissionDenied returns a permission denied error
func ErrPermissionDenied(message string) *Error {
	return New(CodePermissionDenied, message)
}

// ErrNotFound returns a not found error
func ErrNotFound(message string) *Error {
	return New(CodeNotFound, message)
}

// ErrConflict returns a conflict error
func ErrConflict(message string) *Error {
	return New(CodeConflict, message)
}

// ErrInternal returns an internal error
func ErrInternal(message string) *Error {
	return New(CodeInternal, message)
}

// ErrUnavailable returns an unavailable error
func ErrUnavailable(message string) *Error {
	return New(CodeUnavailable, message)
}

// ErrTimeout returns a timeout error
func ErrTimeout(message string) *Error {
	return New(CodeTimeout, message)
}

// Business errors

// ErrInsufficientBalance returns an insufficient balance error
func ErrInsufficientBalance(userID string, required, available float64) *Error {
	return New(CodeInsufficientBalance, "Insufficient balance").
		WithDetails("user_id", userID).
		WithDetails("required", required).
		WithDetails("available", available)
}

// ErrQuotaExceeded returns a quota exceeded error
func ErrQuotaExceeded(userID string, quota, used float64) *Error {
	return New(CodeQuotaExceeded, "Quota exceeded").
		WithDetails("user_id", userID).
		WithDetails("quota", quota).
		WithDetails("used", used)
}

// ErrProviderError returns a provider error
func ErrProviderError(provider string, cause error) *Error {
	return New(CodeProviderError, "Provider error").
		WithDetails("provider", provider).
		WithCause(cause)
}

// ErrModelNotSupported returns a model not supported error
func ErrModelNotSupported(model, provider string) *Error {
	return New(CodeModelNotSupported, "Model not supported").
		WithDetails("model", model).
		WithDetails("provider", provider)
}

// ErrRateLimited returns a rate limited error
func ErrRateLimited(retryAfter int) *Error {
	return New(CodeRateLimited, "Rate limited").
		WithDetails("retry_after", retryAfter)
}
