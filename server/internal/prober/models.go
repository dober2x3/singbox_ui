package prober

// ProberConfig holds configuration parameters for the prober engine.
// @Description Prober engine configuration
type ProberConfig struct {
	ProbeInterval   int    `json:"probe_interval" example:"60"`    // seconds
	ProbeTimeout    int    `json:"probe_timeout" example:"5000"`   // ms
	ProbeConcurrent int    `json:"probe_concurrent" example:"10"`  // max concurrent probes
	MaxResults      int    `json:"max_results" example:"100"`      // ring buffer size
	BindAddress     string `json:"bind_address,omitempty" example:"192.168.1.100"`     // local IP to bind probes to (bypasses tunnel)
	BindInterface   string `json:"bind_interface,omitempty" example:"eth0"`             // network interface to bind probes to (requires root/CAP_NET_ADMIN)
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
