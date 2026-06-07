// Package scheduler provides periodic subscription auto-update scheduling.
package scheduler

import (
	"log"
	"sync"
	"time"
)

// Scheduler periodically checks subscriptions and triggers auto-updates.
type Scheduler struct {
	subUpdater   SubscriptionUpdater
	containerMgr ContainerManager
	interval     time.Duration
	stopCh       chan struct{}
	mu           sync.Mutex
	running      bool
}

// New creates a new Scheduler with the given updater, container manager, and config.
func New(subUpdater SubscriptionUpdater, containerMgr ContainerManager, cfg Config) *Scheduler {
	return &Scheduler{
		subUpdater:   subUpdater,
		containerMgr: containerMgr,
		interval:     time.Duration(cfg.Interval) * time.Second,
	}
}

// Start begins the scheduler loop. No-op if already running.
func (s *Scheduler) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.stopCh = make(chan struct{})
	s.mu.Unlock()

	go s.loop()
	log.Println("Scheduler started")
}

// Stop halts the scheduler loop. No-op if not running.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return
	}
	s.running = false
	close(s.stopCh)
	log.Println("Scheduler stopped")
}

// IsRunning reports whether the scheduler loop is active.
func (s *Scheduler) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// loop is the internal ticker loop that triggers subscription checks.
func (s *Scheduler) loop() {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.checkAndAutoUpdateSubscriptions()
		}
	}
}

// checkAndAutoUpdateSubscriptions iterates subscriptions and updates those that are due.
func (s *Scheduler) checkAndAutoUpdateSubscriptions() {
	entries, err := s.subUpdater.LoadAll()
	if err != nil {
		log.Printf("Scheduler: failed to load subscriptions: %v", err)
		return
	}

	for _, entry := range entries {
		if !entry.AutoUpdate {
			continue
		}
		if entry.UpdateInterval <= 0 {
			continue
		}
		if entry.LastUpdated == "" {
			s.updateOne(entry.ID)
			continue
		}

		lastUpdated, err := time.Parse(time.RFC3339, entry.LastUpdated)
		if err != nil {
			s.updateOne(entry.ID)
			continue
		}

		interval := time.Duration(entry.UpdateInterval) * time.Hour
		if time.Since(lastUpdated) >= interval {
			s.updateOne(entry.ID)
		}
	}
}

// updateOne triggers an update for a single subscription by ID.
func (s *Scheduler) updateOne(id string) {
	log.Printf("Scheduler: auto-updating subscription %s", id)
	_, err := s.subUpdater.UpdateOne(id)
	if err != nil {
		log.Printf("Scheduler: failed to update subscription %s: %v", id, err)
	}
}
