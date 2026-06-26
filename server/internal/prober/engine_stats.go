package prober

import "singbox-config-service/internal/pkg/types"

// GetStats returns a snapshot of prober state and configuration.
func (p *Prober) GetStats() map[string]interface{} {
	var totalNodes, onlineNodes, offlineNodes, timeoutNodes int

	p.results.Range(func(_, value interface{}) bool {
		result := value.(*types.ProbeResult)
		totalNodes++
		switch result.Status {
		case "online":
			onlineNodes++
		case "offline":
			offlineNodes++
		case "timeout":
			timeoutNodes++
		}
		return true
	})

	return map[string]interface{}{
		"running":      p.IsRunning(),
		"totalNodes":   totalNodes,
		"onlineNodes":  onlineNodes,
		"offlineNodes": offlineNodes,
		"timeoutNodes": timeoutNodes,
		"config": map[string]interface{}{
			"probeInterval":   p.config.Interval,
			"probeTimeout":    p.config.Timeout,
			"probeConcurrent": p.config.Concurrent,
			"maxResults":      p.config.MaxResults,
		},
	}
}
