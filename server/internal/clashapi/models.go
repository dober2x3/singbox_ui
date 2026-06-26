package clashapi

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
)

// === Proxies ===

type ProxyGroup struct {
	Type string   `json:"type"`
	Now  string   `json:"now"`
	All  []string `json:"all"`
}

type ProxiesResponse struct {
	Proxies map[string]ProxyGroup `json:"proxies"`
}

// === Proxy Detail ===

type ProxyDetail struct {
	Type    string   `json:"type"`
	Name    string   `json:"name"`
	History []struct {
		Time      string  `json:"time"`
		Delay     int     `json:"delay"`
		MeanDelay float64 `json:"meanDelay"`
	} `json:"history"`
}

// === Delay ===

type DelayResponse struct {
	Delay int `json:"delay"`
}

func (d *DelayResponse) UnmarshalJSON(data []byte) error {
	var raw map[string]int
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	for _, v := range raw {
		d.Delay = v
		return nil
	}
	return fmt.Errorf("empty delay response")
}

// === Traffic (бинарный формат Clash API) ===

type TrafficMessage struct {
	Up   int64
	Down int64
}

func (m *TrafficMessage) UnmarshalBinary(data []byte) error {
	if len(data) < 16 {
		return fmt.Errorf("traffic message too short: %d bytes", len(data))
	}
	m.Up = int64(binary.BigEndian.Uint64(data[0:8]))
	m.Down = int64(binary.BigEndian.Uint64(data[8:16]))
	return nil
}

// === Memory ===

type MemoryMessage struct {
	Inuse   int64 `json:"inuse"`
	OSLimit int64 `json:"oslimit"`
}

// === Connections ===

type ConnectionMeta struct {
	Network string `json:"network"`
	Type    string `json:"type"`
	Source  string `json:"source"`
	DstIP   string `json:"dstIP"`
	DstPort string `json:"dstPort"`
	Host    string `json:"host"`
	Process string `json:"process"`
}

type Connection struct {
	ID          string         `json:"id"`
	Metadata    ConnectionMeta `json:"metadata"`
	Upload      int64          `json:"upload"`
	Download    int64          `json:"download"`
	Start       string         `json:"start"`
	Chains      []string       `json:"chains"`
	Rule        string         `json:"rule"`
	RulePayload string         `json:"rulePayload"`
}

type ConnectionsResponse struct {
	DownloadTotal int64        `json:"download_total"`
	UploadTotal   int64        `json:"upload_total"`
	Connections   []Connection `json:"connections"`
}

// === Logs ===

type LogEntry struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

// === Rules ===

type Rule struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
	Proxy   string `json:"proxy"`
}

type RulesResponse struct {
	Rules []Rule `json:"rules"`
}

// === Config ===

type ConfigResponse struct {
	Mode string `json:"mode"`
}
