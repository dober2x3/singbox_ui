package speedtest

// SpeedTestState represents the current state of a speed test.
type SpeedTestState struct {
	Running       bool    `json:"running" example:"true"`
	Tag           string  `json:"tag,omitempty" example:"my-proxy"`
	Status        string  `json:"status,omitempty" example:"testing..."`
	Progress      int     `json:"progress,omitempty" example:"50"`
	DownloadSpeed float64 `json:"download_speed,omitempty" example:"15.5"`
	LatencyMs     int64   `json:"latency_ms,omitempty" example:"120"`
	Error         string  `json:"error,omitempty" example:"timeout"`
}
