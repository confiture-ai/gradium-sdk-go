package gradium

import (
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name        string
		opts        []ClientOption
		wantKey     string
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid API key via option",
			opts:    []ClientOption{WithAPIKey("test-api-key")},
			wantKey: "test-api-key",
			wantErr: false,
		},
		{
			name:        "no API key without env",
			opts:        nil,
			wantErr:     true,
			errContains: "API key is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.opts...)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if tt.errContains != "" && err != nil {
					var authErr *AuthenticationError
					if !errors.As(err, &authErr) {
						t.Errorf("expected AuthenticationError, got %T", err)
					}
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if client == nil {
				t.Error("expected client, got nil")
				return
			}
			if client.APIKey() != tt.wantKey {
				t.Errorf("expected API key %q, got %q", tt.wantKey, client.APIKey())
			}
		})
	}
}

func TestNewClientDefaults(t *testing.T) {
	client, err := NewClient(WithAPIKey("test-key"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test default region is EU
	if client.region != RegionEU {
		t.Errorf("expected region %v, got %v", RegionEU, client.region)
	}

	// Test default base URL
	expectedURL := apiURLs[RegionEU]
	if client.BaseURL() != expectedURL {
		t.Errorf("expected base URL %q, got %q", expectedURL, client.BaseURL())
	}

	// Test default WebSocket URL
	expectedWSURL := wsURLs[RegionEU]
	if client.WSURL() != expectedWSURL {
		t.Errorf("expected WS URL %q, got %q", expectedWSURL, client.WSURL())
	}

	// Test default timeout
	expectedTimeout := 30 * time.Second
	if client.timeout != expectedTimeout {
		t.Errorf("expected timeout %v, got %v", expectedTimeout, client.timeout)
	}

	// Test services are initialized
	if client.TTS == nil {
		t.Error("TTS service is nil")
	}
	if client.STT == nil {
		t.Error("STT service is nil")
	}
	if client.Voices == nil {
		t.Error("Voices service is nil")
	}
	if client.Credits == nil {
		t.Error("Credits service is nil")
	}
}

func TestWithRegion(t *testing.T) {
	tests := []struct {
		name          string
		region        Region
		expectedURL   string
		expectedWSURL string
	}{
		{
			name:          "EU region",
			region:        RegionEU,
			expectedURL:   apiURLs[RegionEU],
			expectedWSURL: wsURLs[RegionEU],
		},
		{
			name:          "US region",
			region:        RegionUS,
			expectedURL:   apiURLs[RegionUS],
			expectedWSURL: wsURLs[RegionUS],
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(WithAPIKey("test-key"), WithRegion(tt.region))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if client.region != tt.region {
				t.Errorf("expected region %v, got %v", tt.region, client.region)
			}
			if client.BaseURL() != tt.expectedURL {
				t.Errorf("expected base URL %q, got %q", tt.expectedURL, client.BaseURL())
			}
			if client.WSURL() != tt.expectedWSURL {
				t.Errorf("expected WS URL %q, got %q", tt.expectedWSURL, client.WSURL())
			}
		})
	}
}

func TestWithBaseURL(t *testing.T) {
	tests := []struct {
		name          string
		baseURL       string
		expectedURL   string
		expectedWSURL string
	}{
		{
			name:          "HTTPS URL",
			baseURL:       "https://custom.api.com/api",
			expectedURL:   "https://custom.api.com/api",
			expectedWSURL: "wss://custom.api.com/api/speech",
		},
		{
			name:          "HTTPS URL with trailing slash",
			baseURL:       "https://custom.api.com/api/",
			expectedURL:   "https://custom.api.com/api",
			expectedWSURL: "wss://custom.api.com/api/speech",
		},
		{
			name:          "HTTP URL",
			baseURL:       "http://localhost:8080/api",
			expectedURL:   "http://localhost:8080/api",
			expectedWSURL: "ws://localhost:8080/api/speech",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(WithAPIKey("test-key"), WithBaseURL(tt.baseURL))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if client.BaseURL() != tt.expectedURL {
				t.Errorf("expected base URL %q, got %q", tt.expectedURL, client.BaseURL())
			}
			if client.WSURL() != tt.expectedWSURL {
				t.Errorf("expected WS URL %q, got %q", tt.expectedWSURL, client.WSURL())
			}
		})
	}
}

func TestWithTimeout(t *testing.T) {
	timeout := 60 * time.Second
	client, err := NewClient(WithAPIKey("test-key"), WithTimeout(timeout))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if client.timeout != timeout {
		t.Errorf("expected timeout %v, got %v", timeout, client.timeout)
	}
	if client.httpClient.Timeout != timeout {
		t.Errorf("expected HTTP client timeout %v, got %v", timeout, client.httpClient.Timeout)
	}
}

func TestWithHTTPClient(t *testing.T) {
	customClient := &http.Client{
		Timeout: 120 * time.Second,
	}

	client, err := NewClient(WithAPIKey("test-key"), WithHTTPClient(customClient))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if client.httpClient != customClient {
		t.Error("expected custom HTTP client to be set")
	}
}

func TestMultipleOptions(t *testing.T) {
	timeout := 45 * time.Second
	client, err := NewClient(
		WithAPIKey("test-key"),
		WithRegion(RegionUS),
		WithTimeout(timeout),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if client.region != RegionUS {
		t.Errorf("expected region %v, got %v", RegionUS, client.region)
	}
	if client.timeout != timeout {
		t.Errorf("expected timeout %v, got %v", timeout, client.timeout)
	}
}

func TestRegionConstants(t *testing.T) {
	if RegionEU != "eu" {
		t.Errorf("expected RegionEU to be 'eu', got %q", RegionEU)
	}
	if RegionUS != "us" {
		t.Errorf("expected RegionUS to be 'us', got %q", RegionUS)
	}
}

func TestAPIURLs(t *testing.T) {
	if _, ok := apiURLs[RegionEU]; !ok {
		t.Error("missing EU API URL")
	}
	if _, ok := apiURLs[RegionUS]; !ok {
		t.Error("missing US API URL")
	}
}

func TestWSURLs(t *testing.T) {
	if _, ok := wsURLs[RegionEU]; !ok {
		t.Error("missing EU WebSocket URL")
	}
	if _, ok := wsURLs[RegionUS]; !ok {
		t.Error("missing US WebSocket URL")
	}
}
