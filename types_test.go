package gradium

import (
	"encoding/json"
	"testing"
)

func TestOutputFormatConstants(t *testing.T) {
	tests := []struct {
		format   OutputFormat
		expected string
	}{
		{FormatWAV, "wav"},
		{FormatPCM, "pcm"},
		{FormatOpus, "opus"},
		{FormatULaw8000, "ulaw_8000"},
		{FormatALaw8000, "alaw_8000"},
		{FormatPCM16000, "pcm_16000"},
		{FormatPCM24000, "pcm_24000"},
	}

	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			if string(tt.format) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, string(tt.format))
			}
		})
	}
}

func TestInputFormatConstants(t *testing.T) {
	tests := []struct {
		format   InputFormat
		expected string
	}{
		{InputFormatPCM, "pcm"},
		{InputFormatWAV, "wav"},
		{InputFormatOpus, "opus"},
	}

	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			if string(tt.format) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, string(tt.format))
			}
		})
	}
}

func TestVoiceJSONMarshal(t *testing.T) {
	desc := "Test description"
	lang := "en"
	stopS := 10.5

	voice := Voice{
		UID:         "test-uid",
		Name:        "Test Voice",
		Description: &desc,
		Language:    &lang,
		StartS:      0.5,
		StopS:       &stopS,
		Filename:    "test.wav",
	}

	data, err := json.Marshal(voice)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed Voice
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.UID != voice.UID {
		t.Errorf("UID mismatch: expected %q, got %q", voice.UID, parsed.UID)
	}
	if parsed.Name != voice.Name {
		t.Errorf("Name mismatch: expected %q, got %q", voice.Name, parsed.Name)
	}
	if parsed.Description == nil || *parsed.Description != *voice.Description {
		t.Error("Description mismatch")
	}
	if parsed.Language == nil || *parsed.Language != *voice.Language {
		t.Error("Language mismatch")
	}
	if parsed.StartS != voice.StartS {
		t.Errorf("StartS mismatch: expected %f, got %f", voice.StartS, parsed.StartS)
	}
	if parsed.StopS == nil || *parsed.StopS != *voice.StopS {
		t.Error("StopS mismatch")
	}
}

func TestVoiceJSONMarshalOmitEmpty(t *testing.T) {
	voice := Voice{
		UID:      "test-uid",
		Name:     "Test Voice",
		StartS:   0.0,
		Filename: "test.wav",
	}

	data, err := json.Marshal(voice)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Verify optional fields are omitted
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if _, ok := parsed["description"]; ok {
		t.Error("description should be omitted when nil")
	}
	if _, ok := parsed["language"]; ok {
		t.Error("language should be omitted when nil")
	}
	if _, ok := parsed["stop_s"]; ok {
		t.Error("stop_s should be omitted when nil")
	}
}

func TestCreditsSummaryJSONUnmarshal(t *testing.T) {
	jsonData := `{
		"remaining_credits": 1000,
		"allocated_credits": 5000,
		"billing_period": "monthly",
		"next_rollover_date": "2024-02-01",
		"plan_name": "Professional"
	}`

	var credits CreditsSummary
	if err := json.Unmarshal([]byte(jsonData), &credits); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if credits.RemainingCredits != 1000 {
		t.Errorf("expected RemainingCredits 1000, got %d", credits.RemainingCredits)
	}
	if credits.AllocatedCredits != 5000 {
		t.Errorf("expected AllocatedCredits 5000, got %d", credits.AllocatedCredits)
	}
	if credits.BillingPeriod != "monthly" {
		t.Errorf("expected BillingPeriod 'monthly', got %q", credits.BillingPeriod)
	}
	if credits.NextRolloverDate == nil || *credits.NextRolloverDate != "2024-02-01" {
		t.Error("NextRolloverDate mismatch")
	}
	if credits.PlanName != "Professional" {
		t.Errorf("expected PlanName 'Professional', got %q", credits.PlanName)
	}
}

func TestTTSParamsJSONMarshal(t *testing.T) {
	params := TTSParams{
		VoiceID:      "voice-123",
		OutputFormat: FormatWAV,
		ModelName:    "default",
		Text:         "Hello, world!", // Should be ignored in JSON
		JSONConfig: &TTSConfig{
			PaddingBonus: -0.5,
		},
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed["voice_id"] != "voice-123" {
		t.Errorf("expected voice_id 'voice-123', got %v", parsed["voice_id"])
	}
	if parsed["output_format"] != "wav" {
		t.Errorf("expected output_format 'wav', got %v", parsed["output_format"])
	}

	// Text should not be in JSON (has json:"-" tag)
	if _, ok := parsed["text"]; ok {
		t.Error("text should not be in JSON")
	}
}

func TestSTTParamsJSONMarshal(t *testing.T) {
	params := STTParams{
		InputFormat: InputFormatPCM,
		ModelName:   "whisper",
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed["input_format"] != "pcm" {
		t.Errorf("expected input_format 'pcm', got %v", parsed["input_format"])
	}
}

func TestSTTReadyInfoJSONUnmarshal(t *testing.T) {
	jsonData := `{
		"request_id": "req-123",
		"model_name": "whisper",
		"sample_rate": 24000,
		"frame_size": 1920,
		"delay_in_tokens": 5,
		"text_stream_names": ["stream1", "stream2"]
	}`

	var info STTReadyInfo
	if err := json.Unmarshal([]byte(jsonData), &info); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if info.RequestID != "req-123" {
		t.Errorf("expected RequestID 'req-123', got %q", info.RequestID)
	}
	if info.ModelName != "whisper" {
		t.Errorf("expected ModelName 'whisper', got %q", info.ModelName)
	}
	if info.SampleRate != 24000 {
		t.Errorf("expected SampleRate 24000, got %d", info.SampleRate)
	}
	if info.FrameSize != 1920 {
		t.Errorf("expected FrameSize 1920, got %d", info.FrameSize)
	}
	if info.DelayInTokens != 5 {
		t.Errorf("expected DelayInTokens 5, got %d", info.DelayInTokens)
	}
	if len(info.TextStreamNames) != 2 {
		t.Errorf("expected 2 TextStreamNames, got %d", len(info.TextStreamNames))
	}
}

func TestSTTTextResultJSONUnmarshal(t *testing.T) {
	jsonData := `{
		"text": "Hello world",
		"start_s": 0.5,
		"stream_id": 1
	}`

	var result STTTextResult
	if err := json.Unmarshal([]byte(jsonData), &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if result.Text != "Hello world" {
		t.Errorf("expected Text 'Hello world', got %q", result.Text)
	}
	if result.StartS != 0.5 {
		t.Errorf("expected StartS 0.5, got %f", result.StartS)
	}
	if result.StreamID == nil || *result.StreamID != 1 {
		t.Error("StreamID mismatch")
	}
}

func TestVADPredictionJSONUnmarshal(t *testing.T) {
	jsonData := `{
		"horizon_s": 0.5,
		"inactivity_prob": 0.95
	}`

	var pred VADPrediction
	if err := json.Unmarshal([]byte(jsonData), &pred); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if pred.HorizonS != 0.5 {
		t.Errorf("expected HorizonS 0.5, got %f", pred.HorizonS)
	}
	if pred.InactivityProb != 0.95 {
		t.Errorf("expected InactivityProb 0.95, got %f", pred.InactivityProb)
	}
}

func TestSTTStepResultJSONUnmarshal(t *testing.T) {
	jsonData := `{
		"vad": [{"horizon_s": 0.5, "inactivity_prob": 0.8}],
		"step_idx": 10,
		"step_duration_s": 0.08,
		"total_duration_s": 0.8
	}`

	var result STTStepResult
	if err := json.Unmarshal([]byte(jsonData), &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if result.StepIdx != 10 {
		t.Errorf("expected StepIdx 10, got %d", result.StepIdx)
	}
	if result.StepDurationS != 0.08 {
		t.Errorf("expected StepDurationS 0.08, got %f", result.StepDurationS)
	}
	if result.TotalDurationS != 0.8 {
		t.Errorf("expected TotalDurationS 0.8, got %f", result.TotalDurationS)
	}
	if len(result.VAD) != 1 {
		t.Errorf("expected 1 VAD prediction, got %d", len(result.VAD))
	}
}

func TestSTTEndTextResultJSONUnmarshal(t *testing.T) {
	jsonData := `{
		"stop_s": 5.5,
		"stream_id": 0
	}`

	var result STTEndTextResult
	if err := json.Unmarshal([]byte(jsonData), &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if result.StopS != 5.5 {
		t.Errorf("expected StopS 5.5, got %f", result.StopS)
	}
	if result.StreamID == nil || *result.StreamID != 0 {
		t.Error("StreamID mismatch")
	}
}

func TestVoiceUpdateParamsJSONMarshal(t *testing.T) {
	name := "Updated Name"
	desc := "Updated description"
	rank := 1.5

	params := VoiceUpdateParams{
		Name:        &name,
		Description: &desc,
		Rank:        &rank,
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed["name"] != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got %v", parsed["name"])
	}
	if parsed["description"] != "Updated description" {
		t.Errorf("expected description 'Updated description', got %v", parsed["description"])
	}
}

func TestVoiceUpdateParamsOmitEmpty(t *testing.T) {
	name := "Only Name"
	params := VoiceUpdateParams{
		Name: &name,
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if _, ok := parsed["description"]; ok {
		t.Error("description should be omitted when nil")
	}
	if _, ok := parsed["language"]; ok {
		t.Error("language should be omitted when nil")
	}
	if _, ok := parsed["tags"]; ok {
		t.Error("tags should be omitted when nil")
	}
	if _, ok := parsed["rank"]; ok {
		t.Error("rank should be omitted when nil")
	}
}

func TestTTSConfigJSONMarshal(t *testing.T) {
	config := TTSConfig{
		PaddingBonus: -0.5,
	}

	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed["padding_bonus"] != -0.5 {
		t.Errorf("expected padding_bonus -0.5, got %v", parsed["padding_bonus"])
	}
}

func TestTTSResultFields(t *testing.T) {
	result := TTSResult{
		RawData:    []byte("test audio data"),
		SampleRate: 48000,
		RequestID:  "req-123",
	}

	if string(result.RawData) != "test audio data" {
		t.Error("RawData mismatch")
	}
	if result.SampleRate != 48000 {
		t.Errorf("expected SampleRate 48000, got %d", result.SampleRate)
	}
	if result.RequestID != "req-123" {
		t.Errorf("expected RequestID 'req-123', got %q", result.RequestID)
	}
}

func TestVoiceCreateResponseJSONUnmarshal(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		checkFn  func(t *testing.T, resp VoiceCreateResponse)
	}{
		{
			name:     "successful creation",
			jsonData: `{"uid": "voice-123", "was_updated": false}`,
			checkFn: func(t *testing.T, resp VoiceCreateResponse) {
				if resp.UID == nil || *resp.UID != "voice-123" {
					t.Error("UID mismatch")
				}
				if resp.WasUpdated != false {
					t.Error("WasUpdated should be false")
				}
			},
		},
		{
			name:     "updated existing",
			jsonData: `{"uid": "voice-123", "was_updated": true}`,
			checkFn: func(t *testing.T, resp VoiceCreateResponse) {
				if !resp.WasUpdated {
					t.Error("WasUpdated should be true")
				}
			},
		},
		{
			name:     "with error",
			jsonData: `{"error": "Invalid audio format", "was_updated": false}`,
			checkFn: func(t *testing.T, resp VoiceCreateResponse) {
				if resp.Error == nil || *resp.Error != "Invalid audio format" {
					t.Error("Error mismatch")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp VoiceCreateResponse
			if err := json.Unmarshal([]byte(tt.jsonData), &resp); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}
			tt.checkFn(t, resp)
		})
	}
}

// Test WebSocket message types
func TestWSMessageJSONMarshal(t *testing.T) {
	msg := wsMessage{Type: "end_of_stream"}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed["type"] != "end_of_stream" {
		t.Errorf("expected type 'end_of_stream', got %v", parsed["type"])
	}
}

func TestTTSSetupMessageJSONMarshal(t *testing.T) {
	msg := ttsSetupMessage{
		Type:         "setup",
		VoiceID:      "voice-123",
		OutputFormat: FormatWAV,
		ModelName:    "default",
		JSONConfig: map[string]interface{}{
			"padding_bonus": -0.5,
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed["type"] != "setup" {
		t.Errorf("expected type 'setup', got %v", parsed["type"])
	}
	if parsed["voice_id"] != "voice-123" {
		t.Errorf("expected voice_id 'voice-123', got %v", parsed["voice_id"])
	}
	if parsed["output_format"] != "wav" {
		t.Errorf("expected output_format 'wav', got %v", parsed["output_format"])
	}
}

func TestSTTSetupMessageJSONMarshal(t *testing.T) {
	msg := sttSetupMessage{
		Type:        "setup",
		InputFormat: InputFormatPCM,
		ModelName:   "whisper",
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed["type"] != "setup" {
		t.Errorf("expected type 'setup', got %v", parsed["type"])
	}
	if parsed["input_format"] != "pcm" {
		t.Errorf("expected input_format 'pcm', got %v", parsed["input_format"])
	}
	if parsed["model_name"] != "whisper" {
		t.Errorf("expected model_name 'whisper', got %v", parsed["model_name"])
	}
}
