package speedtest

type SpeedTestState struct {
	Running       bool    `json:"running"`
	Tag           string  `json:"tag,omitempty"`
	Status        string  `json:"status,omitempty"`
	Progress      int     `json:"progress,omitempty"`
	DownloadSpeed float64 `json:"download_speed,omitempty"`
	LatencyMs     int64   `json:"latency_ms,omitempty"`
	Error         string  `json:"error,omitempty"`
}
