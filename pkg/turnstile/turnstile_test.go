package turnstile

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTurnstile_Disabled(t *testing.T) {
	verifier := NewVerifier(Config{
		Enabled: false,
	})

	assert.False(t, verifier.IsEnabled())
	assert.Empty(t, verifier.GetSiteKey())

	ok, err := verifier.Verify(context.Background(), "", "")
	assert.NoError(t, err)
	assert.True(t, ok)
}

func TestTurnstile_MissingToken(t *testing.T) {
	verifier := NewVerifier(Config{
		Enabled:   true,
		SecretKey: "mock-secret",
	})

	assert.True(t, verifier.IsEnabled())

	ok, err := verifier.Verify(context.Background(), "   ", "")
	assert.ErrorIs(t, err, ErrMissingToken)
	assert.False(t, ok)
}

func TestTurnstile_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

		err := r.ParseForm()
		assert.NoError(t, err)
		assert.Equal(t, "test-secret", r.Form.Get("secret"))
		assert.Equal(t, "valid-token-123", r.Form.Get("response"))
		assert.Equal(t, "192.168.1.1", r.Form.Get("remoteip"))

		resp := SiteVerifyResponse{
			Success:     true,
			ChallengeTS: time.Now(),
			Hostname:    "example.com",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	verifier := NewVerifier(Config{
		SiteKey:   "site-key-123",
		SecretKey: "test-secret",
		Enabled:   true,
		VerifyURL: ts.URL,
		Timeout:   2 * time.Second,
	})

	assert.Equal(t, "site-key-123", verifier.GetSiteKey())
	assert.True(t, verifier.IsEnabled())

	ok, err := verifier.Verify(context.Background(), "valid-token-123", "192.168.1.1")
	assert.NoError(t, err)
	assert.True(t, ok)
}

func TestTurnstile_FailedVerification(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SiteVerifyResponse{
			Success:    false,
			ErrorCodes: []string{"invalid-input-response", "timeout-or-duplicate"},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	verifier := NewVerifier(Config{
		SecretKey: "test-secret",
		Enabled:   true,
		VerifyURL: ts.URL,
	})

	ok, err := verifier.Verify(context.Background(), "invalid-token", "")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrVerificationFailed)
	assert.Contains(t, err.Error(), "invalid-input-response")
	assert.False(t, ok)
}

func TestTurnstile_HttpErrorStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer ts.Close()

	verifier := NewVerifier(Config{
		SecretKey: "test-secret",
		Enabled:   true,
		VerifyURL: ts.URL,
	})

	ok, err := verifier.Verify(context.Background(), "token", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 502")
	assert.False(t, ok)
}

func TestTurnstile_ContextCancelled(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	verifier := NewVerifier(Config{
		SecretKey: "test-secret",
		Enabled:   true,
		VerifyURL: ts.URL,
		Timeout:   1 * time.Second,
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	ok, err := verifier.Verify(ctx, "token", "")
	require.Error(t, err)
	assert.False(t, ok)
}
