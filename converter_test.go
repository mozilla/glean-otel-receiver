package gleanreceiver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

func TestConvertToMetrics(t *testing.T) {
	ping := &GleanPing{
		ClientInfo: ClientInfo{
			ClientID:          "test-client-id",
			AppBuild:          "1.0.0",
			AppDisplayVersion: "1.0",
			OS:                "iOS",
			OSVersion:         "17.0",
		},
		PingInfo: PingInfo{
			Seq:       1,
			StartTime: time.Now(),
			EndTime:   time.Now().Add(time.Minute),
			PingType:  "metrics",
			Reason:    "scheduled",
		},
		Metrics: map[string]interface{}{
			"counter": map[string]interface{}{
				"test_counter": float64(5),
			},
			"boolean": map[string]interface{}{
				"test_bool": true,
			},
			"string": map[string]interface{}{
				"test_string": "value",
			},
			"quantity": map[string]interface{}{
				"test_quantity": float64(100),
			},
		},
	}

	metrics, err := convertToMetrics(ping)
	require.NoError(t, err)
	assert.NotNil(t, metrics)

	// Verify resource attributes
	rm := metrics.ResourceMetrics().At(0)
	attrs := rm.Resource().Attributes()

	clientID, exists := attrs.Get("client.id")
	assert.True(t, exists)
	assert.Equal(t, "test-client-id", clientID.Str())

	osType, exists := attrs.Get("os.type")
	assert.True(t, exists)
	assert.Equal(t, "iOS", osType.Str())

	pingSeq, exists := attrs.Get("ping.seq")
	assert.True(t, exists)
	assert.Equal(t, int64(1), pingSeq.Int())

	// Verify metrics
	scopeMetrics := rm.ScopeMetrics().At(0)
	assert.Equal(t, "glean", scopeMetrics.Scope().Name())
	assert.Equal(t, 4, scopeMetrics.Metrics().Len())

	// Check counter metric
	foundCounter := false
	for i := 0; i < scopeMetrics.Metrics().Len(); i++ {
		metric := scopeMetrics.Metrics().At(i)
		if metric.Name() == "counter.test_counter" {
			foundCounter = true
			assert.Equal(t, pmetric.MetricTypeGauge, metric.Type())
			assert.Equal(t, 5.0, metric.Gauge().DataPoints().At(0).DoubleValue())
		}
	}
	assert.True(t, foundCounter)
}

func TestConvertToEventLogs(t *testing.T) {
	startTime := time.Date(2024, 1, 28, 10, 0, 0, 0, time.UTC)

	ping := &GleanPing{
		ClientInfo: ClientInfo{
			ClientID: "test-client-id",
		},
		PingInfo: PingInfo{
			Seq:       1,
			StartTime: startTime,
			EndTime:   startTime.Add(time.Minute),
			PingType:  "events",
		},
		Events: []Event{
			{
				Timestamp: 1000, // 1 second after start
				Category:  "ui",
				Name:      "button_clicked",
				Extra: map[string]string{
					"button_id": "submit",
					"screen":    "home",
				},
			},
			{
				Timestamp: 5000, // 5 seconds after start
				Category:  "navigation",
				Name:      "screen_view",
				Extra: map[string]string{
					"screen_name": "settings",
				},
			},
		},
	}

	logs, err := convertToEventLogs(ping)
	require.NoError(t, err)
	assert.NotNil(t, logs)

	// Verify resource attributes
	rl := logs.ResourceLogs().At(0)
	attrs := rl.Resource().Attributes()

	clientID, exists := attrs.Get("client.id")
	assert.True(t, exists)
	assert.Equal(t, "test-client-id", clientID.Str())

	// Verify logs
	scopeLogs := rl.ScopeLogs().At(0)
	assert.Equal(t, "glean", scopeLogs.Scope().Name())
	assert.Equal(t, 2, scopeLogs.LogRecords().Len())

	// Check first event
	log1 := scopeLogs.LogRecords().At(0)
	assert.Equal(t, "button_clicked", log1.Body().Str())

	eventName, exists := log1.Attributes().Get("event.name")
	assert.True(t, exists)
	assert.Equal(t, "button_clicked", eventName.Str())

	eventDomain, exists := log1.Attributes().Get("event.domain")
	assert.True(t, exists)
	assert.Equal(t, "ui", eventDomain.Str())

	buttonID, exists := log1.Attributes().Get("button_id")
	assert.True(t, exists)
	assert.Equal(t, "submit", buttonID.Str())

	// Verify timestamp (should be startTime + 1000ms)
	expectedTimestamp := startTime.Add(1000 * time.Millisecond)
	assert.Equal(t, expectedTimestamp.UnixNano(), log1.Timestamp().AsTime().UnixNano())
}

func TestConvertDistributionMetric(t *testing.T) {
	ping := &GleanPing{
		ClientInfo: ClientInfo{
			ClientID: "test-client",
		},
		PingInfo: PingInfo{
			Seq:       1,
			StartTime: time.Now(),
			EndTime:   time.Now().Add(time.Minute),
			PingType:  "metrics",
		},
		Metrics: map[string]interface{}{
			"timing_distribution": map[string]interface{}{
				"page_load": map[string]interface{}{
					"sum": float64(15000),
					"values": map[string]interface{}{
						"1000":  float64(10),
						"2000":  float64(25),
						"5000":  float64(40),
						"10000": float64(20),
						"20000": float64(5),
					},
				},
			},
		},
	}

	metrics, err := convertToMetrics(ping)
	require.NoError(t, err)

	scopeMetrics := metrics.ResourceMetrics().At(0).ScopeMetrics().At(0)
	metric := scopeMetrics.Metrics().At(0)

	assert.Equal(t, "timing_distribution.page_load", metric.Name())
	assert.Equal(t, pmetric.MetricTypeHistogram, metric.Type())

	histogram := metric.Histogram()
	dp := histogram.DataPoints().At(0)
	assert.Equal(t, 15000.0, dp.Sum())
	assert.Equal(t, uint64(100), dp.Count())
}

func TestConvertRateMetric(t *testing.T) {
	ping := &GleanPing{
		ClientInfo: ClientInfo{
			ClientID: "test-client",
		},
		PingInfo: PingInfo{
			Seq:       1,
			StartTime: time.Now(),
			EndTime:   time.Now().Add(time.Minute),
			PingType:  "metrics",
		},
		Metrics: map[string]interface{}{
			"rate": map[string]interface{}{
				"error_rate": map[string]interface{}{
					"numerator":   float64(5),
					"denominator": float64(127),
				},
			},
		},
	}

	metrics, err := convertToMetrics(ping)
	require.NoError(t, err)

	scopeMetrics := metrics.ResourceMetrics().At(0).ScopeMetrics().At(0)
	metric := scopeMetrics.Metrics().At(0)

	assert.Equal(t, "rate.error_rate.rate", metric.Name())
	assert.Equal(t, pmetric.MetricTypeGauge, metric.Type())

	gauge := metric.Gauge()
	dp := gauge.DataPoints().At(0)

	// Check calculated rate
	expectedRate := 5.0 / 127.0
	assert.InDelta(t, expectedRate, dp.DoubleValue(), 0.0001)

	// Check attributes
	num, exists := dp.Attributes().Get("numerator")
	assert.True(t, exists)
	assert.Equal(t, int64(5), num.Int())

	denom, exists := dp.Attributes().Get("denominator")
	assert.True(t, exists)
	assert.Equal(t, int64(127), denom.Int())
}

func TestBoolToFloat(t *testing.T) {
	assert.Equal(t, 1.0, boolToFloat(true))
	assert.Equal(t, 0.0, boolToFloat(false))
}

func TestToFloat64(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected float64
	}{
		{"float64", float64(3.14), 3.14},
		{"int", 42, 42.0},
		{"int64", int64(100), 100.0},
		{"string", "3.14", 3.14},
		{"invalid string", "abc", 0.0},
		{"nil", nil, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toFloat64(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
