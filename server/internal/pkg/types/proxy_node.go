// Package types defines shared data structures for proxy nodes, probes, and speed test results.
package types

// ProxyNode represents a proxy node with connection details and test results.
type ProxyNode struct {
	Name     string                 `json:"name"`
	Protocol string                 `json:"protocol"`
	Address  string                 `json:"address"`
	Port     int                    `json:"port"`
	Settings map[string]interface{} `json:"settings"`
	Outbound map[string]interface{} `json:"outbound"`
	Latency     int64   `json:"latency,omitempty"`
	Online      bool    `json:"online,omitempty"`
	LastProbe   string  `json:"last_probe,omitempty"`
	SuccessRate int     `json:"success_rate,omitempty"`
	SpeedKBps   float64 `json:"speed_kbps,omitempty"`
}

// ProbeNode identifies a node to be probed.
type ProbeNode struct {
	Tag      string `json:"tag"`
	Protocol string `json:"protocol"`
	Address  string `json:"address"`
	Port     int    `json:"port"`
}

// ProbeResult holds the result of a single node probe.
type ProbeResult struct {
	NodeTag     string  `json:"nodeTag"`
	Protocol    string  `json:"protocol"`
	Address     string  `json:"address"`
	Port        int     `json:"port"`
	Latency     int64   `json:"latency"`
	Status      string  `json:"status"`
	LastProbe   string  `json:"lastProbe"`
	FailCount   int     `json:"failCount"`
	SuccessRate float64 `json:"successRate"`
}

// ProbeResultUpdate represents an incremental probe result update to be saved.
type ProbeResultUpdate struct {
	Tag         string `json:"tag"`
	Latency     int64  `json:"latency"`
	Online      bool   `json:"online"`
	LastProbe   string `json:"last_probe"`
	SuccessRate int    `json:"success_rate"`
}

// SpeedTestResult contains the result of a completed speed test for a single node.
type SpeedTestResult struct {
	Tag       string  `json:"tag"`
	Name      string  `json:"name"`
	Status    string  `json:"status"`
	LatencyMs int64   `json:"latency_ms"`
	SpeedKBps float64 `json:"speed_kbps"`
	Error     string  `json:"error,omitempty"`
	TestedAt  string  `json:"tested_at,omitempty"`
}

// SpeedTestUpdate represents an incremental speed test result update to be saved.
type SpeedTestUpdate struct {
	Tag       string  `json:"tag"`
	Latency   int64   `json:"latency"`
	SpeedKBps float64 `json:"speed_kbps"`
	Online    bool    `json:"online"`
	LastProbe string  `json:"last_probe"`
}
