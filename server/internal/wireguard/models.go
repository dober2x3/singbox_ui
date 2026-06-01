package wireguard

type WireGuardKeyPair struct {
	PrivateKey string `json:"privateKey"`
	PublicKey  string `json:"publicKey"`
}

type KeyCacheEntry struct {
	IP         string `json:"ip"`
	PublicKey  string `json:"publicKey"`
	PrivateKey string `json:"privateKey"`
}

type KeyCacheResponse struct {
	IP         string `json:"ip"`
	PrivateKey string `json:"privateKey"`
	PublicKey  string `json:"publicKey"`
}

type ClientConfigFile struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}
