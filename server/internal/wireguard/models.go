package wireguard

// WireGuardKeyPair represents a WireGuard key pair with private and public keys.
// @Description WireGuard key pair with private and public keys
type WireGuardKeyPair struct {
	PrivateKey string `json:"privateKey" example:"gL7fV/1GkH+3..."`
	PublicKey  string `json:"publicKey" example:"xTIBdMd5Fq+3..."`
}

// KeyCacheEntry represents a cached WireGuard key entry for a specific IP.
// @Description Cached WireGuard key entry for a specific IP
type KeyCacheEntry struct {
	IP         string `json:"ip" example:"10.0.0.1"`
	PublicKey  string `json:"publicKey" example:"xTIBdMd5Fq+3..."`
	PrivateKey string `json:"privateKey" example:"gL7fV/1GkH+3..."`
}

// KeyCacheResponse is the API response for a WireGuard key generation request.
// @Description API response for a WireGuard key generation request
type KeyCacheResponse struct {
	IP         string `json:"ip" example:"10.0.0.1"`
	PrivateKey string `json:"privateKey" example:"gL7fV/1GkH+3..."`
	PublicKey  string `json:"publicKey" example:"xTIBdMd5Fq+3..."`
}

// ClientConfigFile represents a saved WireGuard client config file.
// @Description Saved WireGuard client config file
type ClientConfigFile struct {
	Name    string `json:"name" example:"client-0.conf"`
	Content string `json:"content" example:"[Interface]\nPrivateKey = ..."`
}

// WireGuardKeyRequest request body for generating WireGuard keys
// @Description Request to generate or retrieve cached WireGuard keys
type WireGuardKeyRequest struct {
	IP string `json:"ip" example:"10.0.0.1"`
}

// DerivePublicKeyRequest request body for deriving a public key from a private key
// @Description Request to derive a WireGuard public key from a private key
type DerivePublicKeyRequest struct {
	PrivateKey string `json:"private_key" example:"gL7fV/1GkH+3..." binding:"required"`
}

// SaveClientFileRequest request body for saving a client config file
// @Description Request to save a WireGuard client .conf file
type SaveClientFileRequest struct {
	ClientIndex int    `json:"client_index" example:"0"`
	Content     string `json:"content" example:"[Interface]\nPrivateKey = ..."`
}

// PublicIPResponse response with the server's public IP
// @Description Response containing the server's public IP address
type PublicIPResponse struct {
	IP string `json:"ip" example:"203.0.113.1"`
}

// MessageResponse generic message response
// @Description Generic message response
type MessageResponse struct {
	Message string `json:"message" example:"operation completed"`
}
