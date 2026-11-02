package turnstile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	ErrMissingToken       = errors.New("turnstile token is required")
	ErrVerificationFailed = errors.New("turnstile verification failed")
)

const (
	DefaultVerifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"
	DefaultTimeout   = 5 * time.Second
)

// Config holds Cloudflare Turnstile configuration options
type Config struct {
	SiteKey   string
	SecretKey string
	Enabled   bool
	VerifyURL string
	Timeout   time.Duration
}

// SiteVerifyResponse represents the response body returned by Cloudflare Turnstile
type SiteVerifyResponse struct {
	Success     bool      `json:"success"`
	ChallengeTS time.Time `json:"challenge_ts"`
	Hostname    string    `json:"hostname"`
	ErrorCodes  []string  `json:"error-codes"`
	Action      string    `json:"action"`
	CData       string    `json:"cdata"`
}

// Verifier defines the interface for verifying Cloudflare Turnstile tokens
type Verifier interface {
	Verify(ctx context.Context, token string, remoteIP string) (bool, error)
	IsEnabled() bool
	GetSiteKey() string
}

// Client implements Verifier using standard net/http
type Client struct {
	cfg        Config
	httpClient *http.Client
	verifyURL  string
}

// NewVerifier creates a new Turnstile Verifier client
func NewVerifier(cfg Config) Verifier {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	verifyURL := cfg.VerifyURL
	if verifyURL == "" {
		verifyURL = DefaultVerifyURL
	}

	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		verifyURL: verifyURL,
	}
}

// IsEnabled returns whether Turnstile verification is actively enforced
func (c *Client) IsEnabled() bool {
	return c.cfg.Enabled
}

// GetSiteKey returns the configured public SiteKey
func (c *Client) GetSiteKey() string {
	return c.cfg.SiteKey
}

// Verify validates the user response token against Cloudflare's siteverify API.
// If verification is disabled, it returns (true, nil) immediately.
func (c *Client) Verify(ctx context.Context, token string, remoteIP string) (bool, error) {
	if !c.cfg.Enabled {
		return true, nil
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return false, ErrMissingToken
	}

	formData := url.Values{}
	formData.Set("secret", c.cfg.SecretKey)
	formData.Set("response", token)
	if remoteIP != "" {
		formData.Set("remoteip", remoteIP)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.verifyURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return false, fmt.Errorf("failed to create turnstile request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("turnstile verification request error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("turnstile verification returned status %d", resp.StatusCode)
	}

	var res SiteVerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return false, fmt.Errorf("failed to decode turnstile response: %w", err)
	}

	if !res.Success {
		if len(res.ErrorCodes) > 0 {
			return false, fmt.Errorf("%w: %s", ErrVerificationFailed, strings.Join(res.ErrorCodes, ", "))
		}
		return false, ErrVerificationFailed
	}

	return true, nil
}
