package gleanreceiver

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestReceiverStartStop(t *testing.T) {
	cfg := &Config{
		Path: "/test",
	}
	cfg.ServerConfig.NetAddr.Endpoint = "localhost:19888"

	receiver, err := newGleanReceiver(
		cfg,
		receivertest.NewNopSettings(component.MustNewType("glean")),
		consumertest.NewNop(),
		consumertest.NewNop(),
	)
	require.NoError(t, err)

	ctx := context.Background()
	err = receiver.Start(ctx, componenttest.NewNopHost())
	require.NoError(t, err)

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	err = receiver.Shutdown(ctx)
	require.NoError(t, err)
}

func TestReceiverHandleInvalidMethod(t *testing.T) {
	cfg := &Config{
		Path: "/test",
	}
	cfg.ServerConfig.NetAddr.Endpoint = "localhost:19889"

	metricsSink := new(consumertest.MetricsSink)
	logsSink := new(consumertest.LogsSink)

	receiver, err := newGleanReceiver(
		cfg,
		receivertest.NewNopSettings(component.MustNewType("glean")),
		metricsSink,
		logsSink,
	)
	require.NoError(t, err)

	ctx := context.Background()
	err = receiver.Start(ctx, componenttest.NewNopHost())
	require.NoError(t, err)
	defer receiver.Shutdown(ctx)

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Try GET instead of POST
	// Path will be /test/{namespace}/{document_type}/{document_version}/{document_id} due to GetPath()
	resp, err := http.Get("http://localhost:19889/test/test-ns/test-type/1/test-doc-123")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
}

func TestReceiverHandleInvalidJSON(t *testing.T) {
	cfg := &Config{
		Path: "/test",
	}
	cfg.ServerConfig.NetAddr.Endpoint = "localhost:19890"

	metricsSink := new(consumertest.MetricsSink)
	logsSink := new(consumertest.LogsSink)

	receiver, err := newGleanReceiver(
		cfg,
		receivertest.NewNopSettings(component.MustNewType("glean")),
		metricsSink,
		logsSink,
	)
	require.NoError(t, err)

	ctx := context.Background()
	err = receiver.Start(ctx, componenttest.NewNopHost())
	require.NoError(t, err)
	defer receiver.Shutdown(ctx)

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Send invalid JSON
	// Path will be /test/{namespace}/{document_type}/{document_version}/{document_id} due to GetPath()
	resp, err := http.Post(
		"http://localhost:19890/test/test-ns/test-type/1/test-doc-123",
		"application/json",
		bytes.NewBufferString("{invalid json}"),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestReceiverHandleValidPing(t *testing.T) {
	cfg := &Config{
		Path: "/test",
	}
	cfg.ServerConfig.NetAddr.Endpoint = "localhost:19891"

	metricsSink := new(consumertest.MetricsSink)
	logsSink := new(consumertest.LogsSink)

	receiver, err := newGleanReceiver(
		cfg,
		receivertest.NewNopSettings(component.MustNewType("glean")),
		metricsSink,
		logsSink,
	)
	require.NoError(t, err)

	ctx := context.Background()
	err = receiver.Start(ctx, componenttest.NewNopHost())
	require.NoError(t, err)
	defer receiver.Shutdown(ctx)

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Create a valid Glean ping
	ping := GleanPing{
		ClientInfo: ClientInfo{
			ClientID:  "test-client",
			AppBuild:  "1.0.0",
			OS:        "iOS",
			OSVersion: "17.0",
		},
		PingInfo: PingInfo{
			Seq:       1,
			StartTime: time.Now(),
			EndTime:   time.Now().Add(time.Minute),
			PingType:  "metrics",
		},
		Metrics: map[string]any{
			"counter": map[string]any{
				"test_counter": float64(5),
			},
		},
		Events: []Event{
			{
				Timestamp: 1000,
				Category:  "test",
				Name:      "test_event",
				Extra: map[string]string{
					"key": "value",
				},
			},
		},
	}

	body, err := json.Marshal(ping)
	require.NoError(t, err)

	// Path will be /test/{namespace}/{document_type}/{document_version}/{document_id} due to GetPath()
	resp, err := http.Post(
		"http://localhost:19891/test/test-ns/test-type/1/test-doc-123",
		"application/json",
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Give time for async processing
	time.Sleep(100 * time.Millisecond)

	// Verify metrics were consumed
	assert.Eventually(t, func() bool {
		return len(metricsSink.AllMetrics()) > 0
	}, time.Second, 10*time.Millisecond)

	// Verify logs were consumed
	assert.Eventually(t, func() bool {
		return len(logsSink.AllLogs()) > 0
	}, time.Second, 10*time.Millisecond)
}

func TestReceiverMultipleStarts(t *testing.T) {
	cfg := &Config{
		Path: "/test",
	}
	cfg.ServerConfig.NetAddr.Endpoint = "localhost:19892"

	receiver, err := newGleanReceiver(
		cfg,
		receivertest.NewNopSettings(component.MustNewType("glean")),
		consumertest.NewNop(),
		consumertest.NewNop(),
	)
	require.NoError(t, err)

	ctx := context.Background()

	// Start multiple times - should only start once
	err = receiver.Start(ctx, componenttest.NewNopHost())
	require.NoError(t, err)

	err = receiver.Start(ctx, componenttest.NewNopHost())
	require.NoError(t, err)

	err = receiver.Shutdown(ctx)
	require.NoError(t, err)

	// Multiple shutdowns should be safe
	err = receiver.Shutdown(ctx)
	require.NoError(t, err)
}

// TestForwardRawPing tests successful forwarding to downstream
func TestForwardRawPing(t *testing.T) {
	var receivedBody []byte
	var receivedHeaders http.Header
	received := make(chan bool, 1)

	// Create mock downstream server
	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("Downstream received request: method=%s, url=%s", r.Method, r.URL.Path)

		// Verify request method
		assert.Equal(t, "POST", r.Method)

		// Capture headers
		receivedHeaders = r.Header.Clone()

		// Read and store body
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		receivedBody = body

		t.Logf("Downstream received body: %s", string(body))

		w.WriteHeader(http.StatusOK)
		received <- true
	}))
	defer downstream.Close()

	t.Logf("Downstream server URL: %s", downstream.URL)

	// Create receiver with forward URL
	cfg := &Config{
		Path:       "/test",
		ForwardURL: downstream.URL,
		ForwardHeaders: map[string]string{
			"X-Test-Header": "test-value",
			"Authorization": "Bearer test-token",
		},
	}
	cfg.ServerConfig.NetAddr.Endpoint = "localhost:19893"

	receiver, err := newGleanReceiver(
		cfg,
		receivertest.NewNopSettings(component.MustNewType("glean")),
		consumertest.NewNop(),
		consumertest.NewNop(),
	)
	require.NoError(t, err)
	require.NotNil(t, receiver.forwarder, "Forwarder should be created when ForwardURL is set")

	ctx := context.Background()
	err = receiver.Start(ctx, componenttest.NewNopHost())
	require.NoError(t, err)
	defer receiver.Shutdown(ctx)

	time.Sleep(100 * time.Millisecond)

	// Send test ping
	ping := GleanPing{
		ClientInfo: ClientInfo{ClientID: "test"},
		PingInfo:   PingInfo{Seq: 1, StartTime: time.Now(), EndTime: time.Now(), PingType: "metrics"},
	}
	body, err := json.Marshal(ping)
	require.NoError(t, err)

	resp, err := http.Post(
		"http://localhost:19893/test/glean/metrics/1/test-doc-123",
		"application/json",
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	t.Log("Request recieved successfully")

	// Wait for async forwarding to complete BEFORE closing response
	// Note: Forwarding happens asynchronously in a goroutine that uses req.Context()
	// The context is tied to the HTTP request lifecycle, so we need to wait before closing
	select {
	case <-received:
		// Forwarding completed successfully
		t.Log("Forwarding completed successfully")
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for downstream to receive ping")
	}

	resp.Body.Close()

	// Verify downstream received the raw ping
	assert.NotEmpty(t, receivedBody)

	var receivedPing GleanPing
	err = json.Unmarshal(receivedBody, &receivedPing)
	require.NoError(t, err)
	assert.Equal(t, "test", receivedPing.ClientInfo.ClientID)

	// Verify custom headers were sent
	assert.Equal(t, "test-value", receivedHeaders.Get("X-Test-Header"))
	assert.Equal(t, "Bearer test-token", receivedHeaders.Get("Authorization"))
	// Verify original Content-Type header was forwarded
	assert.Equal(t, "application/json", receivedHeaders.Get("Content-Type"))
}

// TestForwardRawPingNoConfig tests that forwarding is skipped when not configured
func TestForwardRawPingNoConfig(t *testing.T) {
	cfg := &Config{
		Path: "/test",
		// ForwardURL not set - forwarding should be skipped
	}
	cfg.ServerConfig.NetAddr.Endpoint = "localhost:19894"

	receiver, err := newGleanReceiver(
		cfg,
		receivertest.NewNopSettings(component.MustNewType("glean")),
		consumertest.NewNop(),
		consumertest.NewNop(),
	)
	require.NoError(t, err)

	ctx := context.Background()
	err = receiver.Start(ctx, componenttest.NewNopHost())
	require.NoError(t, err)
	defer receiver.Shutdown(ctx)

	time.Sleep(100 * time.Millisecond)

	// Send test ping
	ping := GleanPing{
		ClientInfo: ClientInfo{ClientID: "test"},
		PingInfo:   PingInfo{Seq: 1, StartTime: time.Now(), EndTime: time.Now(), PingType: "metrics"},
	}
	body, err := json.Marshal(ping)
	require.NoError(t, err)

	resp, err := http.Post(
		"http://localhost:19894/test/glean/metrics/1/test-doc-123",
		"application/json",
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should still succeed even without forwarding
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestForwardRawPingFailure tests error handling when downstream fails
func TestForwardRawPingFailure(t *testing.T) {
	// Create mock downstream server that returns error
	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("downstream error"))
	}))
	defer downstream.Close()

	cfg := &Config{
		Path:       "/test",
		ForwardURL: downstream.URL,
	}
	cfg.ServerConfig.NetAddr.Endpoint = "localhost:19895"

	receiver, err := newGleanReceiver(
		cfg,
		receivertest.NewNopSettings(component.MustNewType("glean")),
		consumertest.NewNop(),
		consumertest.NewNop(),
	)
	require.NoError(t, err)

	ctx := context.Background()
	err = receiver.Start(ctx, componenttest.NewNopHost())
	require.NoError(t, err)
	defer receiver.Shutdown(ctx)

	time.Sleep(100 * time.Millisecond)

	// Send test ping
	ping := GleanPing{
		ClientInfo: ClientInfo{ClientID: "test"},
		PingInfo:   PingInfo{Seq: 1, StartTime: time.Now(), EndTime: time.Now(), PingType: "metrics"},
	}
	body, err := json.Marshal(ping)
	require.NoError(t, err)

	resp, err := http.Post(
		"http://localhost:19895/test/glean/metrics/1/test-doc-123",
		"application/json",
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should still succeed (log and continue strategy)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestForwardRawPingTimeout tests timeout behavior
func TestForwardRawPingTimeout(t *testing.T) {
	// Create mock downstream server that delays response
	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Longer than timeout
		w.WriteHeader(http.StatusOK)
	}))
	defer downstream.Close()

	cfg := &Config{
		Path:           "/test",
		ForwardURL:     downstream.URL,
		ForwardTimeout: 100 * time.Millisecond, // Short timeout
	}
	cfg.ServerConfig.NetAddr.Endpoint = "localhost:19896"

	receiver, err := newGleanReceiver(
		cfg,
		receivertest.NewNopSettings(component.MustNewType("glean")),
		consumertest.NewNop(),
		consumertest.NewNop(),
	)
	require.NoError(t, err)

	ctx := context.Background()
	err = receiver.Start(ctx, componenttest.NewNopHost())
	require.NoError(t, err)
	defer receiver.Shutdown(ctx)

	time.Sleep(100 * time.Millisecond)

	// Send test ping
	ping := GleanPing{
		ClientInfo: ClientInfo{ClientID: "test"},
		PingInfo:   PingInfo{Seq: 1, StartTime: time.Now(), EndTime: time.Now(), PingType: "metrics"},
	}
	body, err := json.Marshal(ping)
	require.NoError(t, err)

	resp, err := http.Post(
		"http://localhost:19896/test/glean/metrics/1/test-doc-123",
		"application/json",
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should still succeed despite timeout (log and continue strategy)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
