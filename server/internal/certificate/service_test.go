package certificate

import (
	"os"
	"testing"
)

// TestGenerateSelfSignedCert verifies generating a self-signed cert with a domain and valid days.
func TestGenerateSelfSignedCert(t *testing.T) {
	svc := NewService(t.TempDir())

	info, err := svc.GenerateSelfSignedCert("example.com", 30)
	if err != nil {
		t.Fatalf("GenerateSelfSignedCert() error = %v", err)
	}

	if info.CommonName != "example.com" {
		t.Errorf("CommonName = %q, want %q", info.CommonName, "example.com")
	}
	if info.CertPath == "" || info.KeyPath == "" {
		t.Error("CertPath or KeyPath is empty")
	}
	if info.Fingerprint == "" {
		t.Error("Fingerprint is empty")
	}
	if info.ValidFrom == "" || info.ValidTo == "" {
		t.Error("ValidFrom or ValidTo is empty")
	}

	if _, err := os.Stat(info.CertPath); os.IsNotExist(err) {
		t.Errorf("cert file not created: %s", info.CertPath)
	}
	if _, err := os.Stat(info.KeyPath); os.IsNotExist(err) {
		t.Errorf("key file not created: %s", info.KeyPath)
	}
}

// TestGenerateSelfSignedCert_defaultDomain verifies empty domain defaults to "localhost".
func TestGenerateSelfSignedCert_defaultDomain(t *testing.T) {
	svc := NewService(t.TempDir())
	info, err := svc.GenerateSelfSignedCert("", 30)
	if err != nil {
		t.Fatalf("GenerateSelfSignedCert() error = %v", err)
	}
	if info.CommonName != "localhost" {
		t.Errorf("CommonName = %q, want %q", info.CommonName, "localhost")
	}
}

// TestGenerateSelfSignedCert_defaultDays verifies zero valid days defaults to 365.
func TestGenerateSelfSignedCert_defaultDays(t *testing.T) {
	svc := NewService(t.TempDir())
	info, err := svc.GenerateSelfSignedCert("test.com", 0)
	if err != nil {
		t.Fatalf("GenerateSelfSignedCert() error = %v", err)
	}
	if info.CommonName != "test.com" {
		t.Errorf("CommonName = %q, want %q", info.CommonName, "test.com")
	}
}

// TestGetCertificateInfo verifies reading back certificate metadata matches what was generated.
func TestGetCertificateInfo(t *testing.T) {
	svc := NewService(t.TempDir())
	genInfo, _ := svc.GenerateSelfSignedCert("test.com", 30)

	info, err := svc.GetCertificateInfo(genInfo.CertPath)
	if err != nil {
		t.Fatalf("GetCertificateInfo() error = %v", err)
	}
	if info.CommonName != "test.com" {
		t.Errorf("CommonName = %q, want %q", info.CommonName, "test.com")
	}
}

// TestGetCertificateInfo_notFound verifies GetCertificateInfo errors on missing file.
func TestGetCertificateInfo_notFound(t *testing.T) {
	svc := NewService(t.TempDir())
	_, err := svc.GetCertificateInfo("/nonexistent/cert.pem")
	if err == nil {
		t.Error("GetCertificateInfo() expected error, got nil")
	}
}

// TestCertificateExists verifies CertificateExists returns false before and true after generation.
func TestCertificateExists(t *testing.T) {
	svc := NewService(t.TempDir())
	if svc.CertificateExists() {
		t.Error("CertificateExists() = true before generating cert")
	}
	if _, err := svc.GenerateSelfSignedCert("test.com", 30); err != nil {
		t.Fatalf("GenerateSelfSignedCert() error = %v", err)
	}
	if !svc.CertificateExists() {
		t.Error("CertificateExists() = false after generating cert")
	}
}
