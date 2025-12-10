package gradium

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
)

// VoicesService handles voice management operations.
type VoicesService struct {
	client *Client
}

// List returns all voices for the authenticated organization.
func (s *VoicesService) List(ctx context.Context, params *VoiceListParams) ([]Voice, error) {
	url := s.client.baseURL + "/voices/"

	if params != nil {
		query := "?"
		if params.Skip > 0 {
			query += "skip=" + strconv.Itoa(params.Skip) + "&"
		}
		if params.Limit > 0 {
			query += "limit=" + strconv.Itoa(params.Limit) + "&"
		}
		if params.IncludeCatalog {
			query += "include_catalog=true&"
		}
		if len(query) > 1 {
			url += query[:len(query)-1] // Remove trailing &
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("x-api-key", s.client.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.httpClient.Do(req)
	if err != nil {
		return nil, &ConnectionError{Message: err.Error()}
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, handleAPIError(resp)
	}

	var voices []Voice
	if err := json.NewDecoder(resp.Body).Decode(&voices); err != nil {
		return nil, err
	}

	return voices, nil
}

// Get returns a specific voice by its UID.
func (s *VoicesService) Get(ctx context.Context, voiceUID string) (*Voice, error) {
	url := s.client.baseURL + "/voices/" + voiceUID

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("x-api-key", s.client.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.httpClient.Do(req)
	if err != nil {
		return nil, &ConnectionError{Message: err.Error()}
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, handleAPIError(resp)
	}

	var voice Voice
	if err := json.NewDecoder(resp.Body).Decode(&voice); err != nil {
		return nil, err
	}

	return &voice, nil
}

// Create creates a new custom voice from an audio file.
func (s *VoicesService) Create(ctx context.Context, audioData io.Reader, filename string, params VoiceCreateParams) (*VoiceCreateResponse, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add audio file
	part, err := writer.CreateFormFile("audio_file", filename)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, audioData); err != nil {
		return nil, err
	}

	// Add name
	if err := writer.WriteField("name", params.Name); err != nil {
		return nil, err
	}

	// Add optional fields
	if params.InputFormat != "" {
		if err := writer.WriteField("input_format", params.InputFormat); err != nil {
			return nil, err
		}
	}
	if params.Description != nil {
		if err := writer.WriteField("description", *params.Description); err != nil {
			return nil, err
		}
	}
	if params.Language != nil {
		if err := writer.WriteField("language", *params.Language); err != nil {
			return nil, err
		}
	}
	if params.StartS != 0 {
		if err := writer.WriteField("start_s", fmt.Sprintf("%f", params.StartS)); err != nil {
			return nil, err
		}
	}
	if params.TimeoutS != 0 {
		if err := writer.WriteField("timeout_s", fmt.Sprintf("%f", params.TimeoutS)); err != nil {
			return nil, err
		}
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.client.baseURL+"/voices/", &buf)
	if err != nil {
		return nil, err
	}

	req.Header.Set("x-api-key", s.client.apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := s.client.httpClient.Do(req)
	if err != nil {
		return nil, &ConnectionError{Message: err.Error()}
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		return nil, handleAPIError(resp)
	}

	var result VoiceCreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Update updates an existing voice.
func (s *VoicesService) Update(ctx context.Context, voiceUID string, params VoiceUpdateParams) (*Voice, error) {
	body, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, s.client.baseURL+"/voices/"+voiceUID, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("x-api-key", s.client.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.httpClient.Do(req)
	if err != nil {
		return nil, &ConnectionError{Message: err.Error()}
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, handleAPIError(resp)
	}

	var voice Voice
	if err := json.NewDecoder(resp.Body).Decode(&voice); err != nil {
		return nil, err
	}

	return &voice, nil
}

// Delete deletes a voice by its UID.
func (s *VoicesService) Delete(ctx context.Context, voiceUID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, s.client.baseURL+"/voices/"+voiceUID, nil)
	if err != nil {
		return err
	}

	req.Header.Set("x-api-key", s.client.apiKey)

	resp, err := s.client.httpClient.Do(req)
	if err != nil {
		return &ConnectionError{Message: err.Error()}
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent {
		return handleAPIError(resp)
	}

	return nil
}
