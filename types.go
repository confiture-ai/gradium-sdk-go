package gradium

// OutputFormat represents audio output formats for TTS.
type OutputFormat string

// Output format constants for TTS.
const (
	FormatWAV      OutputFormat = "wav"
	FormatPCM      OutputFormat = "pcm"
	FormatOpus     OutputFormat = "opus"
	FormatULaw8000 OutputFormat = "ulaw_8000"
	FormatALaw8000 OutputFormat = "alaw_8000"
	FormatPCM16000 OutputFormat = "pcm_16000"
	FormatPCM24000 OutputFormat = "pcm_24000"
)

// InputFormat represents audio input formats for STT.
type InputFormat string

// Input format constants for STT.
const (
	InputFormatPCM  InputFormat = "pcm"
	InputFormatWAV  InputFormat = "wav"
	InputFormatOpus InputFormat = "opus"
)

// Voice represents a voice in the Gradium system.
type Voice struct {
	UID         string   `json:"uid"`
	Name        string   `json:"name"`
	Description *string  `json:"description,omitempty"`
	Language    *string  `json:"language,omitempty"`
	StartS      float64  `json:"start_s"`
	StopS       *float64 `json:"stop_s,omitempty"`
	Filename    string   `json:"filename"`
}

// VoiceCreateParams contains parameters for creating a voice.
type VoiceCreateParams struct {
	Name        string
	Description *string
	Language    *string
	StartS      float64
	TimeoutS    float64
	InputFormat string
}

// VoiceCreateResponse is the response from voice creation.
type VoiceCreateResponse struct {
	UID        *string `json:"uid,omitempty"`
	Error      *string `json:"error,omitempty"`
	WasUpdated bool    `json:"was_updated"`
}

// VoiceUpdateParams contains parameters for updating a voice.
type VoiceUpdateParams struct {
	Name        *string                  `json:"name,omitempty"`
	Description *string                  `json:"description,omitempty"`
	Language    *string                  `json:"language,omitempty"`
	StartS      *float64                 `json:"start_s,omitempty"`
	Tags        []map[string]interface{} `json:"tags,omitempty"`
	Rank        *float64                 `json:"rank,omitempty"`
}

// VoiceListParams contains parameters for listing voices.
type VoiceListParams struct {
	Skip           int
	Limit          int
	IncludeCatalog bool
}

// CreditsSummary contains credit balance information.
type CreditsSummary struct {
	RemainingCredits int     `json:"remaining_credits"`
	AllocatedCredits int     `json:"allocated_credits"`
	BillingPeriod    string  `json:"billing_period"`
	NextRolloverDate *string `json:"next_rollover_date,omitempty"`
	PlanName         string  `json:"plan_name"`
}

// TTSParams contains parameters for TTS requests.
type TTSParams struct {
	VoiceID      string       `json:"voice_id"`
	OutputFormat OutputFormat `json:"output_format"`
	ModelName    string       `json:"model_name,omitempty"`
	Text         string       `json:"-"` // Not sent in setup message
	JSONConfig   *TTSConfig   `json:"json_config,omitempty"`
}

// TTSConfig contains advanced TTS configuration.
type TTSConfig struct {
	// Speed control: negative = faster (-4.0 to -0.1), positive = slower (0.1 to 4.0)
	PaddingBonus float64 `json:"padding_bonus,omitempty"`
}

// TTSResult contains the result of a TTS request.
type TTSResult struct {
	RawData    []byte
	SampleRate int
	RequestID  string
}

// STTParams contains parameters for STT requests.
type STTParams struct {
	InputFormat InputFormat `json:"input_format"`
	ModelName   string      `json:"model_name,omitempty"`
}

// STTReadyInfo contains information sent when STT is ready.
type STTReadyInfo struct {
	RequestID       string   `json:"request_id"`
	ModelName       string   `json:"model_name"`
	SampleRate      int      `json:"sample_rate"`
	FrameSize       int      `json:"frame_size"`
	DelayInTokens   int      `json:"delay_in_tokens"`
	TextStreamNames []string `json:"text_stream_names"`
}

// STTTextResult contains a transcription result.
type STTTextResult struct {
	Text     string  `json:"text"`
	StartS   float64 `json:"start_s"`
	StreamID *int    `json:"stream_id,omitempty"`
}

// VADPrediction contains voice activity detection prediction.
type VADPrediction struct {
	HorizonS       float64 `json:"horizon_s"`
	InactivityProb float64 `json:"inactivity_prob"`
}

// STTStepResult contains VAD step information.
type STTStepResult struct {
	VAD            []VADPrediction `json:"vad"`
	StepIdx        int             `json:"step_idx"`
	StepDurationS  float64         `json:"step_duration_s"`
	TotalDurationS float64         `json:"total_duration_s"`
}

// STTEndTextResult contains end text information.
type STTEndTextResult struct {
	StopS    float64 `json:"stop_s"`
	StreamID *int    `json:"stream_id,omitempty"`
}

// WebSocket message types

type wsMessage struct {
	Type string `json:"type"`
}

type ttsSetupMessage struct {
	Type         string                 `json:"type"`
	VoiceID      string                 `json:"voice_id"`
	OutputFormat OutputFormat           `json:"output_format"`
	ModelName    string                 `json:"model_name"`
	JSONConfig   map[string]interface{} `json:"json_config,omitempty"`
}

type ttsTextMessage struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type ttsReadyMessage struct {
	Type      string `json:"type"`
	RequestID string `json:"request_id"`
}

type ttsAudioMessage struct {
	Type  string `json:"type"`
	Audio string `json:"audio"`
}

type ttsErrorMessage struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

type sttSetupMessage struct {
	Type        string      `json:"type"`
	InputFormat InputFormat `json:"input_format"`
	ModelName   string      `json:"model_name"`
}

type sttAudioMessage struct {
	Type  string `json:"type"`
	Audio string `json:"audio"`
}

type sttReadyMessage struct {
	Type            string   `json:"type"`
	RequestID       string   `json:"request_id"`
	ModelName       string   `json:"model_name"`
	SampleRate      int      `json:"sample_rate"`
	FrameSize       int      `json:"frame_size"`
	DelayInTokens   int      `json:"delay_in_tokens"`
	TextStreamNames []string `json:"text_stream_names"`
}

type sttTextMessage struct {
	Type     string  `json:"type"`
	Text     string  `json:"text"`
	StartS   float64 `json:"start_s"`
	StreamID *int    `json:"stream_id,omitempty"`
}

type sttStepMessage struct {
	Type           string          `json:"type"`
	VAD            []VADPrediction `json:"vad"`
	StepIdx        int             `json:"step_idx"`
	StepDurationS  float64         `json:"step_duration_s"`
	TotalDurationS float64         `json:"total_duration_s"`
}

type sttEndTextMessage struct {
	Type     string  `json:"type"`
	StopS    float64 `json:"stop_s"`
	StreamID *int    `json:"stream_id,omitempty"`
}

type sttErrorMessage struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}
