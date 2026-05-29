package services

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/curve25519"
)

// WireGuardKeyPair WireGuard key pair
type WireGuardKeyPair struct {
	PrivateKey string `json:"privateKey"`
	PublicKey  string `json:"publicKey"`
}

// KeyCacheEntry key cache entry
type KeyCacheEntry struct {
	IP         string `json:"ip"`
	PublicKey  string `json:"publicKey"`
	PrivateKey string `json:"privateKey"`
}

// KeyCacheResponse key cache response (contains IP and key pair)
type KeyCacheResponse struct {
	IP         string `json:"ip"`
	PrivateKey string `json:"privateKey"`
	PublicKey  string `json:"publicKey"`
}

func getKeysCacheFilePath() string {
	return filepath.Join(singboxDir, "wireguard_keys_cache.txt")
}

// GenerateWireGuardKeysWithCache generates a WireGuard key pair with cache
// ip: must be a complete IP address, e.g. "10.10.0.5"
func GenerateWireGuardKeysWithCache(ip string) (*KeyCacheResponse, error) {
	// IP is required
	if ip == "" {
		return nil, fmt.Errorf("IP address is required")
	}

	// Read existing cache
	cache, err := loadKeysCache()
	if err != nil {
		return nil, fmt.Errorf("failed to load keys cache: %w", err)
	}

	// Check if IP already exists, return cached keys if so
	for _, entry := range cache {
		if entry.IP == ip {
			return &KeyCacheResponse{
				IP:         entry.IP,
				PrivateKey: entry.PrivateKey,
				PublicKey:  entry.PublicKey,
			}, nil
		}
	}

	targetIP := ip

	// Generate new key pair
	privateKey, err := generatePrivateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	publicKey, err := generatePublicKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to generate public key: %w", err)
	}

	// Create new cache entry
	entry := KeyCacheEntry{
		IP:         targetIP,
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}

	// Add to cache
	cache = append(cache, entry)

	// Save cache
	if err := saveKeysCache(cache); err != nil {
		return nil, fmt.Errorf("failed to save keys cache: %w", err)
	}

	return &KeyCacheResponse{
		IP:         targetIP,
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}, nil
}

// loadKeysCache loads the key cache
func loadKeysCache() ([]KeyCacheEntry, error) {
	keysCacheFile := getKeysCacheFilePath()
	// Ensure data directory exists
	dataDir := filepath.Dir(keysCacheFile)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	// If file does not exist, return empty cache
	if _, err := os.Stat(keysCacheFile); os.IsNotExist(err) {
		return []KeyCacheEntry{}, nil
	}

	// Read file
	data, err := os.ReadFile(keysCacheFile)
	if err != nil {
		return nil, err
	}

	// Parse each line: IP public_key private_key
	var cache []KeyCacheEntry
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) != 3 {
			continue
		}

		cache = append(cache, KeyCacheEntry{
			IP:         parts[0],
			PublicKey:  parts[1],
			PrivateKey: parts[2],
		})
	}

	return cache, nil
}

// saveKeysCache saves the key cache
func saveKeysCache(cache []KeyCacheEntry) error {
	keysCacheFile := getKeysCacheFilePath()
	// Ensure data directory exists
	dataDir := filepath.Dir(keysCacheFile)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return err
	}

	// Build file content
	var lines []string
	for _, entry := range cache {
		line := fmt.Sprintf("%s %s %s", entry.IP, entry.PublicKey, entry.PrivateKey)
		lines = append(lines, line)
	}

	// Write file
	content := strings.Join(lines, "\n") + "\n"
	return os.WriteFile(keysCacheFile, []byte(content), 0644)
}

// GetKeysCache gets the key cache list
func GetKeysCache() ([]KeyCacheEntry, error) {
	return loadKeysCache()
}

// generatePrivateKey generates a WireGuard private key
func generatePrivateKey() (string, error) {
	var privateKey [32]byte
	_, err := rand.Read(privateKey[:])
	if err != nil {
		return "", err
	}

	// WireGuard requires specific bit manipulation on the private key
	// These operations ensure the key meets Curve25519 requirements
	privateKey[0] &= 248
	privateKey[31] &= 127
	privateKey[31] |= 64

	return base64.StdEncoding.EncodeToString(privateKey[:]), nil
}

// generatePublicKey generates a public key from a private key
func generatePublicKey(privateKeyStr string) (string, error) {
	privateKey, err := base64.StdEncoding.DecodeString(privateKeyStr)
	if err != nil {
		return "", fmt.Errorf("failed to decode private key: %w", err)
	}

	if len(privateKey) != 32 {
		return "", fmt.Errorf("invalid private key length: expected 32 bytes, got %d", len(privateKey))
	}

	var privateKeyArray [32]byte
	copy(privateKeyArray[:], privateKey)

	// Use Curve25519 to generate public key
	publicKey, err := curve25519.X25519(privateKeyArray[:], curve25519.Basepoint)
	if err != nil {
		return "", fmt.Errorf("failed to generate public key: %w", err)
	}

	return base64.StdEncoding.EncodeToString(publicKey), nil
}

// GeneratePublicKeyFromPrivate generates a public key from a private key (public function)
func GeneratePublicKeyFromPrivate(privateKeyStr string) (string, error) {
	return generatePublicKey(privateKeyStr)
}

// SaveClientConfig saves client config to file
func SaveClientConfig(configData []byte) error {
	if err := os.MkdirAll(singboxDir, 0755); err != nil {
		return fmt.Errorf("failed to create wireguard directory: %w", err)
	}

	configPath := filepath.Join(singboxDir, "client-config.json")
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		return fmt.Errorf("failed to save client config: %w", err)
	}

	return nil
}

// GetClientConfig reads client config from file
func GetClientConfig() ([]byte, error) {
	configPath := filepath.Join(singboxDir, "client-config.json")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("client config file not found")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read client config: %w", err)
	}

	return data, nil
}

// SaveClientConfigFile saves client config file (supports multi-client)
func SaveClientConfigFile(clientIndex int, configContent string) error {
	if err := os.MkdirAll(singboxDir, 0755); err != nil {
		return fmt.Errorf("failed to create wireguard directory: %w", err)
	}

	// Save .conf file
	confPath := filepath.Join(singboxDir, fmt.Sprintf("client%d.conf", clientIndex))
	if err := os.WriteFile(confPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to save client config file: %w", err)
	}

	return nil
}

// ClientConfigFile client config file information
type ClientConfigFile struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// ListClientConfigFiles lists all client config files
func ListClientConfigFiles() ([]ClientConfigFile, error) {
	// Check if directory exists
	if _, err := os.Stat(singboxDir); os.IsNotExist(err) {
		return []ClientConfigFile{}, nil
	}

	// Read all files in directory
	files, err := os.ReadDir(singboxDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read wireguard directory: %w", err)
	}

	configs := []ClientConfigFile{}
	for _, file := range files {
		// Only process .conf files
		if !file.IsDir() && filepath.Ext(file.Name()) == ".conf" {
			confPath := filepath.Join(singboxDir, file.Name())
			content, err := os.ReadFile(confPath)
			if err != nil {
				continue // skip unreadable files
			}

			configs = append(configs, ClientConfigFile{
				Name:    file.Name(),
				Content: string(content),
			})
		}
	}

	return configs, nil
}

// isValidIP validates IP address format (supports IPv4 and IPv6)
func isValidIP(ip string) bool {
	parsedIP := net.ParseIP(ip)
	return parsedIP != nil
}

// fetchIPFromSource fetches public IP from a specified source
func fetchIPFromSource(url string, timeout time.Duration) (string, error) {
	client := &http.Client{
		Timeout: timeout,
	}

	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	ip := strings.TrimSpace(string(body))

	// Validate IP format
	if !isValidIP(ip) {
		return "", fmt.Errorf("invalid IP format: %s", ip)
	}

	return ip, nil
}

// GetPublicIP gets the server's public IP address (supports multi-source and failover)
func GetPublicIP() (string, error) {
	// Multiple IP sources (sorted by priority)
	sources := []string{
		"https://api.ipify.org",
		"https://ifconfig.me",
		"https://icanhazip.com",
		"https://checkip.amazonaws.com",
	}

	timeout := 5 * time.Second
	var lastErr error

	// Try each source until successful
	for _, source := range sources {
		ip, err := fetchIPFromSource(source, timeout)
		if err != nil {
			lastErr = fmt.Errorf("source %s failed: %w", source, err)
			continue
		}

		// Successfully obtained and validated IP
		return ip, nil
	}

	// All sources failed
	if lastErr != nil {
		return "", fmt.Errorf("failed to get public IP from all sources: %w", lastErr)
	}
	return "", fmt.Errorf("failed to get public IP: no sources available")
}
