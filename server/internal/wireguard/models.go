package wireguard

// WireGuardKeyPair represents a WireGuard key pair with private and public keys.
type WireGuardKeyPair struct {
	PrivateKey string `json:"privateKey"`
	PublicKey  string `json:"publicKey"`
}

// KeyCacheEntry represents a cached WireGuard key entry for a specific IP.
type KeyCacheEntry struct {
	IP         string `json:"ip"`
	PublicKey  string `json:"publicKey"`
	PrivateKey string `json:"privateKey"`
}

// KeyCacheResponse is the API response for a WireGuard key generation request.
type KeyCacheResponse struct {
	IP         string `json:"ip"`
	PrivateKey string `json:"privateKey"`
	PublicKey  string `json:"publicKey"`
}

// ClientConfigFile represents a saved WireGuard client config file.
type ClientConfigFile struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}
