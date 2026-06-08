package scheduler

// Config holds configuration parameters for the scheduler.
type Config struct {
	Interval int `json:"interval" yaml:"interval" example:"60"` // seconds
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{Interval: 60}
}
