package errors

import (
	"errors"
	"net/http"
	"testing"
)

func TestError_Error(t *testing.T) {
	err := New(CodeNotFound, "resource not found")
	if err.Error() != "resource not found" {
		t.Errorf("Expected 'resource not found', got '%s'", err.Error())
	}

	cause := errors.New("underlying error")
	errWithCause := New(CodeInternal, "internal error").WithCause(cause)
	expected := "internal error: underlying error"
	if errWithCause.Error() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, errWithCause.Error())
	}
}

func TestError_Unwrap(t *testing.T) {
	cause := errors.New("original error")
	err := New(CodeInternal, "wrapper").WithCause(cause)

	unwrapped := err.Unwrap()
	if unwrapped != cause {
		t.Errorf("Expected unwrapped error to be the cause")
	}
}

func TestError_WithDetails(t *testing.T) {
	err := New(CodeInvalidArgument, "invalid request")
	err.WithDetails("field", "email").WithDetails("reason", "format invalid")

	if err.Details["field"] != "email" {
		t.Errorf("Expected field to be 'email', got '%v'", err.Details["field"])
	}
	if err.Details["reason"] != "format invalid" {
		t.Errorf("Expected reason to be 'format invalid', got '%v'", err.Details["reason"])
	}
}

func TestError_HTTPStatus(t *testing.T) {
	tests := []struct {
		code     int
		expected int
	}{
		{CodeInvalidArgument, http.StatusBadRequest},
		{CodeUnauthenticated, http.StatusUnauthorized},
		{CodePermissionDenied, http.StatusForbidden},
		{CodeNotFound, http.StatusNotFound},
		{CodeConflict, http.StatusConflict},
		{CodeTooManyRequests, http.StatusTooManyRequests},
		{CodeInternal, http.StatusInternalServerError},
		{CodeUnavailable, http.StatusServiceUnavailable},
		{CodeTimeout, http.StatusGatewayTimeout},
		{CodeInsufficientBalance, http.StatusPaymentRequired},
		{CodeQuotaExceeded, http.StatusPaymentRequired},
		{CodeProviderError, http.StatusBadGateway},
		{CodeModelNotSupported, http.StatusBadGateway},
		{CodeRateLimited, http.StatusTooManyRequests},
		{CodeUnknown, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		err := New(tt.code, "test")
		if err.HTTPStatus() != tt.expected {
			t.Errorf("Code %d: expected HTTP status %d, got %d", tt.code, tt.expected, err.HTTPStatus())
		}
	}
}

func TestNewf(t *testing.T) {
	err := Newf(CodeNotFound, "user %s not found", "user-123")
	expected := "user user-123 not found"
	if err.Message != expected {
		t.Errorf("Expected '%s', got '%s'", expected, err.Message)
	}
}

func TestWrap(t *testing.T) {
	// Test wrapping nil
	if Wrap(nil, "test") != nil {
		t.Error("Wrap(nil) should return nil")
	}

	// Test wrapping standard error
	stdErr := errors.New("standard error")
	wrapped := Wrap(stdErr, "wrapped message")
	if wrapped.Code != CodeInternal {
		t.Errorf("Expected code %d, got %d", CodeInternal, wrapped.Code)
	}
	if wrapped.Message != "wrapped message" {
		t.Errorf("Expected 'wrapped message', got '%s'", wrapped.Message)
	}

	// Test wrapping custom error
	customErr := New(CodeNotFound, "original")
	wrapped = Wrap(customErr, "wrapped")
	if wrapped.Code != CodeNotFound {
		t.Errorf("Expected code %d, got %d", CodeNotFound, wrapped.Code)
	}
}

func TestFromError(t *testing.T) {
	// Test nil error
	if FromError(nil) != nil {
		t.Error("FromError(nil) should return nil")
	}

	// Test standard error
	stdErr := errors.New("standard")
	fromStd := FromError(stdErr)
	if fromStd.Code != CodeUnknown {
		t.Errorf("Expected code %d, got %d", CodeUnknown, fromStd.Code)
	}

	// Test custom error
	customErr := New(CodeNotFound, "custom")
	fromCustom := FromError(customErr)
	if fromCustom.Code != CodeNotFound {
		t.Errorf("Expected code %d, got %d", CodeNotFound, fromCustom.Code)
	}
}

func TestIs(t *testing.T) {
	// Test nil error
	if Is(nil, CodeNotFound) {
		t.Error("Is(nil) should return false")
	}

	// Test matching code
	err := New(CodeNotFound, "not found")
	if !Is(err, CodeNotFound) {
		t.Error("Expected Is() to return true for matching code")
	}

	// Test non-matching code
	if Is(err, CodeInternal) {
		t.Error("Expected Is() to return false for non-matching code")
	}
}

func TestCommonErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      *Error
		code     int
		contains string
	}{
		{"ErrInvalidArgument", ErrInvalidArgument("bad input"), CodeInvalidArgument, "bad input"},
		{"ErrUnauthenticated", ErrUnauthenticated("not logged in"), CodeUnauthenticated, "not logged in"},
		{"ErrPermissionDenied", ErrPermissionDenied("forbidden"), CodePermissionDenied, "forbidden"},
		{"ErrNotFound", ErrNotFound("missing"), CodeNotFound, "missing"},
		{"ErrConflict", ErrConflict("duplicate"), CodeConflict, "duplicate"},
		{"ErrInternal", ErrInternal("oops"), CodeInternal, "oops"},
		{"ErrUnavailable", ErrUnavailable("down"), CodeUnavailable, "down"},
		{"ErrTimeout", ErrTimeout("slow"), CodeTimeout, "slow"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Code != tt.code {
				t.Errorf("Expected code %d, got %d", tt.code, tt.err.Code)
			}
			if tt.err.Message != tt.contains {
				t.Errorf("Expected message '%s', got '%s'", tt.contains, tt.err.Message)
			}
		})
	}
}

func TestBusinessErrors(t *testing.T) {
	// Test ErrInsufficientBalance
	insufficientErr := ErrInsufficientBalance("user-1", 100.0, 50.0)
	if insufficientErr.Code != CodeInsufficientBalance {
		t.Errorf("Expected code %d, got %d", CodeInsufficientBalance, insufficientErr.Code)
	}
	if insufficientErr.Details["user_id"] != "user-1" {
		t.Error("Expected user_id in details")
	}
	if insufficientErr.Details["required"] != 100.0 {
		t.Error("Expected required amount in details")
	}

	// Test ErrQuotaExceeded
	quotaErr := ErrQuotaExceeded("user-2", 1000.0, 1500.0)
	if quotaErr.Code != CodeQuotaExceeded {
		t.Errorf("Expected code %d, got %d", CodeQuotaExceeded, quotaErr.Code)
	}

	// Test ErrProviderError
	providerErr := ErrProviderError("openai", errors.New("api failed"))
	if providerErr.Code != CodeProviderError {
		t.Errorf("Expected code %d, got %d", CodeProviderError, providerErr.Code)
	}
	if providerErr.Details["provider"] != "openai" {
		t.Error("Expected provider in details")
	}

	// Test ErrModelNotSupported
	modelErr := ErrModelNotSupported("gpt-5", "anthropic")
	if modelErr.Code != CodeModelNotSupported {
		t.Errorf("Expected code %d, got %d", CodeModelNotSupported, modelErr.Code)
	}

	// Test ErrRateLimited
	rateErr := ErrRateLimited(60)
	if rateErr.Code != CodeRateLimited {
		t.Errorf("Expected code %d, got %d", CodeRateLimited, rateErr.Code)
	}
	if rateErr.Details["retry_after"] != 60 {
		t.Error("Expected retry_after in details")
	}
}
