package prober

import "sync"

// defaultMaxRetries is the fallback retry count when the config value is zero.
const defaultMaxRetries = 2

// nodeHistory tracks probe success history using a ring buffer.
type nodeHistory struct {
	mu      sync.Mutex
	results []bool
	index   int
	size    int
}

// update records a probe result and returns the success rate as a percentage.
func (h *nodeHistory) update(success bool) float64 {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.results[h.index] = success
	h.index = (h.index + 1) % h.size

	successCount := 0
	for _, r := range h.results {
		if r {
			successCount++
		}
	}
	return float64(successCount) / float64(h.size) * 100
}
