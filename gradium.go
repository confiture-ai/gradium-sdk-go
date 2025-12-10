package gradium

import (
	"net/http"
	"os"
	"strings"
	"time"
)

// Region represents the API region.
type Region string

// API region constants.
const (
	RegionEU Region = "eu"
	RegionUS Region = "us"
)

var apiURLs = map[Region]string{
	RegionEU: "https://eu.api.gradium.ai/api",
	RegionUS: "https://us.api.gradium.ai/api",
}

var wsURLs = map[Region]string{
	RegionEU: "wss://eu.api.gradium.ai/api/speech",
	RegionUS: "wss://us.api.gradium.ai/api/speech",
}

// ClientOption configures the Client.
type ClientOption func(*Client)

// WithAPIKey sets the API key for authentication.
// If not provided, the client reads from GRADIUM_API_KEY environment variable.
func WithAPIKey(apiKey string) ClientOption {
	return func(c *Client) {
		c.apiKey = apiKey
	}
}

// WithRegion sets the API region.
func WithRegion(region Region) ClientOption {
	return func(c *Client) {
		c.region = region
		c.baseURL = apiURLs[region]
		c.wsURL = wsURLs[region]
	}
}

// WithBaseURL sets a custom base URL.
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		baseURL = strings.TrimSuffix(baseURL, "/")
		c.baseURL = baseURL
		c.wsURL = strings.Replace(baseURL, "https://", "wss://", 1) + "/speech"
		c.wsURL = strings.Replace(c.wsURL, "http://", "ws://", 1)
	}
}

// WithTimeout sets the HTTP request timeout.
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.timeout = timeout
		c.httpClient.Timeout = timeout
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// Client is the Gradium API client.
type Client struct {
	apiKey     string
	region     Region
	baseURL    string
	wsURL      string
	timeout    time.Duration
	httpClient *http.Client

	// Resources
	TTS     *TTSService
	STT     *STTService
	Voices  *VoicesService
	Credits *CreditsService
}

// NewClient creates a new Gradium client.
func NewClient(opts ...ClientOption) (*Client, error) {
	c := &Client{
		region:  RegionEU,
		baseURL: apiURLs[RegionEU],
		wsURL:   wsURLs[RegionEU],
		timeout: 30 * time.Second,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	// If no API key was set via options, read from environment
	if c.apiKey == "" {
		c.apiKey = os.Getenv("GRADIUM_API_KEY")
	}
	if c.apiKey == "" {
		return nil, &AuthenticationError{Message: "API key is required. Use WithAPIKey option or set GRADIUM_API_KEY environment variable."}
	}

	// Initialize services
	c.TTS = &TTSService{client: c}
	c.STT = &STTService{client: c}
	c.Voices = &VoicesService{client: c}
	c.Credits = &CreditsService{client: c}

	return c, nil
}

// APIKey returns the API key.
func (c *Client) APIKey() string {
	return c.apiKey
}

// BaseURL returns the base URL.
func (c *Client) BaseURL() string {
	return c.baseURL
}

// WSURL returns the WebSocket URL.
func (c *Client) WSURL() string {
	return c.wsURL
}
