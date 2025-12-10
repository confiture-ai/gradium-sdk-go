package gradium

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

// Error is the base error type for all SDK errors.
type Error struct {
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

// AuthenticationError is returned when the API key is missing or invalid.
type AuthenticationError struct {
	Message string
}

func (e *AuthenticationError) Error() string {
	if e.Message == "" {
		return "invalid or missing API key"
	}
	return e.Message
}

// ValidationErrorDetail contains details about a validation error.
type ValidationErrorDetail struct {
	Loc  []interface{} `json:"loc"`
	Msg  string        `json:"msg"`
	Type string        `json:"type"`
}

// ValidationError is returned when the API returns a 422 validation error.
type ValidationError struct {
	Status int
	Errors []ValidationErrorDetail
}

func (e *ValidationError) Error() string {
	if len(e.Errors) == 0 {
		return "validation error"
	}
	msg := "validation error: "
	for i, err := range e.Errors {
		if i > 0 {
			msg += "; "
		}
		msg += err.Msg
	}
	return msg
}

// APIError is returned for general API errors.
type APIError struct {
	Status  int
	Message string
	Body    interface{}
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error (%d): %s", e.Status, e.Message)
}

// NotFoundError is returned when a resource is not found.
type NotFoundError struct {
	Message string
}

func (e *NotFoundError) Error() string {
	if e.Message == "" {
		return "resource not found"
	}
	return e.Message
}

// RateLimitError is returned when the rate limit is exceeded.
type RateLimitError struct {
	Message    string
	RetryAfter int
}

func (e *RateLimitError) Error() string {
	if e.Message == "" {
		return "rate limit exceeded"
	}
	return e.Message
}

// InternalServerError is returned for 5xx errors.
type InternalServerError struct {
	Status  int
	Message string
}

func (e *InternalServerError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("internal server error (%d)", e.Status)
	}
	return e.Message
}

// WebSocketError is returned when a WebSocket operation fails.
type WebSocketError struct {
	Message string
	Code    int
}

func (e *WebSocketError) Error() string {
	if e.Code != 0 {
		return fmt.Sprintf("websocket error (%d): %s", e.Code, e.Message)
	}
	return fmt.Sprintf("websocket error: %s", e.Message)
}

// TimeoutError is returned when a request times out.
type TimeoutError struct {
	Message string
}

func (e *TimeoutError) Error() string {
	if e.Message == "" {
		return "request timed out"
	}
	return e.Message
}

// ConnectionError is returned when a connection fails.
type ConnectionError struct {
	Message string
}

func (e *ConnectionError) Error() string {
	if e.Message == "" {
		return "failed to connect to the API"
	}
	return e.Message
}

// httpValidationError is the JSON structure for 422 errors.
type httpValidationError struct {
	Detail []ValidationErrorDetail `json:"detail"`
}

// handleAPIError parses an HTTP response and returns the appropriate error.
func handleAPIError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	var detail struct {
		Detail interface{} `json:"detail"`
	}
	_ = json.Unmarshal(body, &detail)

	getMessage := func() string {
		if s, ok := detail.Detail.(string); ok {
			return s
		}
		return string(body)
	}

	switch resp.StatusCode {
	case 422:
		var validationErr httpValidationError
		if err := json.Unmarshal(body, &validationErr); err == nil {
			return &ValidationError{Status: 422, Errors: validationErr.Detail}
		}
		return &ValidationError{Status: 422}

	case 401, 403:
		return &AuthenticationError{Message: getMessage()}

	case 404:
		return &NotFoundError{Message: getMessage()}

	case 429:
		retryAfter := 0
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			retryAfter, _ = strconv.Atoi(ra)
		}
		return &RateLimitError{Message: getMessage(), RetryAfter: retryAfter}
	}

	if resp.StatusCode >= 500 {
		return &InternalServerError{Status: resp.StatusCode, Message: getMessage()}
	}

	return &APIError{Status: resp.StatusCode, Message: getMessage(), Body: body}
}
