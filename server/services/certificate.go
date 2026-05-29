package services

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

// CertificateInfo certificate information
type CertificateInfo struct {
	CertPath    string `json:"cert_path"`
	KeyPath     string `json:"key_path"`
	CommonName  string `json:"common_name"`
	ValidFrom   string `json:"valid_from"`
	ValidTo     string `json:"valid_to"`
	Fingerprint string `json:"fingerprint"`
}

// GenerateSelfSignedCert generates a self-signed certificate
// domain: domain or IP, used for CN and SAN
// validDays: certificate validity days
// certDir: certificate save directory
func GenerateSelfSignedCert(domain string, validDays int, certDir string) (*CertificateInfo, error) {
	if domain == "" {
		domain = "localhost"
	}
	if validDays <= 0 {
		validDays = 365
	}

	// Ensure directory exists
	if err := os.MkdirAll(certDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create certificate directory: %v", err)
	}

	// Generate ECDSA private key (P-256 curve)
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %v", err)
	}

	// Generate certificate serial number
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %v", err)
	}

	// Certificate validity period
	notBefore := time.Now()
	notAfter := notBefore.AddDate(0, 0, validDays)

	// Certificate template
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Sing-box UI Self-Signed"},
			CommonName:   domain,
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// Add SAN (Subject Alternative Names)
	if ip := net.ParseIP(domain); ip != nil {
		template.IPAddresses = []net.IP{ip}
	} else {
		template.DNSNames = []string{domain}
	}

	// Always add localhost
	if domain != "localhost" {
		template.DNSNames = append(template.DNSNames, "localhost")
	}
	template.IPAddresses = append(template.IPAddresses, net.ParseIP("127.0.0.1"))
	template.IPAddresses = append(template.IPAddresses, net.ParseIP("::1"))

	// Self-signed certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %v", err)
	}

	// Save certificate file
	certPath := filepath.Join(certDir, "cert.pem")
	certOut, err := os.Create(certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create cert.pem: %v", err)
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return nil, fmt.Errorf("failed to write cert.pem: %v", err)
	}

	// Save private key file
	keyPath := filepath.Join(certDir, "key.pem")
	keyOut, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to create key.pem: %v", err)
	}
	defer keyOut.Close()

	keyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private key: %v", err)
	}

	if err := pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes}); err != nil {
		return nil, fmt.Errorf("failed to write key.pem: %v", err)
	}

	// Calculate certificate fingerprint
	cert, err := x509.ParseCertificate(derBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %v", err)
	}

	fingerprint := fmt.Sprintf("%X", cert.Raw[:20])

	return &CertificateInfo{
		CertPath:    certPath,
		KeyPath:     keyPath,
		CommonName:  domain,
		ValidFrom:   notBefore.Format(time.RFC3339),
		ValidTo:     notAfter.Format(time.RFC3339),
		Fingerprint: fingerprint[:40], // first 40 characters
	}, nil
}

// GetCertificateInfo gets certificate information
func GetCertificateInfo(certPath string) (*CertificateInfo, error) {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate: %v", err)
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %v", err)
	}

	// Infer key path
	keyPath := filepath.Join(filepath.Dir(certPath), "key.pem")

	fingerprint := fmt.Sprintf("%X", cert.Raw[:20])

	return &CertificateInfo{
		CertPath:    certPath,
		KeyPath:     keyPath,
		CommonName:  cert.Subject.CommonName,
		ValidFrom:   cert.NotBefore.Format(time.RFC3339),
		ValidTo:     cert.NotAfter.Format(time.RFC3339),
		Fingerprint: fingerprint[:40],
	}, nil
}

// CertificateExists checks if a certificate exists
func CertificateExists(certDir string) bool {
	certPath := filepath.Join(certDir, "cert.pem")
	keyPath := filepath.Join(certDir, "key.pem")

	_, certErr := os.Stat(certPath)
	_, keyErr := os.Stat(keyPath)

	return certErr == nil && keyErr == nil
}
