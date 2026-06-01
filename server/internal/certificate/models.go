package certificate

// CertificateInfo contains metadata about a TLS certificate.
// @Description Certificate metadata including validity period and fingerprint
type CertificateInfo struct {
	CertPath    string `json:"cert_path" example:"/data/singbox/cert.pem"`
	KeyPath     string `json:"key_path" example:"/data/singbox/key.pem"`
	CommonName  string `json:"common_name" example:"example.com"`
	ValidFrom   string `json:"valid_from" example:"2026-06-01T00:00:00Z"`
	ValidTo     string `json:"valid_to" example:"2027-06-01T00:00:00Z"`
	Fingerprint string `json:"fingerprint" example:"SHA256:abc123def456"`
}

// GenerateCertRequest request body for generating a self-signed certificate
// @Description Request to generate a self-signed TLS certificate
type GenerateCertRequest struct {
	Domain    string `json:"domain" example:"example.com" binding:"required"`
	ValidDays int    `json:"valid_days" example:"365" binding:"required"`
}

// RealityKeypairResponse response for Reality keypair generation
// @Description x25519 key pair for Reality TLS
type RealityKeypairResponse struct {
	PrivateKey string `json:"private_key" example:"ARVXKXp6V9XmXQmQmQmQmQmQmQmQmQmQmQmQmQmQmQmQmQ"`
	PublicKey  string `json:"public_key" example:"9XmXQmQmQmQmQmQmQmQmQmQmQmQmQmQmQmQmQmQmQmQmQ"`
}

// DerivePublicKeyRequest request body for deriving a Reality public key
// @Description Request to derive a public key from a private key
type DerivePublicKeyRequest struct {
	PrivateKey string `json:"private_key" example:"ARVXKXp6V9XmXQmQmQmQmQmQmQmQmQmQmQmQmQmQmQmQmQ" binding:"required"`
}

// CheckTLS13Request request body for checking TLS 1.3 support
// @Description Request to check if a server supports TLS 1.3 (required for Reality disguise domain)
type CheckTLS13Request struct {
	Server string `json:"server" example:"example.com" binding:"required"`
	Port   int    `json:"port" example:"443"`
}

// CheckTLS13Response response for TLS 1.3 support check
// @Description Result of TLS 1.3 support check
type CheckTLS13Response struct {
	Supported  bool   `json:"supported" example:"true"`
	TLSVersion string `json:"tls_version,omitempty" example:"TLS 1.3"`
	Error      string `json:"error,omitempty" example:"connection failed"`
}
