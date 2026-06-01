package certificate

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/curve25519"
)

// GenerateRealityKeypair generates a x25519 key pair for Reality TLS
func GenerateRealityKeypair(c *gin.Context) {
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

// DeriveRealityPublicKey derives a public key from a Reality private key
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

	// Clamp private key for x25519
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

// CheckTLS13Support checks if a server supports TLS 1.3 (Reality disguise domain requirement)
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

	// Security check: reject IP addresses (Reality target must be a domain name)
	if ip := net.ParseIP(req.Server); ip != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "IP addresses not allowed, use a domain name",
		})
		return
	}

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
