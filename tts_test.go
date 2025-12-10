package gradium

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocket upgrader for tests
var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(_ *http.Request) bool { return true },
}

func TestTTSStream_WaitReady(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("failed to upgrade: %v", err)
			return
		}
		defer conn.Close()

		// Read setup message
		var setup ttsSetupMessage
		if err := conn.ReadJSON(&setup); err != nil {
			t.Errorf("failed to read setup: %v", err)
			return
		}

		// Send ready
		conn.WriteJSON(map[string]interface{}{
			"type":       "ready",
			"request_id": "req-123",
		})

		// Keep connection open
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client, _ := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
	client.wsURL = wsURL

	stream, err := client.TTS.Stream(context.Background(), TTSParams{
		VoiceID:      "voice-123",
		OutputFormat: FormatPCM,
	})
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}
	defer stream.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := stream.WaitReady(ctx); err != nil {
		t.Errorf("WaitReady failed: %v", err)
	}

	if stream.RequestID() != "req-123" {
		t.Errorf("expected request ID 'req-123', got %q", stream.RequestID())
	}
}

func TestTTSStream_WaitReadyTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Read setup but never send ready
		var setup ttsSetupMessage
		conn.ReadJSON(&setup)

		// Keep connection open but don't respond
		time.Sleep(2 * time.Second)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client, _ := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
	client.wsURL = wsURL

	stream, err := client.TTS.Stream(context.Background(), TTSParams{
		VoiceID:      "voice-123",
		OutputFormat: FormatPCM,
	})
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}
	defer stream.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err = stream.WaitReady(ctx)
	if err != context.DeadlineExceeded {
		t.Errorf("expected DeadlineExceeded, got %v", err)
	}
}

func TestTTSStream_SendText(t *testing.T) {
	var receivedText string
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Read setup
		var setup ttsSetupMessage
		conn.ReadJSON(&setup)

		// Send ready
		conn.WriteJSON(map[string]string{"type": "ready", "request_id": "req-123"})

		// Read text message
		var textMsg ttsTextMessage
		if err := conn.ReadJSON(&textMsg); err == nil {
			mu.Lock()
			receivedText = textMsg.Text
			mu.Unlock()
		}

		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client, _ := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
	client.wsURL = wsURL

	stream, _ := client.TTS.Stream(context.Background(), TTSParams{
		VoiceID:      "voice-123",
		OutputFormat: FormatPCM,
	})
	defer stream.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream.WaitReady(ctx)

	if err := stream.SendText("Hello, world!"); err != nil {
		t.Errorf("SendText failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if receivedText != "Hello, world!" {
		t.Errorf("expected text 'Hello, world!', got %q", receivedText)
	}
	mu.Unlock()
}

func TestTTSStream_ReceiveAudio(t *testing.T) {
	audioData := []byte("test audio data")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Read setup
		var setup ttsSetupMessage
		conn.ReadJSON(&setup)

		// Send ready
		conn.WriteJSON(map[string]string{"type": "ready", "request_id": "req-123"})

		// Wait for text
		var textMsg ttsTextMessage
		conn.ReadJSON(&textMsg)

		// Wait for end_of_stream
		var eos wsMessage
		conn.ReadJSON(&eos)

		// Send audio chunk
		conn.WriteJSON(map[string]string{
			"type":  "audio",
			"audio": base64.StdEncoding.EncodeToString(audioData),
		})

		// Send end_of_stream
		conn.WriteJSON(map[string]string{"type": "end_of_stream"})
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client, _ := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
	client.wsURL = wsURL

	stream, _ := client.TTS.Stream(context.Background(), TTSParams{
		VoiceID:      "voice-123",
		OutputFormat: FormatPCM,
	})
	defer stream.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream.WaitReady(ctx)
	stream.SendText("Hello")
	stream.SendEndOfStream()

	var receivedAudio []byte
	for chunk := range stream.Audio() {
		receivedAudio = append(receivedAudio, chunk...)
	}

	if string(receivedAudio) != string(audioData) {
		t.Errorf("expected audio %q, got %q", string(audioData), string(receivedAudio))
	}
}

func TestTTSStream_Collect(t *testing.T) {
	chunk1 := []byte("chunk1")
	chunk2 := []byte("chunk2")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Setup
		var setup ttsSetupMessage
		conn.ReadJSON(&setup)
		conn.WriteJSON(map[string]string{"type": "ready", "request_id": "req-123"})

		// Read text and EOS
		var msg wsMessage
		conn.ReadJSON(&msg)
		conn.ReadJSON(&msg)

		// Send multiple audio chunks
		conn.WriteJSON(map[string]string{
			"type":  "audio",
			"audio": base64.StdEncoding.EncodeToString(chunk1),
		})
		conn.WriteJSON(map[string]string{
			"type":  "audio",
			"audio": base64.StdEncoding.EncodeToString(chunk2),
		})
		conn.WriteJSON(map[string]string{"type": "end_of_stream"})
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client, _ := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
	client.wsURL = wsURL

	stream, _ := client.TTS.Stream(context.Background(), TTSParams{
		VoiceID:      "voice-123",
		OutputFormat: FormatPCM,
	})
	defer stream.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream.WaitReady(ctx)
	stream.SendText("Hello")
	stream.SendEndOfStream()

	result, err := stream.Collect(ctx)
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	expected := string(chunk1) + string(chunk2)
	if string(result.RawData) != expected {
		t.Errorf("expected %q, got %q", expected, string(result.RawData))
	}
	if result.SampleRate != 48000 {
		t.Errorf("expected sample rate 48000, got %d", result.SampleRate)
	}
	if result.RequestID != "req-123" {
		t.Errorf("expected request ID 'req-123', got %q", result.RequestID)
	}
}

func TestTTSService_Create(t *testing.T) {
	audioData := []byte("synthesized audio")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Setup
		var setup ttsSetupMessage
		conn.ReadJSON(&setup)

		// Verify setup params
		if setup.VoiceID != "voice-456" {
			t.Errorf("expected voice ID 'voice-456', got %q", setup.VoiceID)
		}
		if setup.OutputFormat != FormatWAV {
			t.Errorf("expected format 'wav', got %q", setup.OutputFormat)
		}

		conn.WriteJSON(map[string]string{"type": "ready", "request_id": "req-456"})

		// Read text
		var textMsg ttsTextMessage
		conn.ReadJSON(&textMsg)
		if textMsg.Text != "Test message" {
			t.Errorf("expected text 'Test message', got %q", textMsg.Text)
		}

		// Read EOS
		var eos wsMessage
		conn.ReadJSON(&eos)

		// Send audio and close
		conn.WriteJSON(map[string]string{
			"type":  "audio",
			"audio": base64.StdEncoding.EncodeToString(audioData),
		})
		conn.WriteJSON(map[string]string{"type": "end_of_stream"})
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client, _ := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
	client.wsURL = wsURL

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := client.TTS.Create(ctx, TTSParams{
		VoiceID:      "voice-456",
		OutputFormat: FormatWAV,
		Text:         "Test message",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if string(result.RawData) != string(audioData) {
		t.Errorf("expected audio data %q, got %q", string(audioData), string(result.RawData))
	}
}

func TestTTSStream_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Setup
		var setup ttsSetupMessage
		conn.ReadJSON(&setup)

		// Send error instead of ready
		conn.WriteJSON(map[string]interface{}{
			"type":    "error",
			"message": "Invalid voice ID",
			"code":    400,
		})
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client, _ := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
	client.wsURL = wsURL

	stream, _ := client.TTS.Stream(context.Background(), TTSParams{
		VoiceID:      "invalid",
		OutputFormat: FormatPCM,
	})
	defer stream.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := stream.WaitReady(ctx)
	if err == nil {
		t.Error("expected error, got nil")
		return
	}

	wsErr, ok := err.(*WebSocketError)
	if !ok {
		t.Errorf("expected WebSocketError, got %T", err)
		return
	}

	if wsErr.Code != 400 {
		t.Errorf("expected code 400, got %d", wsErr.Code)
	}
	if !strings.Contains(wsErr.Message, "Invalid voice ID") {
		t.Errorf("expected message to contain 'Invalid voice ID', got %q", wsErr.Message)
	}
}

func TestTTSStream_Done(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		var setup ttsSetupMessage
		conn.ReadJSON(&setup)
		conn.WriteJSON(map[string]string{"type": "ready", "request_id": "req-123"})

		// Immediately send end_of_stream
		conn.WriteJSON(map[string]string{"type": "end_of_stream"})
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client, _ := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
	client.wsURL = wsURL

	stream, _ := client.TTS.Stream(context.Background(), TTSParams{
		VoiceID:      "voice-123",
		OutputFormat: FormatPCM,
	})
	defer stream.Close()

	select {
	case <-stream.Done():
		// Expected
	case <-time.After(5 * time.Second):
		t.Error("Done channel not closed within timeout")
	}
}

func TestTTSStream_Close(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		var setup ttsSetupMessage
		conn.ReadJSON(&setup)
		conn.WriteJSON(map[string]string{"type": "ready", "request_id": "req-123"})

		// Keep connection open
		time.Sleep(5 * time.Second)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client, _ := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
	client.wsURL = wsURL

	stream, _ := client.TTS.Stream(context.Background(), TTSParams{
		VoiceID:      "voice-123",
		OutputFormat: FormatPCM,
	})

	// Close should be idempotent
	if err := stream.Close(); err != nil {
		t.Errorf("first Close failed: %v", err)
	}
	if err := stream.Close(); err != nil {
		t.Errorf("second Close failed: %v", err)
	}
}

func TestTTSStream_WithJSONConfig(t *testing.T) {
	var receivedConfig map[string]interface{}
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Read setup
		_, msg, _ := conn.ReadMessage()
		var setup map[string]interface{}
		json.Unmarshal(msg, &setup)

		mu.Lock()
		if cfg, ok := setup["json_config"].(map[string]interface{}); ok {
			receivedConfig = cfg
		}
		mu.Unlock()

		conn.WriteJSON(map[string]string{"type": "ready", "request_id": "req-123"})
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client, _ := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
	client.wsURL = wsURL

	stream, _ := client.TTS.Stream(context.Background(), TTSParams{
		VoiceID:      "voice-123",
		OutputFormat: FormatPCM,
		JSONConfig: &TTSConfig{
			PaddingBonus: -0.5,
		},
	})
	defer stream.Close()

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if receivedConfig == nil {
		t.Error("expected json_config to be sent")
	} else if receivedConfig["padding_bonus"] != -0.5 {
		t.Errorf("expected padding_bonus -0.5, got %v", receivedConfig["padding_bonus"])
	}
	mu.Unlock()
}

func TestTTSStream_DefaultModelName(t *testing.T) {
	var receivedModelName string
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		var setup ttsSetupMessage
		conn.ReadJSON(&setup)

		mu.Lock()
		receivedModelName = setup.ModelName
		mu.Unlock()

		conn.WriteJSON(map[string]string{"type": "ready", "request_id": "req-123"})
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client, _ := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
	client.wsURL = wsURL

	// Don't specify ModelName
	stream, _ := client.TTS.Stream(context.Background(), TTSParams{
		VoiceID:      "voice-123",
		OutputFormat: FormatPCM,
	})
	defer stream.Close()

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if receivedModelName != "default" {
		t.Errorf("expected model name 'default', got %q", receivedModelName)
	}
	mu.Unlock()
}
