package wireguard

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests for WireGuard-related endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new Handler with the given Service.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// GenerateWireGuardKeys handles POST /keygen to generate or retrieve cached WireGuard keys.
func (h *Handler) GenerateWireGuardKeys(c *gin.Context) {
	var req struct {
		IP string `json:"ip"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.IP = ""
	}
	resp, err := h.svc.GenerateWireGuardKeysWithCache(req.IP)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GetPublicKeyFromPrivate handles POST /pubkey to derive a public key from a private key.
func (h *Handler) GetPublicKeyFromPrivate(c *gin.Context) {
	var req struct {
		PrivateKey string `json:"private_key"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.PrivateKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "private_key is required"})
		return
	}
	pub, err := h.svc.GeneratePublicKey(req.PrivateKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"public_key": pub})
}

// GetKeysCache handles GET /keys-cache to retrieve all cached WireGuard key entries.
func (h *Handler) GetKeysCache(c *gin.Context) {
	cache, err := h.svc.GetKeysCache()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, cache)
}

// GetPublicIP handles GET /public-ip to retrieve the server's public IP address.
func (h *Handler) GetPublicIP(c *gin.Context) {
	ip, err := h.svc.GetPublicIP()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ip": ip})
}

// GetClientConfig handles GET /client-config to retrieve the saved client JSON configuration.
func (h *Handler) GetClientConfig(c *gin.Context) {
	data, err := h.svc.GetClientConfig()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Data(http.StatusOK, "application/json", data)
}

// SaveClientConfig handles POST /client-config to save client JSON configuration.
func (h *Handler) SaveClientConfig(c *gin.Context) {
	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.SaveClientConfig(body); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "config saved"})
}

// SaveClientConfigFile handles POST /save-client-file to save a client config (.conf) file.
func (h *Handler) SaveClientConfigFile(c *gin.Context) {
	var req struct {
		ClientIndex int    `json:"client_index"`
		Content     string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Override content from form if provided
	if content := c.PostForm("content"); content != "" {
		req.Content = content
	}
	if indexStr := c.PostForm("client_index"); indexStr != "" {
		if idx, err := strconv.Atoi(indexStr); err == nil {
			req.ClientIndex = idx
		}
	}
	if err := h.svc.SaveClientConfigFile(req.ClientIndex, req.Content); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "config file saved"})
}

// ListClientConfigFiles handles GET /client-files to list all saved client config files.
func (h *Handler) ListClientConfigFiles(c *gin.Context) {
	files, err := h.svc.ListClientConfigFiles()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, files)
}
