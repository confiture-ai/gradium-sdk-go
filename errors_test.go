package gradium

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestError(t *testing.T) {
	err := &Error{Message: "test error"}
	if err.Error() != "test error" {
		t.Errorf("expected 'test error', got %q", err.Error())
	}
}

func TestAuthenticationError(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "with message",
			message:  "invalid API key provided",
			expected: "invalid API key provided",
		},
		{
			name:     "without message",
			message:  "",
			expected: "invalid or missing API key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &AuthenticationError{Message: tt.message}
			if err.Error() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, err.Error())
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	tests := []struct {
		name     string
		errors   []ValidationErrorDetail
		expected string
	}{
		{
			name:     "no errors",
			errors:   nil,
			expected: "validation error",
		},
		{
			name: "single error",
			errors: []ValidationErrorDetail{
				{Msg: "field is required", Type: "missing"},
			},
			expected: "validation error: field is required",
		},
		{
			name: "multiple errors",
			errors: []ValidationErrorDetail{
				{Msg: "field1 is required", Type: "missing"},
				{Msg: "field2 is invalid", Type: "type_error"},
			},
			expected: "validation error: field1 is required; field2 is invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &ValidationError{Status: 422, Errors: tt.errors}
			if err.Error() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, err.Error())
			}
		})
	}
}

func TestAPIError(t *testing.T) {
	err := &APIError{Status: 400, Message: "bad request"}
	expected := "API error (400): bad request"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "with message",
			message:  "voice not found",
			expected: "voice not found",
		},
		{
			name:     "without message",
			message:  "",
			expected: "resource not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &NotFoundError{Message: tt.message}
			if err.Error() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, err.Error())
			}
		})
	}
}

func TestRateLimitError(t *testing.T) {
	tests := []struct {
		name       string
		message    string
		retryAfter int
		expected   string
	}{
		{
			name:       "with message",
			message:    "too many requests",
			retryAfter: 60,
			expected:   "too many requests",
		},
		{
			name:       "without message",
			message:    "",
			retryAfter: 30,
			expected:   "rate limit exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &RateLimitError{Message: tt.message, RetryAfter: tt.retryAfter}
			if err.Error() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, err.Error())
			}
		})
	}
}

func TestInternalServerError(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		message  string
		expected string
	}{
		{
			name:     "with message",
			status:   500,
			message:  "database connection failed",
			expected: "database connection failed",
		},
		{
			name:     "without message",
			status:   503,
			message:  "",
			expected: "internal server error (503)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &InternalServerError{Status: tt.status, Message: tt.message}
			if err.Error() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, err.Error())
			}
		})
	}
}

func TestWebSocketError(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		code     int
		expected string
	}{
		{
			name:     "with code",
			message:  "connection closed",
			code:     1006,
			expected: "websocket error (1006): connection closed",
		},
		{
			name:     "without code",
			message:  "connection failed",
			code:     0,
			expected: "websocket error: connection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &WebSocketError{Message: tt.message, Code: tt.code}
			if err.Error() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, err.Error())
			}
		})
	}
}

func TestTimeoutError(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "with message",
			message:  "connection timed out after 30s",
			expected: "connection timed out after 30s",
		},
		{
			name:     "without message",
			message:  "",
			expected: "request timed out",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &TimeoutError{Message: tt.message}
			if err.Error() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, err.Error())
			}
		})
	}
}

func TestConnectionError(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "with message",
			message:  "dial tcp: connection refused",
			expected: "dial tcp: connection refused",
		},
		{
			name:     "without message",
			message:  "",
			expected: "failed to connect to the API",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &ConnectionError{Message: tt.message}
			if err.Error() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, err.Error())
			}
		})
	}
}

// mockReadCloser implements io.ReadCloser for testing
type mockReadCloser struct {
	io.Reader
}

func (m *mockReadCloser) Close() error { return nil }

func TestHandleAPIError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		headers    map[string]string
		errType    string
	}{
		{
			name:       "422 validation error",
			statusCode: 422,
			body:       `{"detail": [{"loc": ["body", "name"], "msg": "field required", "type": "value_error.missing"}]}`,
			errType:    "*gradium.ValidationError",
		},
		{
			name:       "401 authentication error",
			statusCode: 401,
			body:       `{"detail": "Invalid API key"}`,
			errType:    "*gradium.AuthenticationError",
		},
		{
			name:       "403 authentication error",
			statusCode: 403,
			body:       `{"detail": "Access denied"}`,
			errType:    "*gradium.AuthenticationError",
		},
		{
			name:       "404 not found error",
			statusCode: 404,
			body:       `{"detail": "Voice not found"}`,
			errType:    "*gradium.NotFoundError",
		},
		{
			name:       "429 rate limit error",
			statusCode: 429,
			body:       `{"detail": "Rate limit exceeded"}`,
			headers:    map[string]string{"Retry-After": "60"},
			errType:    "*gradium.RateLimitError",
		},
		{
			name:       "500 internal server error",
			statusCode: 500,
			body:       `{"detail": "Internal error"}`,
			errType:    "*gradium.InternalServerError",
		},
		{
			name:       "502 internal server error",
			statusCode: 502,
			body:       `{"detail": "Bad gateway"}`,
			errType:    "*gradium.InternalServerError",
		},
		{
			name:       "400 generic API error",
			statusCode: 400,
			body:       `{"detail": "Bad request"}`,
			errType:    "*gradium.APIError",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				StatusCode: tt.statusCode,
				Body:       &mockReadCloser{Reader: strings.NewReader(tt.body)},
				Header:     make(http.Header),
			}
			for k, v := range tt.headers {
				resp.Header.Set(k, v)
			}

			err := handleAPIError(resp)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			// Check error type using errors.As
			var validationErr *ValidationError
			var authErr *AuthenticationError
			var notFoundErr *NotFoundError
			var rateLimitErr *RateLimitError
			var internalErr *InternalServerError
			var apiErr *APIError

			errTypeName := ""
			switch {
			case errors.As(err, &validationErr):
				errTypeName = "*gradium.ValidationError"
			case errors.As(err, &authErr):
				errTypeName = "*gradium.AuthenticationError"
			case errors.As(err, &notFoundErr):
				errTypeName = "*gradium.NotFoundError"
			case errors.As(err, &rateLimitErr):
				errTypeName = "*gradium.RateLimitError"
			case errors.As(err, &internalErr):
				errTypeName = "*gradium.InternalServerError"
			case errors.As(err, &apiErr):
				errTypeName = "*gradium.APIError"
			}

			if errTypeName != tt.errType {
				t.Errorf("expected error type %s, got %s", tt.errType, errTypeName)
			}
		})
	}
}

func TestHandleAPIErrorRetryAfter(t *testing.T) {
	resp := &http.Response{
		StatusCode: 429,
		Body:       &mockReadCloser{Reader: strings.NewReader(`{"detail": "Rate limit exceeded"}`)},
		Header:     make(http.Header),
	}
	resp.Header.Set("Retry-After", "120")

	err := handleAPIError(resp)
	var rateLimitErr *RateLimitError
	if !errors.As(err, &rateLimitErr) {
		t.Fatalf("expected RateLimitError, got %T", err)
	}
	if rateLimitErr.RetryAfter != 120 {
		t.Errorf("expected RetryAfter 120, got %d", rateLimitErr.RetryAfter)
	}
}

func TestValidationErrorDetail(t *testing.T) {
	detail := ValidationErrorDetail{
		Loc:  []interface{}{"body", "voice_id"},
		Msg:  "field required",
		Type: "value_error.missing",
	}

	if detail.Msg != "field required" {
		t.Errorf("expected msg 'field required', got %q", detail.Msg)
	}
	if detail.Type != "value_error.missing" {
		t.Errorf("expected type 'value_error.missing', got %q", detail.Type)
	}
	if len(detail.Loc) != 2 {
		t.Errorf("expected loc length 2, got %d", len(detail.Loc))
	}
}

// Test that all error types implement the error interface
func TestErrorInterface(_ *testing.T) {
	var _ error = &Error{}
	var _ error = &AuthenticationError{}
	var _ error = &ValidationError{}
	var _ error = &APIError{}
	var _ error = &NotFoundError{}
	var _ error = &RateLimitError{}
	var _ error = &InternalServerError{}
	var _ error = &WebSocketError{}
	var _ error = &TimeoutError{}
	var _ error = &ConnectionError{}
}
