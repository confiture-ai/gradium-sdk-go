package gradium

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestSTTStream_WaitReady(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("failed to upgrade: %v", err)
			return
		}
		defer conn.Close()

		// Read setup message
		var setup sttSetupMessage
		if err := conn.ReadJSON(&setup); err != nil {
			t.Errorf("failed to read setup: %v", err)
			return
		}

		// Send ready
		conn.WriteJSON(map[string]interface{}{
			"type":              "ready",
			"request_id":        "req-stt-123",
			"model_name":        "whisper",
			"sample_rate":       24000,
			"frame_size":        1920,
			"delay_in_tokens":   5,
			"text_stream_names": []string{"main"},
		})

		// Keep connection open
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client, _ := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
	client.wsURL = wsURL

	stream, err := client.STT.Stream(context.Background(), STTParams{
		InputFormat: InputFormatPCM,
	})
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}
	defer stream.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	info, err := stream.WaitReady(ctx)
	if err != nil {
		t.Errorf("WaitReady failed: %v", err)
	}

	if info == nil {
		t.Fatal("expected info, got nil")
	}

	if info.RequestID != "req-stt-123" {
		t.Errorf("expected request ID 'req-stt-123', got %q", info.RequestID)
	}
	if info.ModelName != "whisper" {
		t.Errorf("expected model name 'whisper', got %q", info.ModelName)
	}
	if info.SampleRate != 24000 {
		t.Errorf("expected sample rate 24000, got %d", info.SampleRate)
	}
	if info.FrameSize != 1920 {
		t.Errorf("expected frame size 1920, got %d", info.FrameSize)
	}
}

func TestSTTStream_WaitReadyTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Read setup but never send ready
		var setup sttSetupMessage
		conn.ReadJSON(&setup)

		time.Sleep(2 * time.Second)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client, _ := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
	client.wsURL = wsURL

	stream, _ := client.STT.Stream(context.Background(), STTParams{
		InputFormat: InputFormatPCM,
	})
	defer stream.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := stream.WaitReady(ctx)
	if err != context.DeadlineExceeded {
		t.Errorf("expected DeadlineExceeded, got %v", err)
	}
}

func TestSTTStream_SendAudio(t *testing.T) {
	var receivedAudio []byte
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Read setup
		var setup sttSetupMessage
		conn.ReadJSON(&setup)

		// Send ready
		conn.WriteJSON(map[string]interface{}{
			"type":              "ready",
			"request_id":        "req-123",
			"model_name":        "default",
			"sample_rate":       24000,
			"frame_size":        1920,
			"delay_in_tokens":   5,
			"text_stream_names": []string{"main"},
		})

		// Read audio message
		var audioMsg sttAudioMessage
		if err := conn.ReadJSON(&audioMsg); err == nil {
			decoded, _ := base64.StdEncoding.DecodeString(audioMsg.Audio)
			mu.Lock()
			receivedAudio = decoded
			mu.Unlock()
		}

		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client, _ := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
	client.wsURL = wsURL

	stream, _ := client.STT.Stream(context.Background(), STTParams{
		InputFormat: InputFormatPCM,
	})
	defer stream.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream.WaitReady(ctx)

	audioData := []byte("test audio samples")
	if err := stream.SendAudio(audioData); err != nil {
		t.Errorf("SendAudio failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if string(receivedAudio) != string(audioData) {
		t.Errorf("expected audio %q, got %q", string(audioData), string(receivedAudio))
	}
	mu.Unlock()
}

func TestSTTStream_ReceiveText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Read setup
		var setup sttSetupMessage
		conn.ReadJSON(&setup)

		// Send ready
		conn.WriteJSON(map[string]interface{}{
			"type":              "ready",
			"request_id":        "req-123",
			"model_name":        "default",
			"sample_rate":       24000,
			"frame_size":        1920,
			"delay_in_tokens":   5,
			"text_stream_names": []string{"main"},
		})

		// Wait for audio and EOS
		var msg wsMessage
		conn.ReadJSON(&msg) // audio
		conn.ReadJSON(&msg) // end_of_stream

		// Send text results
		streamID := 0
		conn.WriteJSON(map[string]interface{}{
			"type":      "text",
			"text":      "Hello",
			"start_s":   0.0,
			"stream_id": streamID,
		})
		conn.WriteJSON(map[string]interface{}{
			"type":      "text",
			"text":      "world",
			"start_s":   0.5,
			"stream_id": streamID,
		})

		// Send end_of_stream
		conn.WriteJSON(map[string]string{"type": "end_of_stream"})
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client, _ := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
	client.wsURL = wsURL

	stream, _ := client.STT.Stream(context.Background(), STTParams{
		InputFormat: InputFormatPCM,
	})
	defer stream.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream.WaitReady(ctx)
	stream.SendAudio([]byte("audio"))
	stream.SendEndOfStream()

	var texts []string
	for text := range stream.Text() {
		texts = append(texts, text.Text)
	}

	if len(texts) != 2 {
		t.Errorf("expected 2 texts, got %d", len(texts))
	}
	if len(texts) > 0 && texts[0] != "Hello" {
		t.Errorf("expected first text 'Hello', got %q", texts[0])
	}
	if len(texts) > 1 && texts[1] != "world" {
		t.Errorf("expected second text 'world', got %q", texts[1])
	}
}

func TestSTTStream_CollectText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Setup
		var setup sttSetupMessage
		conn.ReadJSON(&setup)
		conn.WriteJSON(map[string]interface{}{
			"type":              "ready",
			"request_id":        "req-123",
			"model_name":        "default",
			"sample_rate":       24000,
			"frame_size":        1920,
			"delay_in_tokens":   5,
			"text_stream_names": []string{"main"},
		})

		// Wait for audio and EOS
		var msg wsMessage
		conn.ReadJSON(&msg)
		conn.ReadJSON(&msg)

		// Send text results
		conn.WriteJSON(map[string]interface{}{
			"type":    "text",
			"text":    "The quick",
			"start_s": 0.0,
		})
		conn.WriteJSON(map[string]interface{}{
			"type":    "text",
			"text":    "brown fox",
			"start_s": 1.0,
		})
		conn.WriteJSON(map[string]string{"type": "end_of_stream"})
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client, _ := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
	client.wsURL = wsURL

	stream, _ := client.STT.Stream(context.Background(), STTParams{
		InputFormat: InputFormatPCM,
	})
	defer stream.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream.WaitReady(ctx)
	stream.SendAudio([]byte("audio"))
	stream.SendEndOfStream()

	text, err := stream.CollectText(ctx)
	if err != nil {
		t.Fatalf("CollectText failed: %v", err)
	}

	if text != "The quick brown fox" {
		t.Errorf("expected 'The quick brown fox', got %q", text)
	}
}

func TestSTTService_Transcribe(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Setup
		var setup sttSetupMessage
		conn.ReadJSON(&setup)

		if setup.InputFormat != InputFormatWAV {
			t.Errorf("expected format 'wav', got %q", setup.InputFormat)
		}

		conn.WriteJSON(map[string]interface{}{
			"type":              "ready",
			"request_id":        "req-transcribe",
			"model_name":        "default",
			"sample_rate":       24000,
			"frame_size":        1920,
			"delay_in_tokens":   5,
			"text_stream_names": []string{"main"},
		})

		// Read all audio chunks and EOS
		for {
			var msg wsMessage
			if err := conn.ReadJSON(&msg); err != nil {
				return
			}
			if msg.Type == "end_of_stream" {
				break
			}
		}

		// Send transcription
		conn.WriteJSON(map[string]interface{}{
			"type":    "text",
			"text":    "Transcribed",
			"start_s": 0.0,
		})
		conn.WriteJSON(map[string]interface{}{
			"type":    "text",
			"text":    "text",
			"start_s": 0.5,
		})
		conn.WriteJSON(map[string]string{"type": "end_of_stream"})
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client, _ := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
	client.wsURL = wsURL

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create audio data larger than chunk size to test chunking
	audioData := make([]byte, 10000)
	for i := range audioData {
		audioData[i] = byte(i % 256)
	}

	text, err := client.STT.Transcribe(ctx, STTParams{
		InputFormat: InputFormatWAV,
	}, audioData)
	if err != nil {
		t.Fatalf("Transcribe failed: %v", err)
	}

	if text != "Transcribed text" {
		t.Errorf("expected 'Transcribed text', got %q", text)
	}
}

func TestSTTStream_VAD(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Setup
		var setup sttSetupMessage
		conn.ReadJSON(&setup)
		conn.WriteJSON(map[string]interface{}{
			"type":              "ready",
			"request_id":        "req-123",
			"model_name":        "default",
			"sample_rate":       24000,
			"frame_size":        1920,
			"delay_in_tokens":   5,
			"text_stream_names": []string{"main"},
		})

		// Wait for audio
		var msg wsMessage
		conn.ReadJSON(&msg)
		conn.ReadJSON(&msg) // EOS

		// Send VAD step
		conn.WriteJSON(map[string]interface{}{
			"type": "step",
			"vad": []map[string]interface{}{
				{"horizon_s": 0.5, "inactivity_prob": 0.1},
				{"horizon_s": 1.0, "inactivity_prob": 0.8},
			},
			"step_idx":         1,
			"step_duration_s":  0.08,
			"total_duration_s": 0.08,
		})

		conn.WriteJSON(map[string]string{"type": "end_of_stream"})
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client, _ := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
	client.wsURL = wsURL

	stream, _ := client.STT.Stream(context.Background(), STTParams{
		InputFormat: InputFormatPCM,
	})
	defer stream.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream.WaitReady(ctx)
	stream.SendAudio([]byte("audio"))
	stream.SendEndOfStream()

	var vadResults []STTStepResult
	for step := range stream.VAD() {
		vadResults = append(vadResults, step)
	}

	if len(vadResults) != 1 {
		t.Errorf("expected 1 VAD result, got %d", len(vadResults))
	}

	if len(vadResults) > 0 {
		if vadResults[0].StepIdx != 1 {
			t.Errorf("expected step_idx 1, got %d", vadResults[0].StepIdx)
		}
		if len(vadResults[0].VAD) != 2 {
			t.Errorf("expected 2 VAD predictions, got %d", len(vadResults[0].VAD))
		}
	}
}

func TestSTTStream_EndText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Setup
		var setup sttSetupMessage
		conn.ReadJSON(&setup)
		conn.WriteJSON(map[string]interface{}{
			"type":              "ready",
			"request_id":        "req-123",
			"model_name":        "default",
			"sample_rate":       24000,
			"frame_size":        1920,
			"delay_in_tokens":   5,
			"text_stream_names": []string{"main"},
		})

		// Wait for audio and EOS
		var msg wsMessage
		conn.ReadJSON(&msg)
		conn.ReadJSON(&msg)

		// Send text and end_text
		streamID := 0
		conn.WriteJSON(map[string]interface{}{
			"type":      "text",
			"text":      "Hello",
			"start_s":   0.0,
			"stream_id": streamID,
		})
		conn.WriteJSON(map[string]interface{}{
			"type":      "end_text",
			"stop_s":    0.5,
			"stream_id": streamID,
		})

		conn.WriteJSON(map[string]string{"type": "end_of_stream"})
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client, _ := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
	client.wsURL = wsURL

	stream, _ := client.STT.Stream(context.Background(), STTParams{
		InputFormat: InputFormatPCM,
	})
	defer stream.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream.WaitReady(ctx)
	stream.SendAudio([]byte("audio"))
	stream.SendEndOfStream()

	var endTexts []STTEndTextResult
	for et := range stream.EndText() {
		endTexts = append(endTexts, et)
	}

	if len(endTexts) != 1 {
		t.Errorf("expected 1 end_text result, got %d", len(endTexts))
	}

	if len(endTexts) > 0 {
		if endTexts[0].StopS != 0.5 {
			t.Errorf("expected stop_s 0.5, got %f", endTexts[0].StopS)
		}
	}
}

func TestSTTStream_All(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Setup
		var setup sttSetupMessage
		conn.ReadJSON(&setup)
		conn.WriteJSON(map[string]interface{}{
			"type":              "ready",
			"request_id":        "req-123",
			"model_name":        "default",
			"sample_rate":       24000,
			"frame_size":        1920,
			"delay_in_tokens":   5,
			"text_stream_names": []string{"main"},
		})

		// Wait for audio and EOS
		var msg wsMessage
		conn.ReadJSON(&msg)
		conn.ReadJSON(&msg)

		// Send mixed messages
		conn.WriteJSON(map[string]interface{}{
			"type":    "text",
			"text":    "Hello",
			"start_s": 0.0,
		})
		conn.WriteJSON(map[string]interface{}{
			"type":             "step",
			"vad":              []map[string]interface{}{},
			"step_idx":         1,
			"step_duration_s":  0.08,
			"total_duration_s": 0.08,
		})
		conn.WriteJSON(map[string]interface{}{
			"type":   "end_text",
			"stop_s": 0.5,
		})

		conn.WriteJSON(map[string]string{"type": "end_of_stream"})
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client, _ := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
	client.wsURL = wsURL

	stream, _ := client.STT.Stream(context.Background(), STTParams{
		InputFormat: InputFormatPCM,
	})
	defer stream.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream.WaitReady(ctx)
	stream.SendAudio([]byte("audio"))
	stream.SendEndOfStream()

	msgCount := 0
	textCount := 0
	stepCount := 0
	endTextCount := 0

	for msg := range stream.All() {
		msgCount++
		switch msg.(type) {
		case STTTextResult:
			textCount++
		case STTStepResult:
			stepCount++
		case STTEndTextResult:
			endTextCount++
		}
	}

	if msgCount != 3 {
		t.Errorf("expected 3 messages, got %d", msgCount)
	}
	if textCount != 1 {
		t.Errorf("expected 1 text, got %d", textCount)
	}
	if stepCount != 1 {
		t.Errorf("expected 1 step, got %d", stepCount)
	}
	if endTextCount != 1 {
		t.Errorf("expected 1 end_text, got %d", endTextCount)
	}
}

func TestSTTStream_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Setup
		var setup sttSetupMessage
		conn.ReadJSON(&setup)

		// Send error
		conn.WriteJSON(map[string]interface{}{
			"type":    "error",
			"message": "Invalid input format",
			"code":    400,
		})
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client, _ := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
	client.wsURL = wsURL

	stream, _ := client.STT.Stream(context.Background(), STTParams{
		InputFormat: "invalid",
	})
	defer stream.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := stream.WaitReady(ctx)
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
}

func TestSTTStream_ReadyInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		var setup sttSetupMessage
		conn.ReadJSON(&setup)
		conn.WriteJSON(map[string]interface{}{
			"type":              "ready",
			"request_id":        "req-123",
			"model_name":        "whisper",
			"sample_rate":       24000,
			"frame_size":        1920,
			"delay_in_tokens":   5,
			"text_stream_names": []string{"main", "partial"},
		})

		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client, _ := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
	client.wsURL = wsURL

	stream, _ := client.STT.Stream(context.Background(), STTParams{
		InputFormat: InputFormatPCM,
	})
	defer stream.Close()

	// Before ready, should be nil
	if stream.ReadyInfo() != nil {
		t.Error("expected ReadyInfo to be nil before ready")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream.WaitReady(ctx)

	info := stream.ReadyInfo()
	if info == nil {
		t.Fatal("expected ReadyInfo after ready")
	}

	if len(info.TextStreamNames) != 2 {
		t.Errorf("expected 2 text stream names, got %d", len(info.TextStreamNames))
	}
}

func TestSTTStream_Close(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		var setup sttSetupMessage
		conn.ReadJSON(&setup)
		conn.WriteJSON(map[string]interface{}{
			"type":              "ready",
			"request_id":        "req-123",
			"model_name":        "default",
			"sample_rate":       24000,
			"frame_size":        1920,
			"delay_in_tokens":   5,
			"text_stream_names": []string{"main"},
		})

		time.Sleep(5 * time.Second)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client, _ := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
	client.wsURL = wsURL

	stream, _ := client.STT.Stream(context.Background(), STTParams{
		InputFormat: InputFormatPCM,
	})

	// Close should be idempotent
	if err := stream.Close(); err != nil {
		t.Errorf("first Close failed: %v", err)
	}
	if err := stream.Close(); err != nil {
		t.Errorf("second Close failed: %v", err)
	}
}

func TestSTTStream_Done(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		var setup sttSetupMessage
		conn.ReadJSON(&setup)
		conn.WriteJSON(map[string]interface{}{
			"type":              "ready",
			"request_id":        "req-123",
			"model_name":        "default",
			"sample_rate":       24000,
			"frame_size":        1920,
			"delay_in_tokens":   5,
			"text_stream_names": []string{"main"},
		})

		// Immediately send end_of_stream
		conn.WriteJSON(map[string]string{"type": "end_of_stream"})
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client, _ := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
	client.wsURL = wsURL

	stream, _ := client.STT.Stream(context.Background(), STTParams{
		InputFormat: InputFormatPCM,
	})
	defer stream.Close()

	select {
	case <-stream.Done():
		// Expected
	case <-time.After(5 * time.Second):
		t.Error("Done channel not closed within timeout")
	}
}

func TestSTTStream_DefaultModelName(t *testing.T) {
	var receivedModelName string
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		var setup sttSetupMessage
		conn.ReadJSON(&setup)

		mu.Lock()
		receivedModelName = setup.ModelName
		mu.Unlock()

		conn.WriteJSON(map[string]interface{}{
			"type":              "ready",
			"request_id":        "req-123",
			"model_name":        "default",
			"sample_rate":       24000,
			"frame_size":        1920,
			"delay_in_tokens":   5,
			"text_stream_names": []string{"main"},
		})
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client, _ := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
	client.wsURL = wsURL

	// Don't specify ModelName
	stream, _ := client.STT.Stream(context.Background(), STTParams{
		InputFormat: InputFormatPCM,
	})
	defer stream.Close()

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if receivedModelName != "default" {
		t.Errorf("expected model name 'default', got %q", receivedModelName)
	}
	mu.Unlock()
}
