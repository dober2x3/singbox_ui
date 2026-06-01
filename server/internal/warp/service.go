// Package warp provides WARP client registration, key management, and endpoint scanning.
package warp

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/curve25519"
)

// warpAPIBase is the base URL for the Cloudflare WARP API.
var warpAPIBase = "https://api.cloudflareclient.com"

// WARP API and client constants.
const (
	warpAPIVersion  = "v0a2158"
	warpClientUA    = "okhttp/3.12.1"
	warpClientVer   = "a-6.11-2158"
	warpDefaultHost = "engage.cloudflareclient.com"
	warpDefaultPort = 2408
)

// Service manages WARP device registration, record persistence, and outbound configuration.
type Service struct {
	baseDir string
	record  *WarpRecord
}

// NewService creates a new Service with the given base directory for storing records.
func NewService(baseDir string) *Service {
	return &Service{baseDir: baseDir}
}

// RegisterDevice registers a new WARP device with Cloudflare and stores the record.
func (s *Service) RegisterDevice() (*WarpRecord, error) {
	privKey, err := generatePrivateKey()
	if err != nil {
		return nil, fmt.Errorf("generate private key: %w", err)
	}
	pubKey, err := generatePublicKey(privKey)
	if err != nil {
		return nil, fmt.Errorf("generate public key: %w", err)
	}

	serial, err := randomHexStr(8)
	if err != nil {
		return nil, err
	}

	body := map[string]interface{}{
		"key": pubKey, "install_id": "", "fcm_token": "",
		"tos": time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		"model": "PC", "serial_number": serial, "locale": "en_US",
	}
	payload, _ := json.Marshal(body)

	url := fmt.Sprintf("%s/%s/reg", warpAPIBase, warpAPIVersion)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("User-Agent", warpClientUA)
	req.Header.Set("CF-Client-Version", warpClientVer)

	resp, err := httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("WARP registration failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("WARP registration HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var regResp WarpRegisterResponse
	if err := json.Unmarshal(respBody, &regResp); err != nil {
		return nil, fmt.Errorf("parse WARP response: %w", err)
	}
	if len(regResp.Config.Peers) == 0 {
		return nil, fmt.Errorf("WARP response missing peer config")
	}

	now := time.Now().Format(time.RFC3339)
	s.record = &WarpRecord{
		PrivateKey: privKey,
		PublicKey:  pubKey,
		Device:     WarpDevice(regResp),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := s.saveRecord(); err != nil {
		return nil, err
	}
	return s.record, nil
}

// LoadRecord loads the WARP device record from disk. Returns nil if no record exists.
func (s *Service) LoadRecord() (*WarpRecord, error) {
	path := s.recordPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var rec WarpRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return nil, err
	}
	s.record = &rec
	return &rec, nil
}

// DeleteRecord removes the WARP device record from memory and disk.
func (s *Service) DeleteRecord() error {
	s.record = nil
	path := s.recordPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	return os.Remove(path)
}

// BindLicense binds a WARP+ license key to the registered device.
func (s *Service) BindLicense(license string) (*WarpRecord, error) {
	if s.record == nil {
		if _, err := s.LoadRecord(); err != nil || s.record == nil {
			return nil, fmt.Errorf("no WARP device registered")
		}
	}
	if license == "" {
		return nil, fmt.Errorf("license cannot be empty")
	}

	rec := s.record
	body, _ := json.Marshal(map[string]string{"license": license})
	url := fmt.Sprintf("%s/%s/reg/%s/account", warpAPIBase, warpAPIVersion, rec.Device.ID)
	req, _ := http.NewRequest("PUT", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("User-Agent", warpClientUA)
	req.Header.Set("CF-Client-Version", warpClientVer)
	req.Header.Set("Authorization", "Bearer "+rec.Device.Token)

	resp, err := httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("license request failed: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("license HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var acct WarpAccount
	if err := json.Unmarshal(respBody, &acct); err != nil {
		return nil, fmt.Errorf("parse license response: %w", err)
	}

	rec.Device.Account.License = acct.License
	rec.Device.Account.AccountType = acct.AccountType
	rec.Device.Account.WarpPlus = acct.WarpPlus
	if acct.ID != "" {
		rec.Device.Account.ID = acct.ID
	}
	if acct.PremiumData > 0 {
		rec.Device.Account.PremiumData = acct.PremiumData
	}
	rec.UpdatedAt = time.Now().Format(time.RFC3339)
	if err := s.saveRecord(); err != nil {
		return nil, err
	}
	return rec, nil
}

// BuildWarpOutbound builds a WireGuard outbound configuration from the WARP record.
func (s *Service) BuildWarpOutbound(endpointHost string, endpointPort int, mtu int) (map[string]interface{}, error) {
	if s.record == nil {
		if _, err := s.LoadRecord(); err != nil || s.record == nil {
			return nil, fmt.Errorf("no WARP device record")
		}
	}
	rec := s.record
	if len(rec.Device.Config.Peers) == 0 {
		return nil, fmt.Errorf("WARP record missing peer config")
	}

	var reserved []int
	if rec.Device.Config.ClientID != "" {
		raw, err := decodeWarpClientID(rec.Device.Config.ClientID)
		if err != nil {
			return nil, fmt.Errorf("parse client_id: %w", err)
		}
		reserved = []int{int(raw[0]), int(raw[1]), int(raw[2])}
	}

	host := endpointHost
	if host == "" {
		host = warpDefaultHost
	}
	port := endpointPort
	if port == 0 {
		port = warpDefaultPort
	}
	if mtu <= 0 || mtu > 1500 {
		mtu = 1280
	}

	v4 := rec.Device.Config.Interface.Addresses.V4
	v6 := rec.Device.Config.Interface.Addresses.V6
	var addresses []string
	if v4 != "" {
		if !strings.Contains(v4, "/") {
			v4 += "/32"
		}
		addresses = append(addresses, v4)
	}
	if v6 != "" {
		if !strings.Contains(v6, "/") {
			v6 += "/128"
		}
		addresses = append(addresses, v6)
	}
	if len(addresses) == 0 {
		return nil, fmt.Errorf("WARP record missing client address")
	}

	peer := rec.Device.Config.Peers[0]
	peerMap := map[string]interface{}{
		"address": host, "port": port, "public_key": peer.PublicKey,
		"allowed_ips": []string{"0.0.0.0/0", "::/0"},
	}
	if len(reserved) == 3 {
		peerMap["reserved"] = reserved
	}

	return map[string]interface{}{
		"type": "wireguard", "tag": "proxy_out",
		"address": addresses, "private_key": rec.PrivateKey,
		"mtu": mtu, "peers": []interface{}{peerMap},
	}, nil
}

// recordPath returns the file path for persisting the WARP device record.
func (s *Service) recordPath() string {
	return filepath.Join(s.baseDir, "warp-account.json")
}

// saveRecord persists the current WARP record to disk atomically.
func (s *Service) saveRecord() error {
	if err := os.MkdirAll(s.baseDir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.record, "", "  ")
	if err != nil {
		return err
	}
	path := s.recordPath()
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	_ = os.Chmod(tmp, 0600)
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	_ = os.Chmod(path, 0600)
	return nil
}

// generatePrivateKey generates a new Curve25519 private key encoded in base64.
func generatePrivateKey() (string, error) {
	var key [32]byte
	if _, err := rand.Read(key[:]); err != nil {
		return "", err
	}
	key[0] &= 248
	key[31] &= 127
	key[31] |= 64
	return base64.StdEncoding.EncodeToString(key[:]), nil
}

// generatePublicKey derives a Curve25519 public key from a base64-encoded private key.
func generatePublicKey(priv string) (string, error) {
	b, err := base64.StdEncoding.DecodeString(priv)
	if err != nil {
		return "", err
	}
	var arr [32]byte
	copy(arr[:], b)
	pub, err := curve25519.X25519(arr[:], curve25519.Basepoint)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(pub), nil
}

// randomHexStr generates a random hexadecimal string of the given byte length.
func randomHexStr(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// httpClient returns an HTTP client with a 30-second timeout.
func httpClient() *http.Client {
	return &http.Client{Timeout: 30 * time.Second}
}

// decodeWarpClientID decodes a WARP client_id from base64 using multiple encoding variants.
func decodeWarpClientID(cid string) ([]byte, error) {
	if cid == "" {
		return nil, fmt.Errorf("empty client_id")
	}
	encs := []*base64.Encoding{
		base64.StdEncoding, base64.RawStdEncoding,
		base64.URLEncoding, base64.RawURLEncoding,
	}
	for _, enc := range encs {
		if b, err := enc.DecodeString(cid); err == nil && len(b) >= 3 {
			return b, nil
		}
	}
	return nil, fmt.Errorf("invalid client_id base64")
}
