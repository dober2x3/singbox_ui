package prober

// ProberConfig holds configuration parameters for the prober engine.
type ProberConfig struct {
	ProbeInterval   int `json:"probe_interval"`   // seconds
	ProbeTimeout    int `json:"probe_timeout"`    // ms
	ProbeConcurrent int `json:"probe_concurrent"` // max concurrent probes
	MaxResults      int `json:"max_results"`      // ring buffer size
}
