package certificate

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

type Service struct {
	certDir string
}

func NewService(certDir string) *Service {
	return &Service{certDir: certDir}
}

func (s *Service) GenerateSelfSignedCert(domain string, validDays int) (*CertificateInfo, error) {
	if domain == "" {
		domain = "localhost"
	}
	if validDays <= 0 {
		validDays = 365
	}

	if err := os.MkdirAll(s.certDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create certificate directory: %w", err)
	}

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	notBefore := time.Now()
	notAfter := notBefore.AddDate(0, 0, validDays)

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

	if ip := net.ParseIP(domain); ip != nil {
		template.IPAddresses = []net.IP{ip}
	} else {
		template.DNSNames = []string{domain}
	}
	if domain != "localhost" {
		template.DNSNames = append(template.DNSNames, "localhost")
	}
	template.IPAddresses = append(template.IPAddresses, net.ParseIP("127.0.0.1"), net.ParseIP("::1"))

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	certPath := filepath.Join(s.certDir, "cert.pem")
	certOut, err := os.Create(certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create cert.pem: %w", err)
	}
	defer certOut.Close()
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return nil, fmt.Errorf("failed to write cert.pem: %w", err)
	}

	keyPath := filepath.Join(s.certDir, "key.pem")
	keyOut, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to create key.pem: %w", err)
	}
	defer keyOut.Close()
	keyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private key: %w", err)
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes}); err != nil {
		return nil, fmt.Errorf("failed to write key.pem: %w", err)
	}

	cert, err := x509.ParseCertificate(derBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}
	fingerprint := fmt.Sprintf("%X", cert.Raw[:20])

	return &CertificateInfo{
		CertPath:    certPath,
		KeyPath:     keyPath,
		CommonName:  domain,
		ValidFrom:   notBefore.Format(time.RFC3339),
		ValidTo:     notAfter.Format(time.RFC3339),
		Fingerprint: fingerprint[:40],
	}, nil
}

func (s *Service) GetCertificateInfo(certPath string) (*CertificateInfo, error) {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate: %w", err)
	}
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}
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

func (s *Service) CertificateExists() bool {
	certPath := filepath.Join(s.certDir, "cert.pem")
	keyPath := filepath.Join(s.certDir, "key.pem")
	_, certErr := os.Stat(certPath)
	_, keyErr := os.Stat(keyPath)
	return certErr == nil && keyErr == nil
}
