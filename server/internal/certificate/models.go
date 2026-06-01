package certificate

// CertificateInfo contains metadata about a TLS certificate.
type CertificateInfo struct {
	CertPath    string `json:"cert_path"`
	KeyPath     string `json:"key_path"`
	CommonName  string `json:"common_name"`
	ValidFrom   string `json:"valid_from"`
	ValidTo     string `json:"valid_to"`
	Fingerprint string `json:"fingerprint"`
}
