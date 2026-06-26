package prober

import "singbox-config-service/internal/pkg/types"

// GetResult returns a copy of the latest probe result for the given node tag.
func (p *Prober) GetResult(tag string) *types.ProbeResult {
	if result, ok := p.results.Load(tag); ok {
		r := result.(*types.ProbeResult)
		copy := *r
		return &copy
	}
	return nil
}

// GetAllResults returns copies of all probe results keyed by node tag.
func (p *Prober) GetAllResults() map[string]*types.ProbeResult {
	results := make(map[string]*types.ProbeResult)
	p.results.Range(func(key, value interface{}) bool {
		r := value.(*types.ProbeResult)
		copy := *r
		results[key.(string)] = &copy
		return true
	})
	return results
}

// GetBestNode returns the online node with the lowest latency.
func (p *Prober) GetBestNode() *types.ProbeResult {
	var best *types.ProbeResult
	p.results.Range(func(_, value interface{}) bool {
		result := value.(*types.ProbeResult)
		if result.Status == "online" && result.Latency > 0 {
			if best == nil || result.Latency < best.Latency {
				copy := *result
				best = &copy
			}
		}
		return true
	})
	return best
}

// GetOnlineNodes returns all nodes currently marked as online.
func (p *Prober) GetOnlineNodes() []*types.ProbeResult {
	var online []*types.ProbeResult
	p.results.Range(func(_, value interface{}) bool {
		result := value.(*types.ProbeResult)
		if result.Status == "online" {
			copy := *result
			online = append(online, &copy)
		}
		return true
	})
	return online
}
