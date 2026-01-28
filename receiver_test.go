package gleanreceiver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestReceiverStartStop(t *testing.T) {
	cfg := &Config{
		Path: "/test",
	}
	cfg.ServerConfig.Endpoint = "localhost:19888"

	receiver, err := newGleanReceiver(
		cfg,
		receivertest.NewNopSettings(),
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
	cfg.ServerConfig.Endpoint = "localhost:19889"

	metricsSink := new(consumertest.MetricsSink)
	logsSink := new(consumertest.LogsSink)

	receiver, err := newGleanReceiver(
		cfg,
		receivertest.NewNopSettings(),
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
	resp, err := http.Get("http://localhost:19889/test")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
}

func TestReceiverHandleInvalidJSON(t *testing.T) {
	cfg := &Config{
		Path: "/test",
	}
	cfg.ServerConfig.Endpoint = "localhost:19890"

	metricsSink := new(consumertest.MetricsSink)
	logsSink := new(consumertest.LogsSink)

	receiver, err := newGleanReceiver(
		cfg,
		receivertest.NewNopSettings(),
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
	resp, err := http.Post(
		"http://localhost:19890/test",
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
	cfg.ServerConfig.Endpoint = "localhost:19891"

	metricsSink := new(consumertest.MetricsSink)
	logsSink := new(consumertest.LogsSink)

	receiver, err := newGleanReceiver(
		cfg,
		receivertest.NewNopSettings(),
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
			ClientID:   "test-client",
			AppBuild:   "1.0.0",
			OS:         "iOS",
			OSVersion:  "17.0",
		},
		PingInfo: PingInfo{
			Seq:       1,
			StartTime: time.Now(),
			EndTime:   time.Now().Add(time.Minute),
			PingType:  "metrics",
		},
		Metrics: map[string]interface{}{
			"counter": map[string]interface{}{
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

	resp, err := http.Post(
		"http://localhost:19891/test",
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
	cfg.ServerConfig.Endpoint = "localhost:19892"

	receiver, err := newGleanReceiver(
		cfg,
		receivertest.NewNopSettings(),
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
