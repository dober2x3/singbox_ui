package wireguard

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

type Service struct {
	baseDir string
}

func NewService(baseDir string) *Service {
	return &Service{baseDir: baseDir}
}

func (s *Service) GeneratePrivateKey() (string, error) {
	var privateKey [32]byte
	_, err := rand.Read(privateKey[:])
	if err != nil {
		return "", err
	}
	privateKey[0] &= 248
	privateKey[31] &= 127
	privateKey[31] |= 64
	return base64.StdEncoding.EncodeToString(privateKey[:]), nil
}

func (s *Service) GeneratePublicKey(privateKeyStr string) (string, error) {
	privateKey, err := base64.StdEncoding.DecodeString(privateKeyStr)
	if err != nil {
		return "", fmt.Errorf("failed to decode private key: %w", err)
	}
	if len(privateKey) != 32 {
		return "", fmt.Errorf("invalid private key length: expected 32 bytes, got %d", len(privateKey))
	}
	var privKeyArr [32]byte
	copy(privKeyArr[:], privateKey)
	pubKey, err := curve25519.X25519(privKeyArr[:], curve25519.Basepoint)
	if err != nil {
		return "", fmt.Errorf("failed to generate public key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(pubKey), nil
}

func (s *Service) GenerateWireGuardKeysWithCache(ip string) (*KeyCacheResponse, error) {
	if ip == "" {
		return nil, fmt.Errorf("IP address is required")
	}

	cache, err := s.loadKeysCache()
	if err != nil {
		return nil, fmt.Errorf("failed to load keys cache: %w", err)
	}
	for _, entry := range cache {
		if entry.IP == ip {
			return &KeyCacheResponse{
				IP:         entry.IP,
				PrivateKey: entry.PrivateKey,
				PublicKey:  entry.PublicKey,
			}, nil
		}
	}

	privKey, err := s.GeneratePrivateKey()
	if err != nil {
		return nil, err
	}
	pubKey, err := s.GeneratePublicKey(privKey)
	if err != nil {
		return nil, err
	}

	cache = append(cache, KeyCacheEntry{
		IP:         ip,
		PublicKey:  pubKey,
		PrivateKey: privKey,
	})
	if err := s.saveKeysCache(cache); err != nil {
		return nil, err
	}

	return &KeyCacheResponse{
		IP:         ip,
		PrivateKey: privKey,
		PublicKey:  pubKey,
	}, nil
}

func (s *Service) GetKeysCache() ([]KeyCacheEntry, error) {
	return s.loadKeysCache()
}

func (s *Service) getKeysCacheFilePath() string {
	return filepath.Join(s.baseDir, "wireguard_keys_cache.txt")
}

func (s *Service) loadKeysCache() ([]KeyCacheEntry, error) {
	filePath := s.getKeysCacheFilePath()
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return nil, err
	}
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return []KeyCacheEntry{}, nil
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var cache []KeyCacheEntry
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) != 3 {
			continue
		}
		cache = append(cache, KeyCacheEntry{
			IP: parts[0], PublicKey: parts[1], PrivateKey: parts[2],
		})
	}
	return cache, nil
}

func (s *Service) saveKeysCache(cache []KeyCacheEntry) error {
	var lines []string
	for _, entry := range cache {
		lines = append(lines, fmt.Sprintf("%s %s %s", entry.IP, entry.PublicKey, entry.PrivateKey))
	}
	content := strings.Join(lines, "\n") + "\n"
	return os.WriteFile(s.getKeysCacheFilePath(), []byte(content), 0644)
}

func (s *Service) GetPublicIP() (string, error) {
	sources := []string{
		"https://api.ipify.org",
		"https://ifconfig.me",
		"https://icanhazip.com",
		"https://checkip.amazonaws.com",
	}
	timeout := 5 * time.Second
	var lastErr error
	for _, source := range sources {
		ip, err := s.fetchIPFromSource(source, timeout)
		if err != nil {
			lastErr = err
			continue
		}
		return ip, nil
	}
	if lastErr != nil {
		return "", fmt.Errorf("all IP sources failed: %w", lastErr)
	}
	return "", fmt.Errorf("no IP sources available")
}

func (s *Service) fetchIPFromSource(url string, timeout time.Duration) (string, error) {
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	ip := strings.TrimSpace(string(body))
	if net.ParseIP(ip) == nil {
		return "", fmt.Errorf("invalid IP: %s", ip)
	}
	return ip, nil
}

func (s *Service) SaveClientConfig(configData []byte) error {
	if err := os.MkdirAll(s.baseDir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.baseDir, "client-config.json"), configData, 0644)
}

func (s *Service) GetClientConfig() ([]byte, error) {
	path := filepath.Join(s.baseDir, "client-config.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("client config not found")
	}
	return os.ReadFile(path)
}

func (s *Service) SaveClientConfigFile(clientIndex int, configContent string) error {
	if err := os.MkdirAll(s.baseDir, 0755); err != nil {
		return err
	}
	confPath := filepath.Join(s.baseDir, fmt.Sprintf("client%d.conf", clientIndex))
	return os.WriteFile(confPath, []byte(configContent), 0644)
}

func (s *Service) ListClientConfigFiles() ([]ClientConfigFile, error) {
	if _, err := os.Stat(s.baseDir); os.IsNotExist(err) {
		return []ClientConfigFile{}, nil
	}
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, err
	}
	var configs []ClientConfigFile
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".conf" {
			content, err := os.ReadFile(filepath.Join(s.baseDir, entry.Name()))
			if err != nil {
				continue
			}
			configs = append(configs, ClientConfigFile{
				Name: entry.Name(), Content: string(content),
			})
		}
	}
	return configs, nil
}
