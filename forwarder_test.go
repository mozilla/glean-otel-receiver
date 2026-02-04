package gleanreceiver

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestGleanPingForwarderCreation(t *testing.T) {
	cfg := &Config{
		ForwardURL:     "http://example.com",
		ForwardTimeout: 30 * time.Second,
	}

	forwarder, err := newGleanPingForwarder(cfg, receivertest.NewNopSettings(component.MustNewType("glean")))
	require.NoError(t, err)
	require.NotNil(t, forwarder)
	assert.Equal(t, cfg, forwarder.cfg)
	assert.NotNil(t, forwarder.client)
	assert.Equal(t, 30*time.Second, forwarder.client.Timeout)
}

func TestGleanPingForwarderForward(t *testing.T) {
	received := make(chan bool, 1)
	var receivedBody []byte
	var receivedHeaders http.Header

	// Create mock downstream server
	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		body, _ := io.ReadAll(r.Body)
		receivedBody = body
		w.WriteHeader(http.StatusOK)
		received <- true
	}))
	defer downstream.Close()

	cfg := &Config{
		ForwardURL: downstream.URL,
		ForwardHeaders: map[string]string{
			"X-Custom": "value",
		},
		ForwardTimeout: 5 * time.Second,
	}

	forwarder, err := newGleanPingForwarder(cfg, receivertest.NewNopSettings(component.MustNewType("glean")))
	require.NoError(t, err)

	gleanReq := GleanPingRequest{
		Namespace:       "test-ns",
		DocumentType:    "metrics",
		DocumentVersion: "1",
		DocumentID:      "test-doc",
		Headers:         http.Header{"Content-Type": []string{"application/json"}},
	}

	body := []byte(`{"test": "data"}`)

	err = forwarder.forwardRawPing(context.Background(), gleanReq, body)
	require.NoError(t, err)

	// Wait for async forwarding
	select {
	case <-received:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for forwarding")
	}

	assert.Equal(t, body, receivedBody)
	assert.Equal(t, "value", receivedHeaders.Get("X-Custom"))
	assert.Equal(t, "application/json", receivedHeaders.Get("Content-Type"))
}

func TestGleanPingForwarderWithCanceledContext(t *testing.T) {
	received := make(chan bool, 1)

	// Create mock downstream server
	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.ReadAll(r.Body) // Consume body
		w.WriteHeader(http.StatusOK)
		received <- true
	}))
	defer downstream.Close()

	cfg := &Config{
		ForwardURL:     downstream.URL,
		ForwardTimeout: 5 * time.Second,
	}

	forwarder, err := newGleanPingForwarder(cfg, receivertest.NewNopSettings(component.MustNewType("glean")))
	require.NoError(t, err)

	gleanReq := GleanPingRequest{
		Namespace:       "test-ns",
		DocumentType:    "metrics",
		DocumentVersion: "1",
		DocumentID:      "test-doc",
		Headers:         http.Header{"Content-Type": []string{"application/json"}},
	}

	body := []byte(`{"test": "data"}`)

	// Create a context that gets canceled immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err = forwarder.forwardRawPing(ctx, gleanReq, body)
	require.NoError(t, err) // forwardRawPing returns nil even if context is canceled

	// Wait to see if forwarding happens despite canceled context
	select {
	case <-received:
		t.Log("Forwarding completed despite canceled context")
	case <-time.After(1 * time.Second):
		t.Log("Forwarding did not complete with canceled context (expected)")
	}
}
