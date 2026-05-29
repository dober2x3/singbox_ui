package handlers

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"singbox-config-service/services"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/curve25519"
)

// GenerateCertRequest certificate generation request
type GenerateCertRequest struct {
	Domain    string `json:"domain" binding:"required"`   // domain or IP
	ValidDays int    `json:"valid_days"`                  // validity period in days, default 365
	Instance  string `json:"instance" binding:"required"` // instance name (required)
}

// GenerateSelfSignedCert generate self-signed certificate
func GenerateSelfSignedCert(c *gin.Context) {
	var req GenerateCertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid request body",
		})
		return
	}

	// get certificate directory (instance directory)
	certDir := getInstanceCertDir(req.Instance)

	// generate certificate
	certInfo, err := services.GenerateSelfSignedCert(req.Domain, req.ValidDays, certDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": err.Error(),
		})
		return
	}

	// return the path inside the container (sing-box container path)
	c.JSON(http.StatusOK, gin.H{
		"cert_path":      "/etc/sing-box/cert.pem",
		"key_path":       "/etc/sing-box/key.pem",
		"host_cert_path": certInfo.CertPath,
		"host_key_path":  certInfo.KeyPath,
		"common_name":    certInfo.CommonName,
		"valid_from":     certInfo.ValidFrom,
		"valid_to":       certInfo.ValidTo,
		"fingerprint":    certInfo.Fingerprint,
	})
}

// GetCertificateInfo get certificate info
func GetCertificateInfo(c *gin.Context) {
	instance := c.Query("instance")
	if instance == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "instance parameter is required",
		})
		return
	}

	certDir := getInstanceCertDir(instance)
	certPath := filepath.Join(certDir, "cert.pem")

	// check if certificate exists
	if !services.CertificateExists(certDir) {
		c.JSON(http.StatusNotFound, gin.H{
			"exists":  false,
			"message": "No certificate found",
		})
		return
	}

	// get certificate info
	certInfo, err := services.GetCertificateInfo(certPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"exists":         true,
		"cert_path":      "/etc/sing-box/cert.pem",
		"key_path":       "/etc/sing-box/key.pem",
		"host_cert_path": certInfo.CertPath,
		"host_key_path":  certInfo.KeyPath,
		"common_name":    certInfo.CommonName,
		"valid_from":     certInfo.ValidFrom,
		"valid_to":       certInfo.ValidTo,
		"fingerprint":    certInfo.Fingerprint,
	})
}

// getInstanceCertDir get instance certificate directory
func getInstanceCertDir(instance string) string {
	return filepath.Join(services.GetSingboxDir(), "configs", instance)
}

// UploadCertificate upload certificate file
func UploadCertificate(c *gin.Context) {
	instance := c.PostForm("instance")
	if instance == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "instance parameter is required",
		})
		return
	}

	// get certificate directory
	certDir := getInstanceCertDir(instance)

	// ensure directory exists
	if err := os.MkdirAll(certDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to create certificate directory: " + err.Error(),
		})
		return
	}

	// handle certificate file
	certFile, err := c.FormFile("cert")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "cert file is required",
		})
		return
	}

	// handle private key file
	keyFile, err := c.FormFile("key")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "key file is required",
		})
		return
	}

	// save certificate file
	certPath := filepath.Join(certDir, "cert.pem")
	certSrc, err := certFile.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to open cert file: " + err.Error(),
		})
		return
	}
	defer certSrc.Close()

	certDst, err := os.Create(certPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to create cert file: " + err.Error(),
		})
		return
	}
	defer certDst.Close()

	if _, err := io.Copy(certDst, certSrc); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to save cert file: " + err.Error(),
		})
		return
	}

	// save private key file
	keyPath := filepath.Join(certDir, "key.pem")
	keySrc, err := keyFile.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to open key file: " + err.Error(),
		})
		return
	}
	defer keySrc.Close()

	keyDst, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to create key file: " + err.Error(),
		})
		return
	}
	defer keyDst.Close()

	if _, err := io.Copy(keyDst, keySrc); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to save key file: " + err.Error(),
		})
		return
	}

	// get certificate info
	certInfo, err := services.GetCertificateInfo(certPath)
	if err != nil {
		// certificate saved, but could not parse certificate info
		c.JSON(http.StatusOK, gin.H{
			"cert_path":      "/etc/sing-box/cert.pem",
			"key_path":       "/etc/sing-box/key.pem",
			"host_cert_path": certPath,
			"host_key_path":  keyPath,
			"message":        "Certificate uploaded but could not parse info: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"cert_path":      "/etc/sing-box/cert.pem",
		"key_path":       "/etc/sing-box/key.pem",
		"host_cert_path": certPath,
		"host_key_path":  keyPath,
		"common_name":    certInfo.CommonName,
		"valid_from":     certInfo.ValidFrom,
		"valid_to":       certInfo.ValidTo,
		"fingerprint":    certInfo.Fingerprint,
	})
}

// DeriveRealityPublicKey derive public key from Reality private key
func DeriveRealityPublicKey(c *gin.Context) {
	var req struct {
		PrivateKey string `json:"private_key" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "private_key is required",
		})
		return
	}

	privateKeyBytes, err := base64.RawURLEncoding.DecodeString(req.PrivateKey)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid private key encoding",
		})
		return
	}

	if len(privateKeyBytes) != 32 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid private key length",
		})
		return
	}

	// Clamp private key for x25519 (consistent with GenerateRealityKeypair)
	privateKeyBytes[0] &= 248
	privateKeyBytes[31] &= 127
	privateKeyBytes[31] |= 64

	publicKey, err := curve25519.X25519(privateKeyBytes, curve25519.Basepoint)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to derive public key: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"public_key": base64.RawURLEncoding.EncodeToString(publicKey),
	})
}

// GenerateRealityKeypair generate Reality x25519 key pair
func GenerateRealityKeypair(c *gin.Context) {
	// generate random private key
	var privateKey [32]byte
	if _, err := rand.Read(privateKey[:]); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to generate private key: " + err.Error(),
		})
		return
	}

	// Clamp private key for x25519
	privateKey[0] &= 248
	privateKey[31] &= 127
	privateKey[31] |= 64

	// derive public key
	publicKey, err := curve25519.X25519(privateKey[:], curve25519.Basepoint)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to derive public key: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"private_key": base64.RawURLEncoding.EncodeToString(privateKey[:]),
		"public_key":  base64.RawURLEncoding.EncodeToString(publicKey),
	})
}

// CheckTLS13Support check if the target domain supports TLS 1.3 (Reality disguise domain requirement)
func CheckTLS13Support(c *gin.Context) {
	var req struct {
		Server string `json:"server" binding:"required"`
		Port   int    `json:"port"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "server is required",
		})
		return
	}
	if req.Port == 0 {
		req.Port = 443
	}

	// security check: reject IP addresses (Reality target must be a domain name)
	if ip := net.ParseIP(req.Server); ip != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "IP addresses not allowed, use a domain name",
		})
		return
	}

	// port range validation
	if req.Port < 1 || req.Port > 65535 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "port must be between 1 and 65535",
		})
		return
	}

	addr := fmt.Sprintf("%s:%d", req.Server, req.Port)
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{
		ServerName:         req.Server,
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS12,
		MaxVersion:         tls.VersionTLS13,
	})
	if err != nil {
		log.Printf("TLS check failed for %s:%d: %v", req.Server, req.Port, err)
		c.JSON(http.StatusOK, gin.H{
			"supported":   false,
			"tls_version": "",
			"error":       "connection failed",
		})
		return
	}
	defer conn.Close()

	state := conn.ConnectionState()
	var versionStr string
	switch state.Version {
	case tls.VersionTLS13:
		versionStr = "TLS 1.3"
	case tls.VersionTLS12:
		versionStr = "TLS 1.2"
	case tls.VersionTLS11:
		versionStr = "TLS 1.1"
	case tls.VersionTLS10:
		versionStr = "TLS 1.0"
	default:
		versionStr = fmt.Sprintf("unknown (0x%04x)", state.Version)
	}

	c.JSON(http.StatusOK, gin.H{
		"supported":   state.Version == tls.VersionTLS13,
		"tls_version": versionStr,
	})
}
