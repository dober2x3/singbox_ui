// Package subscription provides subscription management including fetching,
// parsing, and storing proxy node subscriptions.
package subscription

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"strings"
	"time"

	"singbox-config-service/internal/pkg/types"
)

// Service manages subscription operations including fetching, adding,
// updating, deleting, and refreshing proxy node subscriptions.
type Service struct {
	store *FileStore
}

// NewService creates a new Service backed by the given FileStore.
func NewService(store *FileStore) *Service {
	return &Service{store: store}
}

// generateID returns a unique subscription ID based on the current Unix nanosecond timestamp.
func generateID() string {
	return fmt.Sprintf("sub_%d", time.Now().UnixNano())
}

// allowInsecureTLS returns true if the SUBSCRIPTION_INSECURE_TLS env var is set to a truthy value.
func allowInsecureTLS() bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv("SUBSCRIPTION_INSECURE_TLS")))
	return value == "1" || value == "true" || value == "yes"
}

// isPublicAddr checks whether the IP is a public address not in the blocked prefix list.
func isPublicAddr(ip net.IP) bool {
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return false
	}
	addr = addr.Unmap()
	for _, prefix := range types.BlockedSubscriptionPrefixes {
		if prefix.Contains(addr) {
			return false
		}
	}
	return true
}

// validateHost ensures the host is non-empty, not localhost, and resolves to a public IP address.
func validateHost(host string) error {
	normalizedHost := strings.Trim(strings.TrimSpace(host), "[]")
	if normalizedHost == "" {
		return fmt.Errorf("subscription URL host is empty")
	}
	if strings.EqualFold(normalizedHost, "localhost") {
		return fmt.Errorf("subscription host localhost is not allowed")
	}
	if ip := net.ParseIP(normalizedHost); ip != nil {
		if !isPublicAddr(ip) {
			return fmt.Errorf("subscription host %s is not a public address", normalizedHost)
		}
		return nil
	}
	ips, err := net.LookupIP(normalizedHost)
	if err != nil {
		return fmt.Errorf("failed to resolve subscription host %s: %w", normalizedHost, err)
	}
	if len(ips) == 0 {
		return fmt.Errorf("subscription host %s resolves to no address", normalizedHost)
	}
	for _, ip := range ips {
		if !isPublicAddr(ip) {
			return fmt.Errorf("subscription host %s resolves to non-public address %s", normalizedHost, ip.String())
		}
	}
	return nil
}

// validateURL checks that the parsed URL uses http/https and has a valid public host.
func validateURL(parsedURL *url.URL) error {
	if parsedURL == nil {
		return fmt.Errorf("subscription URL is nil")
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("unsupported URL scheme: %s (only http/https allowed)", parsedURL.Scheme)
	}
	return validateHost(parsedURL.Hostname())
}

// FetchSubscription fetches a subscription URL, validates the response, and parses it into proxy nodes.
func (s *Service) FetchSubscription(subURL string, userAgent ...string) ([]types.ProxyNode, error) {
	parsedURL, err := url.Parse(subURL)
	if err != nil {
		return nil, fmt.Errorf("invalid subscription URL: %w", err)
	}
	if err := validateURL(parsedURL); err != nil {
		return nil, err
	}

	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: allowInsecureTLS(),
		},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("too many redirects while fetching subscription")
			}
			return validateURL(req.URL)
		},
	}

	req, err := http.NewRequest("GET", parsedURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	ua := types.ResolveUserAgent("")
	if len(userAgent) > 0 && userAgent[0] != "" {
		ua = types.ResolveUserAgent(userAgent[0])
	}
	req.Header.Set("User-Agent", ua)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch subscription: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("subscription returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read subscription: %w", err)
	}

	content := strings.TrimSpace(string(body))
	if isClashYAML(content) {
		return parseClashYAML(body)
	}

	decoded, err := decodeBase64(content)
	if err != nil {
		decoded = body
	}

	return parseProxyLines(string(decoded))
}

// AddSubscription fetches a subscription URL and stores it as a new entry. Returns the created entry.
func (s *Service) AddSubscription(name, subURL, userAgent string) (*SubscriptionEntry, error) {
	data, err := s.store.Load()
	if err != nil {
		return nil, err
	}

	nodes, err := s.FetchSubscription(subURL, userAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch subscription: %w", err)
	}

	entry := SubscriptionEntry{
		ID:          generateID(),
		Name:        name,
		URL:         subURL,
		UserAgent:   userAgent,
		LastUpdated: time.Now().Format(time.RFC3339),
		Nodes:       nodes,
	}

	data.Subscriptions = append(data.Subscriptions, entry)
	if err := s.store.Save(*data); err != nil {
		return nil, err
	}

	return &entry, nil
}

// UpdateSubscription re-fetches the subscription URL for the given ID and updates its nodes.
func (s *Service) UpdateSubscription(id string) (*SubscriptionEntry, error) {
	data, err := s.store.Load()
	if err != nil {
		return nil, err
	}

	for i, sub := range data.Subscriptions {
		if sub.ID == id {
			nodes, err := s.FetchSubscription(sub.URL, sub.UserAgent)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch subscription: %w", err)
			}
			data.Subscriptions[i].Nodes = nodes
			data.Subscriptions[i].LastUpdated = time.Now().Format(time.RFC3339)

			if err := s.store.Save(*data); err != nil {
				return nil, err
			}
			return &data.Subscriptions[i], nil
		}
	}

	return nil, fmt.Errorf("subscription not found: %s", id)
}

// UpdateSubscriptionSettings updates the auto-update and interval settings for a subscription.
func (s *Service) UpdateSubscriptionSettings(id string, autoUpdate bool, updateInterval int) (*SubscriptionEntry, error) {
	data, err := s.store.Load()
	if err != nil {
		return nil, err
	}

	for i, sub := range data.Subscriptions {
		if sub.ID == id {
			data.Subscriptions[i].AutoUpdate = autoUpdate
			data.Subscriptions[i].UpdateInterval = updateInterval
			if err := s.store.Save(*data); err != nil {
				return nil, err
			}
			return &data.Subscriptions[i], nil
		}
	}

	return nil, fmt.Errorf("subscription not found: %s", id)
}

// DeleteSubscription removes the subscription entry with the given ID from the store.
func (s *Service) DeleteSubscription(id string) error {
	data, err := s.store.Load()
	if err != nil {
		return err
	}

	for i, sub := range data.Subscriptions {
		if sub.ID == id {
			data.Subscriptions = append(data.Subscriptions[:i], data.Subscriptions[i+1:]...)
			return s.store.Save(*data)
		}
	}

	return fmt.Errorf("subscription not found: %s", id)
}

// GetAllSubscriptions returns all subscription data from the store.
func (s *Service) GetAllSubscriptions() (*SubscriptionData, error) {
	return s.store.Load()
}

// LoadAll returns all subscription entries (for scheduler integration)
func (s *Service) LoadAll() ([]SubscriptionEntry, error) {
	data, err := s.store.Load()
	if err != nil {
		return nil, err
	}
	return data.Subscriptions, nil
}

// UpdateOne refreshes a single subscription (for scheduler integration)
func (s *Service) UpdateOne(id string) (*SubscriptionEntry, error) {
	return s.UpdateSubscription(id)
}

// GetAllNodes returns all proxy nodes across every subscription.
func (s *Service) GetAllNodes() ([]types.ProxyNode, error) {
	data, err := s.store.Load()
	if err != nil {
		return nil, err
	}

	var allNodes []types.ProxyNode
	for _, sub := range data.Subscriptions {
		allNodes = append(allNodes, sub.Nodes...)
	}
	return allNodes, nil
}

// RefreshAllSubscriptions re-fetches all subscriptions and updates their nodes. Returns the refreshed data.
func (s *Service) RefreshAllSubscriptions() (*SubscriptionData, error) {
	data, err := s.store.Load()
	if err != nil {
		return nil, err
	}

	for i, sub := range data.Subscriptions {
		nodes, err := s.FetchSubscription(sub.URL, sub.UserAgent)
		if err != nil {
			continue
		}
		data.Subscriptions[i].Nodes = nodes
	}

	if err := s.store.Save(*data); err != nil {
		return nil, err
	}

	return data, nil
}

// SaveProbeResults stores probe latency results into matching nodes across all subscriptions.
func (s *Service) SaveProbeResults(results []types.ProbeResultUpdate) error {
	data, err := s.store.Load()
	if err != nil {
		return err
	}

	resultMap := make(map[string]types.ProbeResultUpdate)
	for _, r := range results {
		resultMap[r.Tag] = r
	}

	for i := range data.Subscriptions {
		for j := range data.Subscriptions[i].Nodes {
			node := &data.Subscriptions[i].Nodes[j]
			tag := ""
			if node.Outbound != nil {
				if t, ok := node.Outbound["tag"].(string); ok {
					tag = t
				}
			}
			if tag == "" {
				tag = types.SanitizeTag(node.Protocol, node.Address, node.Port)
			}
			if result, exists := resultMap[tag]; exists {
				node.Latency = result.Latency
				node.Online = result.Online
				node.LastProbe = result.LastProbe
				node.SuccessRate = result.SuccessRate
			}
		}
	}

	return s.store.Save(*data)
}

// SaveSpeedTestResults stores speed test results into matching nodes across all subscriptions.
func (s *Service) SaveSpeedTestResults(results []types.SpeedTestUpdate) error {
	data, err := s.store.Load()
	if err != nil {
		return err
	}

	m := make(map[string]types.SpeedTestUpdate, len(results))
	for _, r := range results {
		m[r.Tag] = r
	}

	for i := range data.Subscriptions {
		for j := range data.Subscriptions[i].Nodes {
			node := &data.Subscriptions[i].Nodes[j]
			tag := ""
			if node.Outbound != nil {
				if t, ok := node.Outbound["tag"].(string); ok {
					tag = t
				}
			}
			if tag == "" {
				tag = types.SanitizeTag(node.Protocol, node.Address, node.Port)
			}
			if r, ok := m[tag]; ok {
				node.Latency = r.Latency
				node.SpeedKBps = r.SpeedKBps
				node.Online = r.Online
				node.LastProbe = r.LastProbe
			}
		}
	}

	return s.store.Save(*data)
}
