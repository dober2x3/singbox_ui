package services

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
)

// Cloudflare WARP API constants
const (
	warpAPIBase     = "https://api.cloudflareclient.com"
	warpAPIVersion  = "v0a2158"
	warpClientUA    = "okhttp/3.12.1"
	warpClientVer   = "a-6.11-2158"
	warpDefaultHost = "engage.cloudflareclient.com"
	warpDefaultPort = 2408
)

// WarpInterfaceAddr WARP assigned client address
type WarpInterfaceAddr struct {
	V4 string `json:"v4"`
	V6 string `json:"v6"`
}

// WarpPeerEndpoint WARP peer endpoint info
type WarpPeerEndpoint struct {
	Host string `json:"host"`
	V4   string `json:"v4"`
	V6   string `json:"v6"`
}

// WarpPeer WARP peer config
type WarpPeer struct {
	PublicKey string           `json:"public_key"`
	Endpoint  WarpPeerEndpoint `json:"endpoint"`
}

// WarpInterface local interface config
type WarpInterface struct {
	Addresses WarpInterfaceAddr `json:"addresses"`
}

// WarpConfig device WG config
type WarpConfig struct {
	ClientID  string        `json:"client_id"`
	Interface WarpInterface `json:"interface"`
	Peers     []WarpPeer    `json:"peers"`
}

// WarpAccount account info
type WarpAccount struct {
	ID                string `json:"id"`
	License           string `json:"license"`
	AccountType       string `json:"account_type"`
	PremiumData       int64  `json:"premium_data"`
	WarpPlus          bool   `json:"warp_plus"`
	ReferralCount     int    `json:"referral_count"`
	ReferralRenewalEn int64  `json:"referral_renewal_countdown"`
}

// WarpRegisterResponse /reg response body
type WarpRegisterResponse struct {
	ID      string      `json:"id"`
	Token   string      `json:"token"`
	Account WarpAccount `json:"account"`
	Config  WarpConfig  `json:"config"`
}

// WarpRecord locally persisted WARP device record
type WarpRecord struct {
	PrivateKey string               `json:"private_key"`
	PublicKey  string               `json:"public_key"`
	Device     WarpRegisterResponse `json:"device"`
	CreatedAt  string               `json:"created_at"`
	UpdatedAt  string               `json:"updated_at,omitempty"`
}

func warpRecordPath() string {
	return filepath.Join(singboxDir, "warp-account.json")
}

// LoadWarpRecord reads cached WARP record (returns nil, nil if not found)
func LoadWarpRecord() (*WarpRecord, error) {
	path := warpRecordPath()
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
	return &rec, nil
}

// SaveWarpRecord saves WARP record
// File permission 0600: contains WireGuard private key and Cloudflare Bearer Token, must be read-only by process owner
// Atomic write: write to temp file then rename, to avoid half-written files from crashes destroying existing records
func SaveWarpRecord(rec *WarpRecord) error {
	if err := os.MkdirAll(singboxDir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	path := warpRecordPath()
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	// Fallback chmod, some filesystems/umask may relax initial permissions
	_ = os.Chmod(tmp, 0600)
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	// Chmod again after rename to ensure target file permissions are always 0600
	_ = os.Chmod(path, 0600)
	return nil
}

// DeleteWarpRecord deletes local WARP record
func DeleteWarpRecord() error {
	path := warpRecordPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	return os.Remove(path)
}

// warpHTTPClient unified HTTP client
func warpHTTPClient() *http.Client {
	return &http.Client{Timeout: 30 * time.Second}
}

// randomHexStr returns n-byte random hex string (for secrets)
func randomHexStr(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// RegisterWarpDevice registers a new WARP device via Cloudflare API
func RegisterWarpDevice() (*WarpRecord, error) {
	privKey, err := generatePrivateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}
	pubKey, err := generatePublicKey(privKey)
	if err != nil {
		return nil, fmt.Errorf("failed to generate public key: %w", err)
	}

	serial, err := randomHexStr(8)
	if err != nil {
		return nil, err
	}

	body := map[string]interface{}{
		"key":           pubKey,
		"install_id":    "",
		"fcm_token":     "",
		"tos":           time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		"model":         "PC",
		"serial_number": serial,
		"locale":        "en_US",
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/%s/reg", warpAPIBase, warpAPIVersion)
	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("User-Agent", warpClientUA)
	req.Header.Set("CF-Client-Version", warpClientVer)

	resp, err := warpHTTPClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("WARP registration request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("WARP registration failed HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var regResp WarpRegisterResponse
	if err := json.Unmarshal(respBody, &regResp); err != nil {
		return nil, fmt.Errorf("failed to parse WARP response: %w", err)
	}
	if len(regResp.Config.Peers) == 0 {
		return nil, fmt.Errorf("WARP response missing peer config")
	}

	now := time.Now().Format(time.RFC3339)
	rec := &WarpRecord{
		PrivateKey: privKey,
		PublicKey:  pubKey,
		Device:     regResp,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := SaveWarpRecord(rec); err != nil {
		return nil, fmt.Errorf("failed to save WARP record: %w", err)
	}
	return rec, nil
}

// BindWarpLicense binds WARP+ license to a registered WARP device
func BindWarpLicense(rec *WarpRecord, license string) (*WarpRecord, error) {
	if rec == nil {
		return nil, fmt.Errorf("please register a WARP device first")
	}
	if license == "" {
		return nil, fmt.Errorf("license cannot be empty")
	}
	if rec.Device.Token == "" || rec.Device.ID == "" {
		return nil, fmt.Errorf("current WARP record missing token or device ID")
	}

	// Step 1: PUT /reg/{id}/account — update license
	body, err := json.Marshal(map[string]string{"license": license})
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/%s/reg/%s/account", warpAPIBase, warpAPIVersion, rec.Device.ID)
	req, err := http.NewRequest("PUT", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("User-Agent", warpClientUA)
	req.Header.Set("CF-Client-Version", warpClientVer)
	req.Header.Set("Authorization", "Bearer "+rec.Device.Token)

	resp, err := warpHTTPClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("license binding request failed: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read license response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("license binding failed HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	var acct WarpAccount
	if err := json.Unmarshal(respBody, &acct); err != nil {
		return nil, fmt.Errorf("failed to parse license response: %w", err)
	}

	// Field-level merge: CF's PUT /account in different API versions occasionally returns partial Account,
	// if we replace entirely, historical fields (premium_data / referral_count etc.) could be zeroed.
	// Strategy: plan-related fields use PUT response; numerical stats fields only overwrite when non-zero.
	rec.Device.Account.License = acct.License
	rec.Device.Account.AccountType = acct.AccountType
	rec.Device.Account.WarpPlus = acct.WarpPlus
	if acct.ID != "" {
		rec.Device.Account.ID = acct.ID
	}
	if acct.PremiumData > 0 {
		rec.Device.Account.PremiumData = acct.PremiumData
	}
	if acct.ReferralCount > 0 {
		rec.Device.Account.ReferralCount = acct.ReferralCount
	}
	if acct.ReferralRenewalEn > 0 {
		rec.Device.Account.ReferralRenewalEn = acct.ReferralRenewalEn
	}

	// Step 2: GET /reg/{id} — some license bindings need to refresh device info
	// On successful refresh, use it as authoritative state to overwrite Account; otherwise keep the field-level merge result
	_ = refreshWarpDevice(rec)

	rec.UpdatedAt = time.Now().Format(time.RFC3339)
	if err := SaveWarpRecord(rec); err != nil {
		return nil, fmt.Errorf("failed to save WARP record: %w", err)
	}
	return rec, nil
}

// refreshWarpDevice fetches latest device info and updates config/account
func refreshWarpDevice(rec *WarpRecord) error {
	url := fmt.Sprintf("%s/%s/reg/%s", warpAPIBase, warpAPIVersion, rec.Device.ID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", warpClientUA)
	req.Header.Set("CF-Client-Version", warpClientVer)
	req.Header.Set("Authorization", "Bearer "+rec.Device.Token)
	resp, err := warpHTTPClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var latest WarpRegisterResponse
	if err := json.Unmarshal(data, &latest); err != nil {
		return err
	}
	// Defense: only overwrite old record when latest has complete device info,
	// otherwise keep old data to avoid partial responses breaking persisted config
	if latest.ID == "" || len(latest.Config.Peers) == 0 {
		return fmt.Errorf("refresh returned incomplete record")
	}
	// token is not in GET /reg response, need to preserve it
	token := rec.Device.Token
	rec.Device = latest
	rec.Device.Token = token
	return nil
}

// decodeWarpClientID tries multiple base64 encodings to parse client_id
// CF API may return base64 without padding, or URL-safe variants
func decodeWarpClientID(cid string) ([]byte, error) {
	if cid == "" {
		return nil, fmt.Errorf("empty client_id")
	}
	encs := []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	}
	for _, enc := range encs {
		if b, err := enc.DecodeString(cid); err == nil && len(b) >= 3 {
			return b, nil
		}
	}
	return nil, fmt.Errorf("client_id is not valid base64")
}

// BuildWarpOutbound converts WARP record to sing-box wireguard outbound config
func BuildWarpOutbound(rec *WarpRecord, endpointHost string, endpointPort int, mtu int) (map[string]interface{}, error) {
	if rec == nil {
		return nil, fmt.Errorf("WARP record is nil")
	}
	if len(rec.Device.Config.Peers) == 0 {
		return nil, fmt.Errorf("WARP record missing peer config")
	}

	// Parse client_id → reserved three bytes
	// When client_id is non-empty but decode fails, return error instead of silently constructing incomplete outbound
	var reserved []int
	if rec.Device.Config.ClientID != "" {
		raw, err := decodeWarpClientID(rec.Device.Config.ClientID)
		if err != nil {
			return nil, fmt.Errorf("failed to parse WARP client_id: %w", err)
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
	addresses := []string{}
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
		"address":     host,
		"port":        port,
		"public_key":  peer.PublicKey,
		"allowed_ips": []string{"0.0.0.0/0", "::/0"},
	}
	if len(reserved) == 3 {
		peerMap["reserved"] = reserved
	}

	return map[string]interface{}{
		"type":        "wireguard",
		"tag":         "proxy_out",
		"address":     addresses,
		"private_key": rec.PrivateKey,
		"mtu":         mtu,
		"peers":       []interface{}{peerMap},
	}, nil
}
