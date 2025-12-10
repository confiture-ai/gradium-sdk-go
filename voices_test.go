package gradium

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestVoicesService_List(t *testing.T) {
	tests := []struct {
		name          string
		params        *VoiceListParams
		expectedQuery string
		responseCode  int
		responseBody  interface{}
		expectedErr   bool
	}{
		{
			name:          "list all voices",
			params:        nil,
			expectedQuery: "",
			responseCode:  http.StatusOK,
			responseBody: []Voice{
				{UID: "voice-1", Name: "Voice 1", Filename: "v1.wav"},
				{UID: "voice-2", Name: "Voice 2", Filename: "v2.wav"},
			},
			expectedErr: false,
		},
		{
			name:          "list with skip and limit",
			params:        &VoiceListParams{Skip: 10, Limit: 5},
			expectedQuery: "skip=10&limit=5",
			responseCode:  http.StatusOK,
			responseBody:  []Voice{},
			expectedErr:   false,
		},
		{
			name:          "list with include catalog",
			params:        &VoiceListParams{IncludeCatalog: true},
			expectedQuery: "include_catalog=true",
			responseCode:  http.StatusOK,
			responseBody:  []Voice{},
			expectedErr:   false,
		},
		{
			name:          "list with all params",
			params:        &VoiceListParams{Skip: 5, Limit: 10, IncludeCatalog: true},
			expectedQuery: "skip=5&limit=10&include_catalog=true",
			responseCode:  http.StatusOK,
			responseBody:  []Voice{},
			expectedErr:   false,
		},
		{
			name:         "unauthorized",
			params:       nil,
			responseCode: http.StatusUnauthorized,
			responseBody: map[string]string{"detail": "Invalid API key"},
			expectedErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify path
				if !strings.HasPrefix(r.URL.Path, "/voices/") {
					t.Errorf("expected path starting with '/voices/', got %q", r.URL.Path)
				}

				// Verify method
				if r.Method != http.MethodGet {
					t.Errorf("expected method GET, got %q", r.Method)
				}

				// Verify API key header
				if r.Header.Get("x-api-key") != "test-key" {
					t.Error("missing or wrong API key header")
				}

				w.WriteHeader(tt.responseCode)
				json.NewEncoder(w).Encode(tt.responseBody)
			}))
			defer server.Close()

			client, _ := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
			voices, err := client.Voices.List(context.Background(), tt.params)

			if tt.expectedErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			expected := tt.responseBody.([]Voice)
			if len(voices) != len(expected) {
				t.Errorf("expected %d voices, got %d", len(expected), len(voices))
			}
		})
	}
}

func TestVoicesService_Get(t *testing.T) {
	desc := "Test voice"
	lang := "en"

	tests := []struct {
		name         string
		voiceUID     string
		responseCode int
		responseBody interface{}
		expectedErr  bool
	}{
		{
			name:         "get existing voice",
			voiceUID:     "voice-123",
			responseCode: http.StatusOK,
			responseBody: Voice{
				UID:         "voice-123",
				Name:        "Test Voice",
				Description: &desc,
				Language:    &lang,
				StartS:      0.0,
				Filename:    "test.wav",
			},
			expectedErr: false,
		},
		{
			name:         "voice not found",
			voiceUID:     "nonexistent",
			responseCode: http.StatusNotFound,
			responseBody: map[string]string{"detail": "Voice not found"},
			expectedErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/voices/" + tt.voiceUID
				if r.URL.Path != expectedPath {
					t.Errorf("expected path %q, got %q", expectedPath, r.URL.Path)
				}

				if r.Method != http.MethodGet {
					t.Errorf("expected method GET, got %q", r.Method)
				}

				w.WriteHeader(tt.responseCode)
				json.NewEncoder(w).Encode(tt.responseBody)
			}))
			defer server.Close()

			client, _ := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
			voice, err := client.Voices.Get(context.Background(), tt.voiceUID)

			if tt.expectedErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if voice == nil {
				t.Error("expected voice, got nil")
				return
			}

			expected := tt.responseBody.(Voice)
			if voice.UID != expected.UID {
				t.Errorf("expected UID %q, got %q", expected.UID, voice.UID)
			}
			if voice.Name != expected.Name {
				t.Errorf("expected Name %q, got %q", expected.Name, voice.Name)
			}
		})
	}
}

func TestVoicesService_Create(t *testing.T) {
	tests := []struct {
		name         string
		params       VoiceCreateParams
		audioData    string
		filename     string
		responseCode int
		responseBody interface{}
		expectedErr  bool
	}{
		{
			name: "create voice",
			params: VoiceCreateParams{
				Name:        "Test Voice",
				InputFormat: "wav",
			},
			audioData:    "fake audio data",
			filename:     "test.wav",
			responseCode: http.StatusCreated,
			responseBody: VoiceCreateResponse{
				UID:        stringPtr("voice-new"),
				WasUpdated: false,
			},
			expectedErr: false,
		},
		{
			name: "create with all params",
			params: VoiceCreateParams{
				Name:        "Full Voice",
				Description: stringPtr("A test voice"),
				Language:    stringPtr("en"),
				StartS:      0.5,
				TimeoutS:    10.0,
				InputFormat: "wav",
			},
			audioData:    "fake audio data",
			filename:     "full.wav",
			responseCode: http.StatusCreated,
			responseBody: VoiceCreateResponse{
				UID:        stringPtr("voice-full"),
				WasUpdated: false,
			},
			expectedErr: false,
		},
		{
			name: "update existing voice",
			params: VoiceCreateParams{
				Name: "Existing Voice",
			},
			audioData:    "fake audio data",
			filename:     "existing.wav",
			responseCode: http.StatusCreated,
			responseBody: VoiceCreateResponse{
				UID:        stringPtr("voice-existing"),
				WasUpdated: true,
			},
			expectedErr: false,
		},
		{
			name: "validation error",
			params: VoiceCreateParams{
				Name: "",
			},
			audioData:    "fake audio data",
			filename:     "test.wav",
			responseCode: http.StatusUnprocessableEntity,
			responseBody: map[string]interface{}{
				"detail": []map[string]interface{}{
					{"msg": "name is required", "type": "value_error.missing"},
				},
			},
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/voices/" {
					t.Errorf("expected path '/voices/', got %q", r.URL.Path)
				}

				if r.Method != http.MethodPost {
					t.Errorf("expected method POST, got %q", r.Method)
				}

				// Verify it's multipart form data
				contentType := r.Header.Get("Content-Type")
				if !strings.HasPrefix(contentType, "multipart/form-data") {
					t.Errorf("expected multipart/form-data, got %q", contentType)
				}

				w.WriteHeader(tt.responseCode)
				json.NewEncoder(w).Encode(tt.responseBody)
			}))
			defer server.Close()

			client, _ := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
			audioReader := bytes.NewReader([]byte(tt.audioData))

			result, err := client.Voices.Create(context.Background(), audioReader, tt.filename, tt.params)

			if tt.expectedErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Error("expected result, got nil")
				return
			}

			expected := tt.responseBody.(VoiceCreateResponse)
			if result.WasUpdated != expected.WasUpdated {
				t.Errorf("expected WasUpdated %v, got %v", expected.WasUpdated, result.WasUpdated)
			}
		})
	}
}

func TestVoicesService_Update(t *testing.T) {
	name := "Updated Name"
	desc := "Updated description"

	tests := []struct {
		name         string
		voiceUID     string
		params       VoiceUpdateParams
		responseCode int
		responseBody interface{}
		expectedErr  bool
	}{
		{
			name:     "update voice name",
			voiceUID: "voice-123",
			params: VoiceUpdateParams{
				Name: &name,
			},
			responseCode: http.StatusOK,
			responseBody: Voice{
				UID:      "voice-123",
				Name:     name,
				Filename: "test.wav",
			},
			expectedErr: false,
		},
		{
			name:     "update multiple fields",
			voiceUID: "voice-123",
			params: VoiceUpdateParams{
				Name:        &name,
				Description: &desc,
			},
			responseCode: http.StatusOK,
			responseBody: Voice{
				UID:         "voice-123",
				Name:        name,
				Description: &desc,
				Filename:    "test.wav",
			},
			expectedErr: false,
		},
		{
			name:     "voice not found",
			voiceUID: "nonexistent",
			params: VoiceUpdateParams{
				Name: &name,
			},
			responseCode: http.StatusNotFound,
			responseBody: map[string]string{"detail": "Voice not found"},
			expectedErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/voices/" + tt.voiceUID
				if r.URL.Path != expectedPath {
					t.Errorf("expected path %q, got %q", expectedPath, r.URL.Path)
				}

				if r.Method != http.MethodPut {
					t.Errorf("expected method PUT, got %q", r.Method)
				}

				// Verify JSON body
				var body map[string]interface{}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Errorf("failed to decode request body: %v", err)
				}

				w.WriteHeader(tt.responseCode)
				json.NewEncoder(w).Encode(tt.responseBody)
			}))
			defer server.Close()

			client, _ := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
			voice, err := client.Voices.Update(context.Background(), tt.voiceUID, tt.params)

			if tt.expectedErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if voice == nil {
				t.Error("expected voice, got nil")
				return
			}

			expected := tt.responseBody.(Voice)
			if voice.Name != expected.Name {
				t.Errorf("expected Name %q, got %q", expected.Name, voice.Name)
			}
		})
	}
}

func TestVoicesService_Delete(t *testing.T) {
	tests := []struct {
		name         string
		voiceUID     string
		responseCode int
		responseBody interface{}
		expectedErr  bool
	}{
		{
			name:         "delete existing voice",
			voiceUID:     "voice-123",
			responseCode: http.StatusNoContent,
			responseBody: nil,
			expectedErr:  false,
		},
		{
			name:         "delete nonexistent voice",
			voiceUID:     "nonexistent",
			responseCode: http.StatusNotFound,
			responseBody: map[string]string{"detail": "Voice not found"},
			expectedErr:  true,
		},
		{
			name:         "unauthorized",
			voiceUID:     "voice-123",
			responseCode: http.StatusUnauthorized,
			responseBody: map[string]string{"detail": "Invalid API key"},
			expectedErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/voices/" + tt.voiceUID
				if r.URL.Path != expectedPath {
					t.Errorf("expected path %q, got %q", expectedPath, r.URL.Path)
				}

				if r.Method != http.MethodDelete {
					t.Errorf("expected method DELETE, got %q", r.Method)
				}

				w.WriteHeader(tt.responseCode)
				if tt.responseBody != nil {
					json.NewEncoder(w).Encode(tt.responseBody)
				}
			}))
			defer server.Close()

			client, _ := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
			err := client.Voices.Delete(context.Background(), tt.voiceUID)

			if tt.expectedErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestVoicesService_CreateWithReader(t *testing.T) {
	// Test that Create properly reads from any io.Reader
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse multipart form
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Errorf("failed to parse form: %v", err)
		}

		// Check audio file
		file, header, err := r.FormFile("audio_file")
		if err != nil {
			t.Errorf("failed to get audio file: %v", err)
		}
		defer file.Close()

		if header.Filename != "custom.wav" {
			t.Errorf("expected filename 'custom.wav', got %q", header.Filename)
		}

		content, _ := io.ReadAll(file)
		if string(content) != "test audio bytes" {
			t.Errorf("expected content 'test audio bytes', got %q", string(content))
		}

		// Check name field
		if r.FormValue("name") != "Custom Voice" {
			t.Errorf("expected name 'Custom Voice', got %q", r.FormValue("name"))
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(VoiceCreateResponse{
			UID:        stringPtr("voice-custom"),
			WasUpdated: false,
		})
	}))
	defer server.Close()

	client, _ := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))

	// Use strings.Reader as the io.Reader
	reader := strings.NewReader("test audio bytes")

	result, err := client.Voices.Create(context.Background(), reader, "custom.wav", VoiceCreateParams{
		Name: "Custom Voice",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.UID == nil || *result.UID != "voice-custom" {
		t.Error("unexpected UID")
	}
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
