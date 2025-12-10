package gradium

import (
	"context"
	"encoding/json"
	"net/http"
)

// CreditsService handles credit balance operations.
type CreditsService struct {
	client *Client
}

// Get returns the current credit balance for the authenticated user.
func (s *CreditsService) Get(ctx context.Context) (*CreditsSummary, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.client.baseURL+"/usages/credits", nil)
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

	var credits CreditsSummary
	if err := json.NewDecoder(resp.Body).Decode(&credits); err != nil {
		return nil, err
	}

	return &credits, nil
}
