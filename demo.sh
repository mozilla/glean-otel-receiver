#!/bin/bash
#
# Glean OpenTelemetry Receiver Demo Script
#
# This script demonstrates the Glean receiver by:
# 1. Rebuilding the custom OpenTelemetry Collector
# 2. Starting the collector
# 3. Sending a test Glean ping
# 4. Outputting converted metrics and logs in JSON format
#
# Requirements:
# - OpenTelemetry Collector Builder (~/go/bin/builder)
# - Python 3 (for JSON formatting)
# - collector-config.yaml and example-ping.json in the current directory
#

set -e

# Kill any existing collectors
pkill -9 glean-otelcol 2>/dev/null || true
sleep 1

echo "🔧 Rebuilding OpenTelemetry Collector..."
rm -rf dist
gopath=$(go env GOPATH)

$gopath/bin/builder --config=builder-config.yaml > /dev/null 2>&1

echo "🚀 Starting collector..."
./dist/glean-otelcol --config=collector-config.yaml > collector.log 2>&1 &
COLLECTOR_PID=$!

# Wait for collector to start
echo "⏳ Waiting for collector to start..."
sleep 3

# Send test ping
echo "📤 Sending test Glean ping..."
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST http://localhost:9888/submit/telemetry/test-document-123 \
  -H "Content-Type: application/json" \
  -d @example-ping.json)

HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
if [ "$HTTP_CODE" = "200" ]; then
    echo "✅ Ping accepted (HTTP $HTTP_CODE)"
else
    echo "❌ Ping failed (HTTP $HTTP_CODE)"
fi

# Wait for processing
echo "⏳ Processing data..."
sleep 2

# Stop collector
echo "🛑 Stopping collector..."
kill $COLLECTOR_PID 2>/dev/null || true
wait $COLLECTOR_PID 2>/dev/null || true

# Extract and format output
echo ""
echo "📊 Converted Metrics Summary:"
echo "=============================="
METRICS_COUNT=$(grep -o "\"metrics\": [0-9]*" collector.log | head -1 | grep -o "[0-9]*")
DATAPOINTS_COUNT=$(grep -o "\"data points\": [0-9]*" collector.log | head -1 | grep -o "[0-9]*")
echo "Total Metrics: $METRICS_COUNT"
echo "Total Data Points: $DATAPOINTS_COUNT"

echo ""
echo "📝 Converted Logs Summary:"
echo "=========================="
LOGS_COUNT=$(grep -o "\"log records\": [0-9]*" collector.log | head -1 | grep -o "[0-9]*")
echo "Total Log Records: $LOGS_COUNT"

echo ""
echo "📋 Detailed Metrics (JSON format):"
echo "==================================="

# Parse and create JSON output
# Use jq for syntax coloring 
python3 - <<'PYTHON_SCRIPT' | jq -C
import re
import json

with open('collector.log', 'r') as f:
    log_content = f.read()

# Parse metrics
metrics = []
metric_blocks = re.findall(r'Metric #\d+.*?(?=Metric #|\nResourceLog|$)', log_content, re.DOTALL)

for block in metric_blocks:
    metric = {}

    name_match = re.search(r'-> Name: (.+)', block)
    if name_match:
        metric['name'] = name_match.group(1).strip()

    type_match = re.search(r'-> DataType: (\w+)', block)
    if type_match:
        metric['type'] = type_match.group(1).strip()

    # Handle different value types
    if 'Histogram' in block:
        count_match = re.search(r'Count: (\d+)', block)
        sum_match = re.search(r'Sum: ([\d.]+)', block)
        if count_match and sum_match:
            metric['count'] = int(count_match.group(1))
            metric['sum'] = float(sum_match.group(1))

        # Parse explicit bounds (bucket boundaries)
        bounds = []
        for match in re.finditer(r'ExplicitBounds #(\d+): ([\d.]+)', block):
            bounds.append(float(match.group(2)))
        if bounds:
            metric['explicit_bounds'] = bounds

        # Parse bucket counts
        counts = []
        for match in re.finditer(r'Buckets #(\d+), Count: (\d+)', block):
            counts.append(int(match.group(2)))
        if counts:
            metric['bucket_counts'] = counts
    else:
        value_match = re.search(r'^Value: ([\d.]+)$', block, re.MULTILINE)
        if value_match:
            value = value_match.group(1)
            metric['value'] = float(value) if '.' in value else int(value)

    if metric:
        metrics.append(metric)

# Parse events/logs
events = []
log_blocks = re.findall(r'LogRecord #\d+.*?(?=LogRecord #|Trace ID:|$)', log_content, re.DOTALL)

for block in log_blocks:
    event = {}

    body_match = re.search(r'Body: Str\((.+?)\)', block)
    if body_match:
        event['name'] = body_match.group(1)

    domain_match = re.search(r'-> event\.domain: Str\((.+?)\)', block)
    if domain_match:
        event['category'] = domain_match.group(1)

    timestamp_match = re.search(r'^Timestamp: (.+)$', block, re.MULTILINE)
    if timestamp_match and 'ObservedTimestamp' not in timestamp_match.group(0):
        event['timestamp'] = timestamp_match.group(1).strip()

    if event:
        events.append(event)

# Output JSON
output = {
    'metrics': metrics,
    'events': events
}

print(json.dumps(output, indent=2))
PYTHON_SCRIPT

echo ""
echo "📁 Full logs saved to: collector.log"
echo "✨ Demo complete!"
