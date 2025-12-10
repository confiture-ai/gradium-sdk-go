package gradium

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// TTSService handles text-to-speech operations.
type TTSService struct {
	client *Client
}

// TTSStream handles streaming TTS responses.
type TTSStream struct {
	conn      *websocket.Conn
	requestID string
	ready     chan struct{}
	done      chan struct{}
	err       error
	errMu     sync.RWMutex
	audioCh   chan []byte
	closeOnce sync.Once
}

// Create converts text to speech and returns the complete audio.
//
// Example:
//
//	result, err := client.TTS.Create(ctx, gradium.TTSParams{
//	    VoiceID:      "YTpq7expH9539ERJ",
//	    OutputFormat: gradium.FormatWAV,
//	    Text:         "Hello, world!",
//	})
//	os.WriteFile("output.wav", result.RawData, 0644)
func (s *TTSService) Create(ctx context.Context, params TTSParams) (*TTSResult, error) {
	stream, err := s.Stream(ctx, params)
	if err != nil {
		return nil, err
	}
	defer func() { _ = stream.Close() }()

	if err := stream.WaitReady(ctx); err != nil {
		return nil, err
	}

	if err := stream.SendText(params.Text); err != nil {
		return nil, err
	}

	if err := stream.SendEndOfStream(); err != nil {
		return nil, err
	}

	return stream.Collect(ctx)
}

// Stream creates a streaming TTS connection.
//
// Example:
//
//	stream, err := client.TTS.Stream(ctx, gradium.TTSParams{
//	    VoiceID:      "YTpq7expH9539ERJ",
//	    OutputFormat: gradium.FormatPCM,
//	})
//	defer stream.Close()
//
//	stream.WaitReady(ctx)
//	stream.SendText("Hello, world!")
//	stream.SendEndOfStream()
//
//	for chunk := range stream.Audio() {
//	    // Process audio chunk
//	}
func (s *TTSService) Stream(ctx context.Context, params TTSParams) (*TTSStream, error) {
	wsURL := s.client.wsURL + "/tts"

	header := http.Header{}
	header.Set("x-api-key", s.client.apiKey)

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, header)
	if err != nil {
		return nil, &ConnectionError{Message: "failed to connect to TTS WebSocket: " + err.Error()}
	}

	stream := &TTSStream{
		conn:    conn,
		ready:   make(chan struct{}),
		done:    make(chan struct{}),
		audioCh: make(chan []byte, 100),
	}

	// Send setup message
	modelName := params.ModelName
	if modelName == "" {
		modelName = modelNameDefault
	}

	setupMsg := ttsSetupMessage{
		Type:         "setup",
		VoiceID:      params.VoiceID,
		OutputFormat: params.OutputFormat,
		ModelName:    modelName,
	}

	if params.JSONConfig != nil {
		setupMsg.JSONConfig = map[string]interface{}{
			"padding_bonus": params.JSONConfig.PaddingBonus,
		}
	}

	if err := conn.WriteJSON(setupMsg); err != nil {
		_ = conn.Close()
		return nil, &WebSocketError{Message: "failed to send setup message: " + err.Error()}
	}

	// Start message handler
	go stream.handleMessages()

	return stream, nil
}

func (s *TTSStream) handleMessages() {
	defer close(s.done)
	defer close(s.audioCh)

	readySignaled := false

	for {
		_, data, err := s.conn.ReadMessage()
		if err != nil {
			s.setError(&WebSocketError{Message: "read error: " + err.Error()})
			if !readySignaled {
				close(s.ready)
			}
			return
		}

		var msg wsMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case msgTypeReady:
			var readyMsg ttsReadyMessage
			_ = json.Unmarshal(data, &readyMsg)
			s.requestID = readyMsg.RequestID
			if !readySignaled {
				close(s.ready)
				readySignaled = true
			}

		case "audio":
			var audioMsg ttsAudioMessage
			if err := json.Unmarshal(data, &audioMsg); err != nil {
				continue
			}
			decoded, err := base64.StdEncoding.DecodeString(audioMsg.Audio)
			if err != nil {
				continue
			}
			select {
			case s.audioCh <- decoded:
			default:
				// Channel full, drop audio
			}

		case msgTypeEndOfStream:
			return

		case msgTypeError:
			var errMsg ttsErrorMessage
			_ = json.Unmarshal(data, &errMsg)
			s.setError(&WebSocketError{Message: errMsg.Message, Code: errMsg.Code})
			if !readySignaled {
				close(s.ready)
			}
			return
		}
	}
}

func (s *TTSStream) setError(err error) {
	s.errMu.Lock()
	if s.err == nil {
		s.err = err
	}
	s.errMu.Unlock()
}

func (s *TTSStream) getError() error {
	s.errMu.RLock()
	defer s.errMu.RUnlock()
	return s.err
}

// WaitReady waits for the stream to be ready.
func (s *TTSStream) WaitReady(ctx context.Context) error {
	select {
	case <-s.ready:
		return s.getError()
	case <-ctx.Done():
		return ctx.Err()
	}
}

// SendText sends text to be converted to speech.
func (s *TTSStream) SendText(text string) error {
	msg := ttsTextMessage{Type: "text", Text: text}
	return s.conn.WriteJSON(msg)
}

// SendEndOfStream signals the end of input.
func (s *TTSStream) SendEndOfStream() error {
	return s.conn.WriteJSON(wsMessage{Type: msgTypeEndOfStream})
}

// Audio returns a channel that receives audio chunks.
func (s *TTSStream) Audio() <-chan []byte {
	return s.audioCh
}

// Collect waits for all audio and returns the complete result.
func (s *TTSStream) Collect(ctx context.Context) (*TTSResult, error) {
	var chunks [][]byte
	totalLen := 0

	for {
		select {
		case chunk, ok := <-s.audioCh:
			if !ok {
				// Channel closed, combine all chunks
				if err := s.getError(); err != nil {
					return nil, err
				}

				rawData := make([]byte, totalLen)
				offset := 0
				for _, c := range chunks {
					copy(rawData[offset:], c)
					offset += len(c)
				}

				return &TTSResult{
					RawData:    rawData,
					SampleRate: 48000,
					RequestID:  s.requestID,
				}, nil
			}
			chunks = append(chunks, chunk)
			totalLen += len(chunk)

		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// RequestID returns the request ID.
func (s *TTSStream) RequestID() string {
	return s.requestID
}

// Close closes the stream.
func (s *TTSStream) Close() error {
	var err error
	s.closeOnce.Do(func() {
		err = s.conn.Close()
	})
	return err
}

// Done returns a channel that's closed when the stream ends.
func (s *TTSStream) Done() <-chan struct{} {
	return s.done
}
