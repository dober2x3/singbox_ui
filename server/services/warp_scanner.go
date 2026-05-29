package services

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/rand"
	"net"
	"sort"
	"strconv"
	"sync"
	"time"
)

// WarpEndpointResult scanned WARP endpoint result
type WarpEndpointResult struct {
	Host      string `json:"host"`
	Port      int    `json:"port"`
	LatencyMs int    `json:"latency_ms"`
	LossPct   int    `json:"loss_pct"`
	Reachable bool   `json:"reachable"`
}

// WarpScanConfig scan configuration
type WarpScanConfig struct {
	SamplePerRange int           // number of hosts sampled per /24
	PingTimes      int           // number of handshakes per endpoint
	Timeout        time.Duration // single handshake timeout
	Concurrency    int           // concurrent probes
	MaxCandidates  int           // max candidates after shuffling
	TopN           int           // return fastest N
}

// DefaultWarpScanConfig default scan configuration
//
// Design rationale:
// 8 CIDRs × 4 samples = 32 IPs, × 54 ports = 1728 combinations
// Shuffled then truncated to 600, concurrency 128, each probe max 3×1000ms → completes within ~14s.
func DefaultWarpScanConfig() WarpScanConfig {
	return WarpScanConfig{
		SamplePerRange: 4,
		PingTimes:      3,
		Timeout:        1000 * time.Millisecond,
		Concurrency:    128,
		MaxCandidates:  600,
		TopN:           8,
	}
}

// WARP handshake response fixed length (WG MessageResponse)
const warpHandshakeResponseSize = 92

// warpHandshakePacketHex pre-built WireGuard handshake init packet.
//
// From CloudflareWarpSpeedTest, validity depends on the following facts:
//   - All CF WARP edge nodes share the same peer public key
//     "bmXOC+F1FxEMF9dyiK2H5/1SUtzH0JuVo51h2wPfgyo="
//   - MAC1 in the packet only needs to be valid for that public key to be accepted
//   - Any valid init packet will trigger a 92-byte MessageResponse
//
// This avoids needing the wireguard-go dependency during scanning,
// and avoids using our own device private key (scanning only tests connectivity and latency).
const warpHandshakePacketHex = "013cbdafb4135cac96a29484d7a0175ab152dd3e59be35049beadf758b8d48af14ca65f25a168934746fe8bc8867b1c17113d71c0fac5c141ef9f35783ffa5357c9871f4a006662b83ad71245a862495376a5fe3b4f2e1f06974d748416670e5f9b086297f652e6dfbf742fbfc63c3d8aeb175a3e9b7582fbc67c77577e4c0b32b05f92900000000000000000000000000000000"

var warpHandshakePacket []byte

func init() {
	p, err := hex.DecodeString(warpHandshakePacketHex)
	if err != nil {
		// Compile-time constant, error would mean corrupted package
		panic("invalid warp handshake hex: " + err.Error())
	}
	warpHandshakePacket = p
}

// warpIPRanges public WARP IPv4 /24 range prefixes
var warpIPRanges = []string{
	"162.159.192",
	"162.159.193",
	"162.159.195",
	"162.159.204",
	"188.114.96",
	"188.114.97",
	"188.114.98",
	"188.114.99",
}

// warpEndpointPorts CF WARP known available UDP ports (from CloudflareWarpSpeedTest)
var warpEndpointPorts = []int{
	500, 854, 859, 864, 878, 880, 890, 891, 894, 903,
	908, 928, 934, 939, 942, 943, 945, 946, 955, 968,
	987, 988, 1002, 1010, 1014, 1018, 1070, 1074, 1180, 1387,
	1701, 1843, 2371, 2408, 2506, 3138, 3476, 3581, 3854, 4177,
	4198, 4233, 4500, 5279, 5956, 7103, 7152, 7156, 7281, 7559,
	8319, 8742, 8854, 8886,
}

// ScanWarpEndpoints scans WARP available endpoints and sorts by loss rate and latency
//
// Implementation:
// Sends real WireGuard handshake packets to candidate (IP, Port) combinations, checks if
// 92-byte MessageResponse (CF WARP handshake response fixed length) is received.
// This is more accurate than using TCP/443 connectivity as a proxy, directly reflecting UDP path RTT and packet loss.
func ScanWarpEndpoints(ctx context.Context, cfg WarpScanConfig) ([]WarpEndpointResult, error) {
	if cfg.SamplePerRange <= 0 {
		cfg.SamplePerRange = 4
	}
	if cfg.PingTimes <= 0 {
		cfg.PingTimes = 3
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 1000 * time.Millisecond
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 128
	}
	if cfg.MaxCandidates <= 0 {
		cfg.MaxCandidates = 600
	}
	if cfg.TopN <= 0 {
		cfg.TopN = 8
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Sample several host numbers per /24 to get an IP list
	ips := make([]string, 0, len(warpIPRanges)*cfg.SamplePerRange)
	for _, prefix := range warpIPRanges {
		sample := cfg.SamplePerRange
		if sample > 254 {
			sample = 254
		}
		used := make(map[int]bool, sample)
		for len(used) < sample {
			host := r.Intn(254) + 1
			if used[host] {
				continue
			}
			used[host] = true
			ips = append(ips, fmt.Sprintf("%s.%d", prefix, host))
		}
	}

	// IP x Port Cartesian product
	type candidate struct {
		host string
		port int
	}
	cands := make([]candidate, 0, len(ips)*len(warpEndpointPorts))
	for _, ip := range ips {
		for _, p := range warpEndpointPorts {
			cands = append(cands, candidate{ip, p})
		}
	}
	// Shuffle + truncate to avoid a single scan taking too long
	r.Shuffle(len(cands), func(i, j int) { cands[i], cands[j] = cands[j], cands[i] })
	if len(cands) > cfg.MaxCandidates {
		cands = cands[:cfg.MaxCandidates]
	}

	type probe struct {
		host     string
		port     int
		received int
		totalRtt time.Duration
	}
	results := make([]probe, 0, len(cands))
	var mu sync.Mutex

	sem := make(chan struct{}, cfg.Concurrency)
	var wg sync.WaitGroup

	// Use labeled break to correctly exit outer for;
	// semaphore acquisition inside select so ctx cancellation can interrupt blocking
dispatch:
	for _, cand := range cands {
		select {
		case <-ctx.Done():
			break dispatch
		case sem <- struct{}{}:
		}
		wg.Add(1)
		go func(host string, port int) {
			defer wg.Done()
			defer func() { <-sem }()

			recv, rtt := warpHandshakeProbe(ctx, host, port, cfg.PingTimes, cfg.Timeout)
			if recv == 0 {
				return
			}
			mu.Lock()
			results = append(results, probe{
				host:     host,
				port:     port,
				received: recv,
				totalRtt: rtt,
			})
			mu.Unlock()
		}(cand.host, cand.port)
	}
	wg.Wait()

	// Sort: first by loss rate ascending, then by average RTT ascending
	sort.Slice(results, func(i, j int) bool {
		li := cfg.PingTimes - results[i].received
		lj := cfg.PingTimes - results[j].received
		if li != lj {
			return li < lj
		}
		ai := results[i].totalRtt / time.Duration(results[i].received)
		aj := results[j].totalRtt / time.Duration(results[j].received)
		return ai < aj
	})

	topN := cfg.TopN
	if topN > len(results) {
		topN = len(results)
	}
	out := make([]WarpEndpointResult, 0, topN)
	for i := 0; i < topN; i++ {
		p := results[i]
		avg := p.totalRtt / time.Duration(p.received)
		loss := (cfg.PingTimes - p.received) * 100 / cfg.PingTimes
		out = append(out, WarpEndpointResult{
			Host:      p.host,
			Port:      p.port,
			LatencyMs: int(avg / time.Millisecond),
			LossPct:   loss,
			Reachable: true,
		})
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("no available WARP endpoints found")
	}
	return out, nil
}

// warpHandshakeProbe sends PingTimes WG handshake packets to a single UDP endpoint,
// returns received valid responses count and cumulative RTT.
//
// Valid response = MessageResponse with length exactly 92 bytes;
// any other return packet (ICMP unreachable triggered Read error, or wrong length) is considered lost.
func warpHandshakeProbe(ctx context.Context, host string, port int, times int, timeout time.Duration) (received int, totalRtt time.Duration) {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	d := net.Dialer{Timeout: timeout}
	conn, err := d.DialContext(ctx, "udp", addr)
	if err != nil {
		return 0, 0
	}
	defer conn.Close()

	buf := make([]byte, 1024)
	for i := 0; i < times; i++ {
		select {
		case <-ctx.Done():
			return
		default:
		}

		start := time.Now()
		if _, err := conn.Write(warpHandshakePacket); err != nil {
			// Transient write failure (e.g. ENETUNREACH): consistent with upstream handshake(),
			// only count this ping as lost, next ping will still attempt
			continue
		}
		if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
			continue
		}
		n, err := conn.Read(buf)
		if err != nil {
			// Timeout or ICMP unreachable: count as lost, continue to next attempt
			continue
		}
		if n != warpHandshakeResponseSize {
			continue
		}
		received++
		totalRtt += time.Since(start)
	}
	return
}

// WarpEndpointPorts returns WARP available port list (for frontend display)
func WarpEndpointPorts() []int {
	ports := make([]int, len(warpEndpointPorts))
	copy(ports, warpEndpointPorts)
	return ports
}
