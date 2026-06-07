package subscription

import "singbox-config-service/internal/pkg/types"

// SubscriptionEntry represents a single subscription with its metadata and parsed proxy nodes.
type SubscriptionEntry struct {
	ID             string            `json:"id" example:"sub_abc123" description:"Unique subscription identifier"`
	Name           string            `json:"name" example:"My VPN Server" description:"Display name for the subscription"`
	URL            string            `json:"url" example:"https://example.com/sub" description:"Subscription URL to fetch"`
	UserAgent      string            `json:"user_agent,omitempty" example:"clash-meta" description:"HTTP User-Agent header"`
	AutoUpdate     bool              `json:"auto_update,omitempty" example:"true" description:"Enable or disable auto-update"`
	UpdateInterval int               `json:"update_interval,omitempty" example:"12" description:"Update interval in hours"`
	LastUpdated    string            `json:"last_updated,omitempty" example:"2025-01-15T10:30:00Z" description:"ISO 8601 timestamp of last update"`
	Nodes          []types.ProxyNode `json:"nodes" description:"Parsed proxy nodes from the subscription"`
}

// SubscriptionData is the top-level persistence structure containing all subscriptions.
type SubscriptionData struct {
	URL           string              `json:"url,omitempty" example:"https://example.com/sub" description:"Default subscription URL"`
	Nodes         []types.ProxyNode   `json:"nodes,omitempty" description:"Legacy direct nodes"`
	Subscriptions []SubscriptionEntry `json:"subscriptions,omitempty" description:"All managed subscriptions"`
}

// AddSubscriptionRequest request body for adding a subscription.
type AddSubscriptionRequest struct {
	Name      string `json:"name" example:"My VPN Server" binding:"required" description:"Display name for the subscription"`
	URL       string `json:"url" example:"https://example.com/sub" binding:"required" description:"Subscription URL to fetch"`
	UserAgent string `json:"user_agent" example:"clash-meta" description:"HTTP User-Agent header to use when fetching"`
}

// UpdateSettingsRequest request body for updating subscription settings.
type UpdateSettingsRequest struct {
	AutoUpdate     bool `json:"auto_update" example:"true" description:"Enable or disable auto-update"`
	UpdateInterval int  `json:"update_interval" example:"12" description:"Update interval in hours"`
}

// Config holds configuration parameters for subscription fetching.
type Config struct {
	InsecureTLS bool `json:"insecure_tls" yaml:"insecure_tls" example:"false"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{InsecureTLS: false}
}
