package subscription

import "singbox-config-service/internal/pkg/types"

type SubscriptionEntry struct {
	ID             string              `json:"id"`
	Name           string              `json:"name"`
	URL            string              `json:"url"`
	UserAgent      string              `json:"user_agent,omitempty"`
	AutoUpdate     bool                `json:"auto_update,omitempty"`
	UpdateInterval int                 `json:"update_interval,omitempty"`
	LastUpdated    string              `json:"last_updated,omitempty"`
	Nodes          []types.ProxyNode   `json:"nodes"`
}

type SubscriptionData struct {
	URL           string                `json:"url,omitempty"`
	Nodes         []types.ProxyNode     `json:"nodes,omitempty"`
	Subscriptions []SubscriptionEntry   `json:"subscriptions,omitempty"`
}
