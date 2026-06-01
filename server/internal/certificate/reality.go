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

	"singbox-config-service/internal/docs"
)

// GenerateRealityKeypair generates a x25519 key pair for Reality TLS
// @Summary      Generate Reality key pair
// @Description  Generates a random x25519 key pair for use with Reality TLS
// @Tags         reality
// @Produce      json
// @Success      200  {object}  RealityKeypairResponse
// @Failure      500  {object}  docs.ErrorResponse
// @Router       /singbox/reality/keypair [post]
func GenerateRealityKeypair(c *gin.Context) {
	var privateKey [32]byte
	if _, err := rand.Read(privateKey[:]); err != nil {
		c.JSON(http.StatusInternalServerError, docs.ErrorResponse{
			Error:   "Internal Server Error",
			Message: "Failed to generate private key: " + err.Error(),
		})
		return
	}

	// Clamp private key for x25519
	privateKey[0] &= 248
	privateKey[31] &= 127
	privateKey[31] |= 64

	publicKey, err := curve25519.X25519(privateKey[:], curve25519.Basepoint)
	if err != nil {
		c.JSON(http.StatusInternalServerError, docs.ErrorResponse{
			Error:   "Internal Server Error",
			Message: "Failed to derive public key: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"private_key": base64.RawURLEncoding.EncodeToString(privateKey[:]),
		"public_key":  base64.RawURLEncoding.EncodeToString(publicKey),
	})
}

// DeriveRealityPublicKey derives a public key from a Reality private key
// @Summary      Derive Reality public key
// @Description  Derives the public key from a base64-encoded x25519 private key
// @Tags         reality
// @Accept       json
// @Produce      json
// @Param        request body DerivePublicKeyRequest true "Private key"
// @Success      200  {object}  RealityKeypairResponse
// @Failure      400  {object}  docs.ErrorResponse
// @Failure      500  {object}  docs.ErrorResponse
// @Router       /singbox/reality/public-key [post]
func DeriveRealityPublicKey(c *gin.Context) {
	var req struct {
		PrivateKey string `json:"private_key" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, docs.ErrorResponse{
			Error:   "Bad Request",
			Message: "private_key is required",
		})
		return
	}

	privateKeyBytes, err := base64.RawURLEncoding.DecodeString(req.PrivateKey)
	if err != nil {
		c.JSON(http.StatusBadRequest, docs.ErrorResponse{
			Error:   "Bad Request",
			Message: "Invalid private key encoding",
		})
		return
	}

	if len(privateKeyBytes) != 32 {
		c.JSON(http.StatusBadRequest, docs.ErrorResponse{
			Error:   "Bad Request",
			Message: "Invalid private key length",
		})
		return
	}

	// Clamp private key for x25519
	privateKeyBytes[0] &= 248
	privateKeyBytes[31] &= 127
	privateKeyBytes[31] |= 64

	publicKey, err := curve25519.X25519(privateKeyBytes, curve25519.Basepoint)
	if err != nil {
		c.JSON(http.StatusInternalServerError, docs.ErrorResponse{
			Error:   "Internal Server Error",
			Message: "Failed to derive public key: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"public_key": base64.RawURLEncoding.EncodeToString(publicKey),
	})
}

// CheckTLS13Support checks if a server supports TLS 1.3
// @Summary      Check TLS 1.3 support
// @Description  Checks if a remote server supports TLS 1.3 (required for Reality disguise domain)
// @Tags         reality
// @Accept       json
// @Produce      json
// @Param        request body CheckTLS13Request true "Server domain and port"
// @Success      200  {object}  CheckTLS13Response
// @Failure      400  {object}  docs.ErrorResponse
// @Router       /singbox/reality/check-tls [post]
func CheckTLS13Support(c *gin.Context) {
	var req struct {
		Server string `json:"server" binding:"required"`
		Port   int    `json:"port"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, docs.ErrorResponse{
			Error:   "Bad Request",
			Message: "server is required",
		})
		return
	}
	if req.Port == 0 {
		req.Port = 443
	}

	// Security check: reject IP addresses (Reality target must be a domain name)
	if ip := net.ParseIP(req.Server); ip != nil {
		c.JSON(http.StatusBadRequest, docs.ErrorResponse{
			Error:   "Bad Request",
			Message: "IP addresses not allowed, use a domain name",
		})
		return
	}

	if req.Port < 1 || req.Port > 65535 {
		c.JSON(http.StatusBadRequest, docs.ErrorResponse{
			Error:   "Bad Request",
			Message: "port must be between 1 and 65535",
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
