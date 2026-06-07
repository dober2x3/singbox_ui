package prober

// Config holds configuration parameters for the prober engine.
// @Description Prober engine configuration
type Config struct {
	Interval       int    `json:"interval" yaml:"interval" example:"30"`
	Timeout        int    `json:"timeout" yaml:"timeout" example:"5000"`
	Concurrent     int    `json:"concurrent" yaml:"concurrent" example:"5"`
	MaxResults     int    `json:"max_results" yaml:"max_results" example:"100"`
	MaxRetries     int    `json:"max_retries" yaml:"max_retries" example:"2"`
	BindAddress    string `json:"bind_address,omitempty" yaml:"bind_address,omitempty" example:"192.168.1.100"`
	BindInterface  string `json:"bind_interface,omitempty" yaml:"bind_interface,omitempty" example:"eth0"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Interval:    30,
		Timeout:     5000,
		Concurrent:  5,
		MaxResults:  100,
		MaxRetries:  2,
	}
}

// ProberStatus contains the current prober status and statistics
// @Description Current prober engine status
type ProberStatus struct {
	Running       bool   `json:"running" example:"true"`
	TotalProbes   int    `json:"total_probes" example:"42"`
	OnlineNodes   int    `json:"online_nodes" example:"5"`
	OfflineNodes  int    `json:"offline_nodes" example:"3"`
	LastProbeTime string `json:"last_probe_time,omitempty" example:"2026-06-01T12:00:00Z"`
}

// MessageResponse generic message response
type MessageResponse struct {
	Message string `json:"message" example:"operation completed"`
}
