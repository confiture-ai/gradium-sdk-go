package gradium

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreditsService_Get(t *testing.T) {
	tests := []struct {
		name            string
		responseCode    int
		responseBody    interface{}
		expectedErr     bool
		expectedErrType string
	}{
		{
			name:         "successful response",
			responseCode: http.StatusOK,
			responseBody: CreditsSummary{
				RemainingCredits: 1000,
				AllocatedCredits: 5000,
				BillingPeriod:    "monthly",
				PlanName:         "Professional",
			},
			expectedErr: false,
		},
		{
			name:            "unauthorized",
			responseCode:    http.StatusUnauthorized,
			responseBody:    map[string]string{"detail": "Invalid API key"},
			expectedErr:     true,
			expectedErrType: "*gradium.AuthenticationError",
		},
		{
			name:            "rate limited",
			responseCode:    http.StatusTooManyRequests,
			responseBody:    map[string]string{"detail": "Rate limit exceeded"},
			expectedErr:     true,
			expectedErrType: "*gradium.RateLimitError",
		},
		{
			name:            "internal server error",
			responseCode:    http.StatusInternalServerError,
			responseBody:    map[string]string{"detail": "Internal error"},
			expectedErr:     true,
			expectedErrType: "*gradium.InternalServerError",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				if r.URL.Path != "/usages/credits" {
					t.Errorf("expected path '/usages/credits', got %q", r.URL.Path)
				}
				if r.Method != http.MethodGet {
					t.Errorf("expected method GET, got %q", r.Method)
				}
				if r.Header.Get("x-api-key") != "test-key" {
					t.Error("missing or wrong API key header")
				}
				if r.Header.Get("Accept") != "application/json" {
					t.Error("missing Accept header")
				}

				w.WriteHeader(tt.responseCode)
				_ = json.NewEncoder(w).Encode(tt.responseBody)
			}))
			defer server.Close()

			client, err := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
			if err != nil {
				t.Fatalf("failed to create client: %v", err)
			}

			credits, err := client.Credits.Get(context.Background())

			if tt.expectedErr {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}
				if tt.expectedErrType != "" {
					errTypeName := getErrorTypeName(err)
					if errTypeName != tt.expectedErrType {
						t.Errorf("expected error type %s, got %s", tt.expectedErrType, errTypeName)
					}
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if credits == nil {
				t.Error("expected credits, got nil")
				return
			}

			expected := tt.responseBody.(CreditsSummary)
			if credits.RemainingCredits != expected.RemainingCredits {
				t.Errorf("expected RemainingCredits %d, got %d", expected.RemainingCredits, credits.RemainingCredits)
			}
			if credits.AllocatedCredits != expected.AllocatedCredits {
				t.Errorf("expected AllocatedCredits %d, got %d", expected.AllocatedCredits, credits.AllocatedCredits)
			}
			if credits.PlanName != expected.PlanName {
				t.Errorf("expected PlanName %q, got %q", expected.PlanName, credits.PlanName)
			}
		})
	}
}

func TestCreditsService_GetWithNextRolloverDate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"remaining_credits": 1000,
			"allocated_credits": 5000,
			"billing_period": "monthly",
			"next_rollover_date": "2024-02-01",
			"plan_name": "Professional"
		}`))
	}))
	defer server.Close()

	client, _ := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
	credits, err := client.Credits.Get(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if credits.NextRolloverDate == nil {
		t.Error("expected NextRolloverDate to be set")
	} else if *credits.NextRolloverDate != "2024-02-01" {
		t.Errorf("expected NextRolloverDate '2024-02-01', got %q", *credits.NextRolloverDate)
	}
}

func TestCreditsService_GetContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		// Simulate slow response - won't actually be reached due to context cancellation
		select {}
	}))
	defer server.Close()

	client, _ := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := client.Credits.Get(ctx)
	if err == nil {
		t.Error("expected error due to cancelled context")
	}
}

// Helper function to get error type name
func getErrorTypeName(err error) string {
	var validationErr *ValidationError
	var authErr *AuthenticationError
	var notFoundErr *NotFoundError
	var rateLimitErr *RateLimitError
	var internalErr *InternalServerError
	var apiErr *APIError
	var connErr *ConnectionError

	switch {
	case errors.As(err, &validationErr):
		return "*gradium.ValidationError"
	case errors.As(err, &authErr):
		return "*gradium.AuthenticationError"
	case errors.As(err, &notFoundErr):
		return "*gradium.NotFoundError"
	case errors.As(err, &rateLimitErr):
		return "*gradium.RateLimitError"
	case errors.As(err, &internalErr):
		return "*gradium.InternalServerError"
	case errors.As(err, &apiErr):
		return "*gradium.APIError"
	case errors.As(err, &connErr):
		return "*gradium.ConnectionError"
	default:
		return ""
	}
}
