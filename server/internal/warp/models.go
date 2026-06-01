package warp

// WarpInterfaceAddr WARP assigned client address
type WarpInterfaceAddr struct {
	V4 string `json:"v4" example:"172.16.0.2"`
	V6 string `json:"v6" example:"fd01:5ca1:ab1e:0000:0000:0000:0000:0002"`
}

// WarpPeerEndpoint WARP peer endpoint info
type WarpPeerEndpoint struct {
	Host string `json:"host" example:"engage.cloudflareclient.com"`
	V4   string `json:"v4" example:"162.159.192.1"`
	V6   string `json:"v6" example:"2606:4700:100::1"`
}

// WarpPeer WARP peer config
type WarpPeer struct {
	PublicKey string           `json:"public_key" example:"bmXOC+F1FxEMF9dyiK2H5/1SUtzH0cVo9NFiY5pA0Is="`
	Endpoint  WarpPeerEndpoint `json:"endpoint"`
}

// WarpInterface local interface config
type WarpInterface struct {
	Addresses WarpInterfaceAddr `json:"addresses"`
}

// WarpConfig WARP device config from CF API
type WarpConfig struct {
	ClientID  string        `json:"client_id" example:"abc123"`
	Interface WarpInterface `json:"interface"`
	Peers     []WarpPeer    `json:"peers"`
}

// WarpAccount WARP account info
type WarpAccount struct {
	ID          string `json:"id,omitempty" example:"abcd-1234-ef56-7890"`
	License     string `json:"license,omitempty" example:"xxxx-xxxx-xxxx-xxxx"`
	AccountType string `json:"account_type" example:"free"`
	WarpPlus    bool   `json:"warp_plus" example:"false"`
	PremiumData int64  `json:"premium_data,omitempty" example:"0"`
}

// WarpRegisterResponse WARP device registration response
type WarpRegisterResponse struct {
	ID      string      `json:"id" example:"abcd-1234-ef56-7890"`
	Token   string      `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	Account WarpAccount `json:"account"`
	Config  WarpConfig  `json:"config"`
}

// WarpDevice WARP device info (extended)
type WarpDevice struct {
	ID      string      `json:"id" example:"abcd-1234-ef56-7890"`
	Token   string      `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	Account WarpAccount `json:"account"`
	Config  WarpConfig  `json:"config"`
}

// WarpRecord persisted WARP device record
type WarpRecord struct {
	PrivateKey string      `json:"private_key" example:"gL7fV/1GkH+3..."`
	PublicKey  string      `json:"public_key" example:"xTIBdMd5Fq+3..."`
	Device     WarpDevice  `json:"device"`
	CreatedAt  string      `json:"created_at" example:"2026-06-01T00:00:00Z"`
	UpdatedAt  string      `json:"updated_at" example:"2026-06-01T00:00:00Z"`
}

// WarpEndpointResult scanned endpoint result
type WarpEndpointResult struct {
	Host      string `json:"host" example:"162.159.192.1"`
	Port      int    `json:"port" example:"2408"`
	LatencyMs int    `json:"latency_ms" example:"42"`
	LossPct   int    `json:"loss_pct" example:"0"`
	Reachable bool   `json:"reachable" example:"true"`
}

// WarpScanConfig scan configuration
type WarpScanConfig struct {
	SamplePerRange int `json:"sample_per_range" example:"3"`
	PingTimes      int `json:"ping_times" example:"3"`
	Timeout        int `json:"timeout_ms" example:"5000"`
	Concurrency    int `json:"concurrency" example:"10"`
	MaxCandidates  int `json:"max_candidates" example:"10"`
	TopN           int `json:"top_n" example:"5"`
}

// LicenseBindRequest request body for binding a WARP+ license key
type LicenseBindRequest struct {
	License string `json:"license" example:"xxxx-xxxx-xxxx-xxxx"`
}
