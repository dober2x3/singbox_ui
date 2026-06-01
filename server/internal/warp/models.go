package warp

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

// WarpConfig WARP device config from CF API
type WarpConfig struct {
	ClientID  string        `json:"client_id"`
	Interface WarpInterface `json:"interface"`
	Peers     []WarpPeer    `json:"peers"`
}

// WarpAccount WARP account info
type WarpAccount struct {
	ID          string `json:"id,omitempty"`
	License     string `json:"license,omitempty"`
	AccountType string `json:"account_type"`
	WarpPlus    bool   `json:"warp_plus"`
	PremiumData int64  `json:"premium_data,omitempty"`
}

// WarpRegisterResponse WARP device registration response
type WarpRegisterResponse struct {
	ID      string      `json:"id"`
	Token   string      `json:"token"`
	Account WarpAccount `json:"account"`
	Config  WarpConfig  `json:"config"`
}

// WarpDevice WARP device info (extended)
type WarpDevice struct {
	ID      string      `json:"id"`
	Token   string      `json:"token"`
	Account WarpAccount `json:"account"`
	Config  WarpConfig  `json:"config"`
}

// WarpRecord persisted WARP device record
type WarpRecord struct {
	PrivateKey string      `json:"private_key"`
	PublicKey  string      `json:"public_key"`
	Device     WarpDevice  `json:"device"`
	CreatedAt  string      `json:"created_at"`
	UpdatedAt  string      `json:"updated_at"`
}

// WarpEndpointResult scanned endpoint result
type WarpEndpointResult struct {
	Host      string `json:"host"`
	Port      int    `json:"port"`
	LatencyMs int    `json:"latency_ms"`
	LossPct   int    `json:"loss_pct"`
	Reachable bool   `json:"reachable"`
}

// WarpScanConfig scan configuration
type WarpScanConfig struct {
	SamplePerRange int           `json:"sample_per_range"`
	PingTimes      int           `json:"ping_times"`
	Timeout        int           `json:"timeout_ms"`
	Concurrency    int           `json:"concurrency"`
	MaxCandidates  int           `json:"max_candidates"`
	TopN           int           `json:"top_n"`
}
