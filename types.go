package gleanreceiver

import "time"

// GleanPing represents the top-level structure of a Glean telemetry ping
type GleanPing struct {
	ClientInfo ClientInfo             `json:"client_info"`
	PingInfo   PingInfo               `json:"ping_info"`
	Metrics    map[string]interface{} `json:"metrics,omitempty"`
	Events     []Event                `json:"events,omitempty"`
}

// ClientInfo contains information about the client device and application
type ClientInfo struct {
	ClientID            string `json:"client_id,omitempty"`
	SessionID           string `json:"session_id,omitempty"`
	SessionCount        int    `json:"session_count,omitempty"`
	AppBuild            string `json:"app_build,omitempty"`
	AppDisplayVersion   string `json:"app_display_version,omitempty"`
	AppChannel          string `json:"app_channel,omitempty"`
	TelemetrySDKBuild   string `json:"telemetry_sdk_build,omitempty"`
	Architecture        string `json:"architecture,omitempty"`
	DeviceManufacturer  string `json:"device_manufacturer,omitempty"`
	DeviceModel         string `json:"device_model,omitempty"`
	OS                  string `json:"os,omitempty"`
	OSVersion           string `json:"os_version,omitempty"`
	AndroidSDKVersion   string `json:"android_sdk_version,omitempty"`
	WindowsBuildNumber  int    `json:"windows_build_number,omitempty"`
	Locale              string `json:"locale,omitempty"`
	BuildDate           string `json:"build_date,omitempty"`
	FirstRunDate        string `json:"first_run_date,omitempty"`
}

// PingInfo contains metadata about the ping itself
type PingInfo struct {
	Seq       int       `json:"seq"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	PingType  string    `json:"ping_type"`
	Reason    string    `json:"reason,omitempty"`
}

// Event represents a Glean event
type Event struct {
	Timestamp int64             `json:"timestamp"`
	Category  string            `json:"category"`
	Name      string            `json:"name"`
	Extra     map[string]string `json:"extra,omitempty"`
}

// Distribution types
type Distribution struct {
	Sum    int64            `json:"sum"`
	Values map[string]int64 `json:"values"` // bucket -> count
}

// Rate metric
type Rate struct {
	Numerator   int `json:"numerator"`
	Denominator int `json:"denominator"`
}
