package wireguard

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	_ "singbox-config-service/internal/docs"
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
// @Summary      Generate WireGuard keys
// @Description  Generates or retrieves cached WireGuard key pair. Optionally associates with an IP address
// @Tags         wireguard
// @Accept       json
// @Produce      json
// @Param        request body WireGuardKeyRequest false "Optional IP address"
// @Success      200  {object}  WireGuardKeyPair
// @Failure      500  {object}  docs.ErrorResponse
// @Router       /wireguard/keygen [post]
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
// @Summary      Derive public key
// @Description  Derives a WireGuard public key from a base64-encoded private key
// @Tags         wireguard
// @Accept       json
// @Produce      json
// @Param        request body DerivePublicKeyRequest true "Private key"
// @Success      200  {object}  map[string]string  "public key"
// @Failure      400  {object}  docs.ErrorResponse
// @Failure      500  {object}  docs.ErrorResponse
// @Router       /wireguard/pubkey [post]
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
// @Summary      Get keys cache
// @Description  Returns all cached WireGuard key entries with their associated IPs
// @Tags         wireguard
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "keyed by IP"
// @Failure      500  {object}  docs.ErrorResponse
// @Router       /wireguard/keys-cache [get]
func (h *Handler) GetKeysCache(c *gin.Context) {
	cache, err := h.svc.GetKeysCache()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, cache)
}

// GetPublicIP handles GET /public-ip to retrieve the server's public IP address.
// @Summary      Get public IP
// @Description  Returns the server's detected public IPv4 address
// @Tags         wireguard
// @Produce      json
// @Success      200  {object}  PublicIPResponse
// @Failure      500  {object}  docs.ErrorResponse
// @Router       /wireguard/public-ip [get]
func (h *Handler) GetPublicIP(c *gin.Context) {
	ip, err := h.svc.GetPublicIP()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ip": ip})
}

// GetClientConfig handles GET /client-config to retrieve the saved client JSON configuration.
// @Summary      Get client config
// @Description  Returns the saved WireGuard client configuration as raw JSON
// @Tags         wireguard
// @Produce      json
// @Success      200  {string}  string  "client config JSON"
// @Failure      404  {object}  docs.ErrorResponse
// @Router       /wireguard/client-config [get]
func (h *Handler) GetClientConfig(c *gin.Context) {
	data, err := h.svc.GetClientConfig()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Data(http.StatusOK, "application/json", data)
}

// SaveClientConfig handles POST /client-config to save client JSON configuration.
// @Summary      Save client config
// @Description  Saves raw JSON as the WireGuard client configuration
// @Tags         wireguard
// @Accept       json
// @Produce      json
// @Success      200  {object}  MessageResponse
// @Failure      400  {object}  docs.ErrorResponse
// @Failure      500  {object}  docs.ErrorResponse
// @Router       /wireguard/client-config [post]
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
// @Summary      Save client config file
// @Description  Saves a .conf file content for a specific client. Accepts JSON body or multipart form
// @Tags         wireguard
// @Accept       json
// @Produce      json
// @Param        request body SaveClientFileRequest true "Client file info"
// @Success      200  {object}  MessageResponse
// @Failure      400  {object}  docs.ErrorResponse
// @Failure      500  {object}  docs.ErrorResponse
// @Router       /wireguard/save-client-file [post]
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
// @Summary      List client config files
// @Description  Returns a list of all saved WireGuard .conf files
// @Tags         wireguard
// @Produce      json
// @Success      200  {array}   ClientConfigFile
// @Failure      500  {object}  docs.ErrorResponse
// @Router       /wireguard/client-files [get]
func (h *Handler) ListClientConfigFiles(c *gin.Context) {
	files, err := h.svc.ListClientConfigFiles()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, files)
}
