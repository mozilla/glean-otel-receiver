package gleanreceiver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestNewFactory(t *testing.T) {
	factory := NewFactory()

	assert.NotNil(t, factory)
	assert.Equal(t, component.MustNewType("glean"), factory.Type())
}

func TestCreateMetricsReceiver(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	receiver, err := factory.CreateMetrics(
		context.Background(),
		receivertest.NewNopSettings(),
		cfg,
		consumertest.NewNop(),
	)

	require.NoError(t, err)
	assert.NotNil(t, receiver)
}

func TestCreateLogsReceiver(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	receiver, err := factory.CreateLogs(
		context.Background(),
		receivertest.NewNopSettings(),
		cfg,
		consumertest.NewNop(),
	)

	require.NoError(t, err)
	assert.NotNil(t, receiver)
}

func TestCreateReceiverWithInvalidConfig(t *testing.T) {
	factory := NewFactory()
	cfg := &Config{
		Path: "", // Invalid: empty path
	}

	// Should still create receiver (validation happens separately)
	receiver, err := factory.CreateMetrics(
		context.Background(),
		receivertest.NewNopSettings(),
		cfg,
		consumertest.NewNop(),
	)

	require.NoError(t, err)
	assert.NotNil(t, receiver)
}

func TestSharedReceiverInstance(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	settings := receivertest.NewNopSettings()

	// Create metrics receiver
	metricsReceiver, err := factory.CreateMetrics(
		context.Background(),
		settings,
		cfg,
		consumertest.NewNop(),
	)
	require.NoError(t, err)

	// Create logs receiver with same settings
	logsReceiver, err := factory.CreateLogs(
		context.Background(),
		settings,
		cfg,
		consumertest.NewNop(),
	)
	require.NoError(t, err)

	// Should be the same instance
	assert.Same(t, metricsReceiver, logsReceiver)
}
