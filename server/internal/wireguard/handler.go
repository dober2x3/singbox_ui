package wireguard

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

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

func (h *Handler) GetKeysCache(c *gin.Context) {
	cache, err := h.svc.GetKeysCache()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, cache)
}

func (h *Handler) GetPublicIP(c *gin.Context) {
	ip, err := h.svc.GetPublicIP()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ip": ip})
}

func (h *Handler) GetClientConfig(c *gin.Context) {
	data, err := h.svc.GetClientConfig()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Data(http.StatusOK, "application/json", data)
}

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

func (h *Handler) ListClientConfigFiles(c *gin.Context) {
	files, err := h.svc.ListClientConfigFiles()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, files)
}
