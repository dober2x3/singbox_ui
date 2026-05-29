package services

import (
	"context"
	"net"
	"testing"
	"time"
)

// Start a local UDP listener, reply to handshake packets according to the specified strategy.
// replySize > 0: send a replySize-byte response (to simulate 92-byte valid response or wrong size)
// replySize == 0: do not reply (simulate timeout)
// replySize < 0: only reply on some pings, |replySize| is the sequence number (1-based) to reply on
//
// Returns listener address and close function.
func startFakeWarpUDP(t *testing.T, replySize int) (string, func()) {
	t.Helper()
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	done := make(chan struct{})

	go func() {
		defer close(done)
		buf := make([]byte, 2048)
		count := 0
		for {
			_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
			n, src, err := conn.ReadFromUDP(buf)
			if err != nil {
				return
			}
			count++
			// Confirm it's a valid WARP handshake packet (148 bytes)
			if n != len(warpHandshakePacket) {
				continue
			}

			switch {
			case replySize > 0:
				_, _ = conn.WriteToUDP(make([]byte, replySize), src)
			case replySize == 0:
				// Do not reply
			case replySize < 0:
				// Only reply with 92 bytes on the |replySize|-th ping
				if count == -replySize {
					_, _ = conn.WriteToUDP(make([]byte, warpHandshakeResponseSize), src)
				}
			}
		}
	}()

	return conn.LocalAddr().String(), func() {
		_ = conn.Close()
		<-done
	}
}

// Verify: when the peer always replies with 92 bytes, probe should all succeed
func TestWarpHandshakeProbe_AllSuccess(t *testing.T) {
	addr, stop := startFakeWarpUDP(t, warpHandshakeResponseSize)
	defer stop()

	host, portStr, _ := net.SplitHostPort(addr)
	var port int
	if _, err := net.ResolveUDPAddr("udp", addr); err != nil {
		t.Fatal(err)
	}
	// Simple manual port parsing
	for _, c := range portStr {
		port = port*10 + int(c-'0')
	}

	ctx := context.Background()
	recv, totalRtt := warpHandshakeProbe(ctx, host, port, 3, 500*time.Millisecond)
	if recv != 3 {
		t.Errorf("expected 3 received, got %d", recv)
	}
	// Under loopback, RTT could be nanoseconds or even 0 due to clock resolution, only assert non-negative
	if totalRtt < 0 {
		t.Errorf("expected totalRtt >= 0, got %v", totalRtt)
	}
}

// Verify: when the peer does not reply, probe should all fail but not panic
func TestWarpHandshakeProbe_Timeout(t *testing.T) {
	addr, stop := startFakeWarpUDP(t, 0)
	defer stop()

	host, portStr, _ := net.SplitHostPort(addr)
	var port int
	for _, c := range portStr {
		port = port*10 + int(c-'0')
	}

	ctx := context.Background()
	start := time.Now()
	recv, _ := warpHandshakeProbe(ctx, host, port, 2, 200*time.Millisecond)
	elapsed := time.Since(start)

	if recv != 0 {
		t.Errorf("expected 0 received on timeout, got %d", recv)
	}
	// 2 pings × 200ms ≈ 400ms (plus some scheduling margin)
	if elapsed > 1500*time.Millisecond {
		t.Errorf("probe took too long: %v", elapsed)
	}
}

// Verify: when the peer replies with wrong size, it is not counted as success
func TestWarpHandshakeProbe_WrongSize(t *testing.T) {
	addr, stop := startFakeWarpUDP(t, 50) // Wrong size
	defer stop()

	host, portStr, _ := net.SplitHostPort(addr)
	var port int
	for _, c := range portStr {
		port = port*10 + int(c-'0')
	}

	ctx := context.Background()
	recv, _ := warpHandshakeProbe(ctx, host, port, 3, 300*time.Millisecond)
	if recv != 0 {
		t.Errorf("expected 0 received on wrong size, got %d", recv)
	}
}

// Verify: mixed case - only reply on the 2nd ping, should receive 1 response
func TestWarpHandshakeProbe_PartialSuccess(t *testing.T) {
	addr, stop := startFakeWarpUDP(t, -2) // Only reply on 2nd ping
	defer stop()

	host, portStr, _ := net.SplitHostPort(addr)
	var port int
	for _, c := range portStr {
		port = port*10 + int(c-'0')
	}

	ctx := context.Background()
	recv, totalRtt := warpHandshakeProbe(ctx, host, port, 3, 300*time.Millisecond)
	if recv != 1 {
		t.Errorf("expected 1 received, got %d", recv)
	}
	if totalRtt < 0 {
		t.Errorf("expected totalRtt >= 0, got %v", totalRtt)
	}
}

// Verify: probe exits early when ctx is cancelled
func TestWarpHandshakeProbe_ContextCancel(t *testing.T) {
	addr, stop := startFakeWarpUDP(t, 0)
	defer stop()

	host, portStr, _ := net.SplitHostPort(addr)
	var port int
	for _, c := range portStr {
		port = port*10 + int(c-'0')
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	warpHandshakeProbe(ctx, host, port, 10, 500*time.Millisecond)
	elapsed := time.Since(start)

	// ctx cancelled at 50ms, single Read deadline 500ms. Even if Read is blocked,
	// max wait is 500ms + some scheduling. Should not run full 10 × 500 = 5000ms.
	if elapsed > 1500*time.Millisecond {
		t.Errorf("probe did not honor ctx cancel: %v", elapsed)
	}
}
