package warp

import (
	"context"
	"net"
	"testing"
	"time"
)

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
			if n != len(warpHandshakePacket) {
				continue
			}

			switch {
			case replySize > 0:
				_, _ = conn.WriteToUDP(make([]byte, replySize), src)
			case replySize == 0:
				// Do not reply
			case replySize < 0:
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

func TestWarpHandshakeProbe_AllSuccess(t *testing.T) {
	addr, stop := startFakeWarpUDP(t, warpHandshakeResponseSize)
	defer stop()

	host, portStr, _ := net.SplitHostPort(addr)
	var port int
	for _, c := range portStr {
		port = port*10 + int(c-'0')
	}

	ctx := context.Background()
	recv, totalRtt := warpHandshakeProbe(ctx, host, port, 3, 500*time.Millisecond)
	if recv != 3 {
		t.Errorf("expected 3 received, got %d", recv)
	}
	if totalRtt < 0 {
		t.Errorf("expected totalRtt >= 0, got %v", totalRtt)
	}
}

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
	if elapsed > 1500*time.Millisecond {
		t.Errorf("probe took too long: %v", elapsed)
	}
}

func TestWarpHandshakeProbe_WrongSize(t *testing.T) {
	addr, stop := startFakeWarpUDP(t, 50)
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

func TestWarpHandshakeProbe_PartialSuccess(t *testing.T) {
	addr, stop := startFakeWarpUDP(t, -2)
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

	if elapsed > 1500*time.Millisecond {
		t.Errorf("probe did not honor ctx cancel: %v", elapsed)
	}
}
