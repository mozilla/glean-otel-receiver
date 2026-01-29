package gleanreceiver

import (
	"fmt"
	"slices"
	"strconv"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

// convertToMetrics converts a Glean ping to OpenTelemetry metrics
func convertToMetrics(ping *GleanPing) (pmetric.Metrics, error) {
	metrics := pmetric.NewMetrics()
	rm := metrics.ResourceMetrics().AppendEmpty()

	// Add resource attributes from client_info
	addClientInfoAttributes(rm.Resource().Attributes(), &ping.ClientInfo)

	// Add ping_info attributes
	addPingInfoAttributes(rm.Resource().Attributes(), &ping.PingInfo)

	scopeMetrics := rm.ScopeMetrics().AppendEmpty()
	scopeMetrics.Scope().SetName("glean")

	// Process all metric categories
	if ping.Metrics != nil {
		if err := processMetrics(scopeMetrics, ping.Metrics); err != nil {
			return metrics, err
		}
	}

	return metrics, nil
}

// convertToEventLogs converts Glean events to OpenTelemetry event logs
func convertToEventLogs(ping *GleanPing) (plog.Logs, error) {
	logs := plog.NewLogs()
	rl := logs.ResourceLogs().AppendEmpty()

	// Add resource attributes from client_info
	addClientInfoAttributes(rl.Resource().Attributes(), &ping.ClientInfo)

	// Add ping_info attributes
	addPingInfoAttributes(rl.Resource().Attributes(), &ping.PingInfo)

	scopeLogs := rl.ScopeLogs().AppendEmpty()
	scopeLogs.Scope().SetName("glean")

	// Convert each event to an event log record
	for _, event := range ping.Events {
		logRecord := scopeLogs.LogRecords().AppendEmpty()

		// Set timestamp (Glean timestamps are in milliseconds since ping start)
		timestamp := ping.PingInfo.StartTime.Add(time.Duration(event.Timestamp) * time.Millisecond)
		logRecord.SetTimestamp(pcommon.NewTimestampFromTime(timestamp))

		// Set event name as body
		logRecord.Body().SetStr(event.Name)

		// Mark this as an event log using OpenTelemetry event conventions
		logRecord.Attributes().PutStr("event.name", event.Name)
		logRecord.Attributes().PutStr("event.domain", event.Category)

		// Add extra fields as attributes
		for k, v := range event.Extra {
			logRecord.Attributes().PutStr(k, v)
		}
	}

	return logs, nil
}

// addClientInfoAttributes adds client_info fields as resource attributes
func addClientInfoAttributes(attrs pcommon.Map, clientInfo *ClientInfo) {
	if clientInfo.ClientID != "" {
		attrs.PutStr("client.id", clientInfo.ClientID)
	}
	if clientInfo.SessionID != "" {
		attrs.PutStr("session.id", clientInfo.SessionID)
	}
	if clientInfo.SessionCount > 0 {
		attrs.PutInt("session.count", int64(clientInfo.SessionCount))
	}
	if clientInfo.AppBuild != "" {
		attrs.PutStr("service.version", clientInfo.AppBuild)
	}
	if clientInfo.AppDisplayVersion != "" {
		attrs.PutStr("app.version", clientInfo.AppDisplayVersion)
	}
	if clientInfo.AppChannel != "" {
		attrs.PutStr("app.channel", clientInfo.AppChannel)
	}
	if clientInfo.TelemetrySDKBuild != "" {
		attrs.PutStr("telemetry.sdk.version", clientInfo.TelemetrySDKBuild)
	}
	if clientInfo.Architecture != "" {
		attrs.PutStr("host.arch", clientInfo.Architecture)
	}
	if clientInfo.DeviceManufacturer != "" {
		attrs.PutStr("device.manufacturer", clientInfo.DeviceManufacturer)
	}
	if clientInfo.DeviceModel != "" {
		attrs.PutStr("device.model.name", clientInfo.DeviceModel)
	}
	if clientInfo.OS != "" {
		attrs.PutStr("os.type", clientInfo.OS)
	}
	if clientInfo.OSVersion != "" {
		attrs.PutStr("os.version", clientInfo.OSVersion)
	}
	if clientInfo.Locale != "" {
		attrs.PutStr("host.locale", clientInfo.Locale)
	}
}

// addPingInfoAttributes adds ping_info fields as resource attributes
func addPingInfoAttributes(attrs pcommon.Map, pingInfo *PingInfo) {
	attrs.PutInt("ping.seq", int64(pingInfo.Seq))
	attrs.PutStr("ping.type", pingInfo.PingType)
	if pingInfo.Reason != "" {
		attrs.PutStr("ping.reason", pingInfo.Reason)
	}
}

// processMetrics processes all metric categories and types
func processMetrics(scopeMetrics pmetric.ScopeMetrics, metricsMap map[string]interface{}) error {
	for category, categoryData := range metricsMap {
		categoryMap, ok := categoryData.(map[string]interface{})
		if !ok {
			continue
		}

		for metricName, metricValue := range categoryMap {
			fullName := fmt.Sprintf("%s.%s", category, metricName)

			switch v := metricValue.(type) {
			case bool:
				addGaugeMetric(scopeMetrics, fullName, boolToFloat(v))
			case float64:
				addGaugeMetric(scopeMetrics, fullName, v)
			case int64:
				addCounterMetric(scopeMetrics, fullName, v)
			case string:
				addStringMetric(scopeMetrics, fullName, v)
			case map[string]interface{}:
				// Handle complex metric types (distributions, rates, etc.)
				if err := processComplexMetric(scopeMetrics, fullName, v); err != nil {
					return err
				}
			case []interface{}:
				// Handle string lists
				addStringListMetric(scopeMetrics, fullName, v)
			}
		}
	}

	return nil
}

// processComplexMetric handles complex metric types like distributions and rates
func processComplexMetric(scopeMetrics pmetric.ScopeMetrics, name string, data map[string]interface{}) error {
	// Check if it's a distribution (has "sum" and "values")
	if sum, hasSum := data["sum"]; hasSum {
		if values, hasValues := data["values"]; hasValues {
			return addDistributionMetric(scopeMetrics, name, sum, values)
		}
	}

	// Check if it's a rate (has "numerator" and "denominator")
	if num, hasNum := data["numerator"]; hasNum {
		if denom, hasDenom := data["denominator"]; hasDenom {
			return addRateMetric(scopeMetrics, name, num, denom)
		}
	}

	// For other objects, create a gauge with JSON-encoded value
	return nil
}

// addGaugeMetric adds a gauge metric
func addGaugeMetric(scopeMetrics pmetric.ScopeMetrics, name string, value float64) {
	metric := scopeMetrics.Metrics().AppendEmpty()
	metric.SetName(name)
	metric.SetUnit("1")

	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	dp.SetDoubleValue(value)
}

// addCounterMetric adds a counter metric
func addCounterMetric(scopeMetrics pmetric.ScopeMetrics, name string, value int64) {
	metric := scopeMetrics.Metrics().AppendEmpty()
	metric.SetName(name)
	metric.SetUnit("1")

	sum := metric.SetEmptySum()
	sum.SetIsMonotonic(true)
	sum.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)

	dp := sum.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	dp.SetIntValue(value)
}

// addStringMetric adds a string value as a gauge metric with the string as an attribute
func addStringMetric(scopeMetrics pmetric.ScopeMetrics, name string, value string) {
	metric := scopeMetrics.Metrics().AppendEmpty()
	metric.SetName(name)
	metric.SetUnit("1")

	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	dp.SetIntValue(1)
	dp.Attributes().PutStr("value", value)
}

// addStringListMetric adds a string list as multiple data points
func addStringListMetric(scopeMetrics pmetric.ScopeMetrics, name string, values []interface{}) {
	metric := scopeMetrics.Metrics().AppendEmpty()
	metric.SetName(name)
	metric.SetUnit("1")

	gauge := metric.SetEmptyGauge()

	for i, val := range values {
		if strVal, ok := val.(string); ok {
			dp := gauge.DataPoints().AppendEmpty()
			dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
			dp.SetIntValue(1)
			dp.Attributes().PutStr("value", strVal)
			dp.Attributes().PutInt("index", int64(i))
		}
	}
}

// addDistributionMetric adds a distribution metric
func addDistributionMetric(scopeMetrics pmetric.ScopeMetrics, name string, sum any, values any) error {
	metric := scopeMetrics.Metrics().AppendEmpty()
	metric.SetName(name)
	metric.SetUnit("1")

	histogram := metric.SetEmptyHistogram()
	histogram.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)

	dp := histogram.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))

	// Set sum
	sumValue := toFloat64(sum)
	dp.SetSum(sumValue)

	// Process bucket values
	valuesMap, ok := values.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid distribution values format")
	}

	// Collect bucket/count pairs for sorting
	type bucketPair struct {
		boundary float64
		count    uint64
	}
	var pairs []bucketPair

	for b, c := range valuesMap {
		pairs = append(pairs, bucketPair{
			boundary: toFloat64(b),
			count:    uint64(toInt64(c)),
		})
	}

	// Sort by boundary to ensure deterministic ordering
	// Go maps have non-deterministic iteration order
	slices.SortFunc(pairs, func(a, b bucketPair) int {
		if a.boundary < b.boundary {
			return -1
		} else if a.boundary > b.boundary {
			return 1
		}
		return 0
	})

	// Extract sorted buckets and counts
	var totalCount uint64
	var buckets []float64
	var counts []uint64
	for _, p := range pairs {
		buckets = append(buckets, p.boundary)
		counts = append(counts, p.count)
		totalCount += p.count
	}

	dp.ExplicitBounds().FromRaw(buckets)
	dp.BucketCounts().FromRaw(counts)
	dp.SetCount(totalCount)

	return nil
}

// addRateMetric adds a rate metric as a gauge showing the ratio
func addRateMetric(scopeMetrics pmetric.ScopeMetrics, name string, numerator interface{}, denominator interface{}) error {
	num := toFloat64(numerator)
	denom := toFloat64(denominator)

	var rate float64
	if denom != 0 {
		rate = num / denom
	}

	metric := scopeMetrics.Metrics().AppendEmpty()
	metric.SetName(name + ".rate")
	metric.SetUnit("1")

	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	dp.SetDoubleValue(rate)
	dp.Attributes().PutInt("numerator", int64(num))
	dp.Attributes().PutInt("denominator", int64(denom))

	return nil
}

// Helper functions

func boolToFloat(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}

func toInt64(val any) int64 {
	switch v := val.(type) {
	case float64:
		return int64(v)
	case int:
		return int64(v)
	case uint:
		return int64(v)
	case int64:
		return v
	case string:
		i, _ := strconv.ParseInt(v, 0, 64)
		return i
	default:
		return 0
	}
}

func toFloat64(val interface{}) float64 {
	switch v := val.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		f, _ := strconv.ParseFloat(v, 64)
		return f
	default:
		return 0
	}
}
