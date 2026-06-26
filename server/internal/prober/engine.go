package prober

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
)

// SO_BINDTODEVICE forces a socket to use a specific network interface (Linux).
// Value is 25 (0x19) on Linux; not exported in Go's syscall package.
const soBindToDevice = 0x19

// Prober periodically probes network nodes and tracks their availability.
type Prober struct {
	config    Config
	nodes     sync.Map // map[string]types.ProbeNode
	results   sync.Map // map[string]*types.ProbeResult
	history   sync.Map // map[string]*nodeHistory
	running   int32
	stopChan  chan struct{}
	wg        sync.WaitGroup
	mu        sync.Mutex
	semaphore chan struct{}
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewProber creates a new Prober with the given configuration.
func NewProber(config Config) *Prober {
	ctx, cancel := context.WithCancel(context.Background())
	return &Prober{
		config:    config,
		stopChan:  make(chan struct{}),
		semaphore: make(chan struct{}, config.Concurrent),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Start begins the periodic probe loop. No-op if already running.
func (p *Prober) Start() {
	if !atomic.CompareAndSwapInt32(&p.running, 0, 1) {
		return
	}

	p.mu.Lock()
	p.ctx, p.cancel = context.WithCancel(context.Background())
	p.stopChan = make(chan struct{})
	p.mu.Unlock()

	log.Println("Prober started")

	p.wg.Add(1)
	go p.probeLoop()
}

// Stop halts the probe loop and waits for goroutines to finish. No-op if not running.
func (p *Prober) Stop() {
	if !atomic.CompareAndSwapInt32(&p.running, 1, 0) {
		return
	}

	p.mu.Lock()
	if p.cancel != nil {
		p.cancel()
	}
	close(p.stopChan)
	p.mu.Unlock()

	p.wg.Wait()
	log.Println("Prober stopped")
}

// IsRunning reports whether the probe loop is currently active.
func (p *Prober) IsRunning() bool {
	return atomic.LoadInt32(&p.running) == 1
}
