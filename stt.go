package gradium

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

// WebSocket message type constants.
const (
	msgTypeReady       = "ready"
	msgTypeError       = "error"
	msgTypeEndOfStream = "end_of_stream"
	modelNameDefault   = "default"
)

// STTService handles speech-to-text operations.
type STTService struct {
	client *Client
}

// STTStream handles streaming STT responses.
type STTStream struct {
	conn        *websocket.Conn
	readyInfo   *STTReadyInfo
	readyInfoMu sync.RWMutex
	ready       chan struct{}
	done        chan struct{}
	err         error
	errMu       sync.RWMutex
	textCh      chan STTTextResult
	vadCh       chan STTStepResult
	endTextCh   chan STTEndTextResult
	allMsgCh    chan interface{}
	closeOnce   sync.Once
}

// Stream creates a streaming STT connection.
//
// Example:
//
//	stream, err := client.STT.Stream(ctx, gradium.STTParams{
//	    InputFormat: gradium.InputFormatPCM,
//	})
//	defer stream.Close()
//
//	info, _ := stream.WaitReady(ctx)
//	fmt.Printf("Sample rate: %d\n", info.SampleRate)
//
//	stream.SendAudio(audioChunk)
//	stream.SendEndOfStream()
//
//	for text := range stream.Text() {
//	    fmt.Printf("Transcription: %s\n", text.Text)
//	}
func (s *STTService) Stream(ctx context.Context, params STTParams) (*STTStream, error) {
	wsURL := s.client.wsURL + "/stt"

	header := http.Header{}
	header.Set("x-api-key", s.client.apiKey)

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, header)
	if err != nil {
		return nil, &ConnectionError{Message: "failed to connect to STT WebSocket: " + err.Error()}
	}

	stream := &STTStream{
		conn:      conn,
		ready:     make(chan struct{}),
		done:      make(chan struct{}),
		textCh:    make(chan STTTextResult, 100),
		vadCh:     make(chan STTStepResult, 100),
		endTextCh: make(chan STTEndTextResult, 10),
		allMsgCh:  make(chan interface{}, 100),
	}

	// Send setup message
	modelName := params.ModelName
	if modelName == "" {
		modelName = modelNameDefault
	}

	setupMsg := sttSetupMessage{
		Type:        "setup",
		InputFormat: params.InputFormat,
		ModelName:   modelName,
	}

	if err := conn.WriteJSON(setupMsg); err != nil {
		_ = conn.Close()
		return nil, &WebSocketError{Message: "failed to send setup message: " + err.Error()}
	}

	// Start message handler
	go stream.handleMessages()

	return stream, nil
}

// Transcribe transcribes complete audio data.
//
// Example:
//
//	audioData, _ := os.ReadFile("audio.wav")
//	text, err := client.STT.Transcribe(ctx, gradium.STTParams{
//	    InputFormat: gradium.InputFormatWAV,
//	}, audioData)
func (s *STTService) Transcribe(ctx context.Context, params STTParams, audio []byte) (string, error) {
	stream, err := s.Stream(ctx, params)
	if err != nil {
		return "", err
	}
	defer func() { _ = stream.Close() }()

	if _, err := stream.WaitReady(ctx); err != nil {
		return "", err
	}

	// Send audio in chunks (1920 samples = 80ms at 24kHz, 2 bytes per sample)
	chunkSize := 1920 * 2
	for i := 0; i < len(audio); i += chunkSize {
		end := i + chunkSize
		if end > len(audio) {
			end = len(audio)
		}
		if err := stream.SendAudio(audio[i:end]); err != nil {
			return "", err
		}
	}

	if err := stream.SendEndOfStream(); err != nil {
		return "", err
	}

	return stream.CollectText(ctx)
}

func (s *STTStream) handleMessages() {
	defer close(s.done)
	defer close(s.textCh)
	defer close(s.vadCh)
	defer close(s.endTextCh)
	defer close(s.allMsgCh)

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
			var readyMsg sttReadyMessage
			_ = json.Unmarshal(data, &readyMsg)
			s.readyInfoMu.Lock()
			s.readyInfo = &STTReadyInfo{
				RequestID:       readyMsg.RequestID,
				ModelName:       readyMsg.ModelName,
				SampleRate:      readyMsg.SampleRate,
				FrameSize:       readyMsg.FrameSize,
				DelayInTokens:   readyMsg.DelayInTokens,
				TextStreamNames: readyMsg.TextStreamNames,
			}
			s.readyInfoMu.Unlock()
			if !readySignaled {
				close(s.ready)
				readySignaled = true
			}

		case "text":
			var textMsg sttTextMessage
			if err := json.Unmarshal(data, &textMsg); err != nil {
				continue
			}
			result := STTTextResult{
				Text:     textMsg.Text,
				StartS:   textMsg.StartS,
				StreamID: textMsg.StreamID,
			}
			select {
			case s.textCh <- result:
			default:
			}
			select {
			case s.allMsgCh <- result:
			default:
			}

		case "step":
			var stepMsg sttStepMessage
			if err := json.Unmarshal(data, &stepMsg); err != nil {
				continue
			}
			result := STTStepResult{
				VAD:            stepMsg.VAD,
				StepIdx:        stepMsg.StepIdx,
				StepDurationS:  stepMsg.StepDurationS,
				TotalDurationS: stepMsg.TotalDurationS,
			}
			select {
			case s.vadCh <- result:
			default:
			}
			select {
			case s.allMsgCh <- result:
			default:
			}

		case "end_text":
			var endMsg sttEndTextMessage
			if err := json.Unmarshal(data, &endMsg); err != nil {
				continue
			}
			result := STTEndTextResult{
				StopS:    endMsg.StopS,
				StreamID: endMsg.StreamID,
			}
			select {
			case s.endTextCh <- result:
			default:
			}
			select {
			case s.allMsgCh <- result:
			default:
			}

		case msgTypeEndOfStream:
			return

		case msgTypeError:
			var errMsg sttErrorMessage
			_ = json.Unmarshal(data, &errMsg)
			s.setError(&WebSocketError{Message: errMsg.Message, Code: errMsg.Code})
			if !readySignaled {
				close(s.ready)
			}
			return
		}
	}
}

func (s *STTStream) setError(err error) {
	s.errMu.Lock()
	if s.err == nil {
		s.err = err
	}
	s.errMu.Unlock()
}

func (s *STTStream) getError() error {
	s.errMu.RLock()
	defer s.errMu.RUnlock()
	return s.err
}

// WaitReady waits for the stream to be ready and returns the ready info.
func (s *STTStream) WaitReady(ctx context.Context) (*STTReadyInfo, error) {
	select {
	case <-s.ready:
		if err := s.getError(); err != nil {
			return nil, err
		}
		return s.readyInfo, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// SendAudio sends audio data to be transcribed.
// Audio should be PCM 24kHz 16-bit mono.
func (s *STTStream) SendAudio(audio []byte) error {
	encoded := base64.StdEncoding.EncodeToString(audio)
	msg := sttAudioMessage{Type: "audio", Audio: encoded}
	return s.conn.WriteJSON(msg)
}

// SendEndOfStream signals the end of audio input.
func (s *STTStream) SendEndOfStream() error {
	return s.conn.WriteJSON(wsMessage{Type: msgTypeEndOfStream})
}

// Text returns a channel that receives transcription results.
func (s *STTStream) Text() <-chan STTTextResult {
	return s.textCh
}

// VAD returns a channel that receives voice activity detection results.
func (s *STTStream) VAD() <-chan STTStepResult {
	return s.vadCh
}

// EndText returns a channel that receives end text markers.
func (s *STTStream) EndText() <-chan STTEndTextResult {
	return s.endTextCh
}

// All returns a channel that receives all message types.
func (s *STTStream) All() <-chan interface{} {
	return s.allMsgCh
}

// CollectText waits for all text and returns the combined transcription.
func (s *STTStream) CollectText(ctx context.Context) (string, error) {
	var texts []string

	for {
		select {
		case text, ok := <-s.textCh:
			if !ok {
				if err := s.getError(); err != nil {
					return "", err
				}
				return strings.Join(texts, " "), nil
			}
			texts = append(texts, text.Text)

		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
}

// ReadyInfo returns the ready info (nil if not ready yet).
func (s *STTStream) ReadyInfo() *STTReadyInfo {
	s.readyInfoMu.RLock()
	defer s.readyInfoMu.RUnlock()
	return s.readyInfo
}

// Close closes the stream.
func (s *STTStream) Close() error {
	var err error
	s.closeOnce.Do(func() {
		err = s.conn.Close()
	})
	return err
}

// Done returns a channel that's closed when the stream ends.
func (s *STTStream) Done() <-chan struct{} {
	return s.done
}
