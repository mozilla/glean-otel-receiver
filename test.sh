#!/bin/bash

# Test script for Glean OpenTelemetry Receiver

set -e

echo "Starting collector in background..."
./dist/glean-otelcol --config=collector-config.yaml &
COLLECTOR_PID=$!

# Wait for collector to start
sleep 3

echo "Sending test Glean ping..."
curl -X POST http://localhost:9888/submit/telemetry \
  -H "Content-Type: application/json" \
  -d @example-ping.json

echo -e "\n\nTest completed! Check the collector logs above."
echo "Press Ctrl+C to stop the collector, or run: kill $COLLECTOR_PID"
