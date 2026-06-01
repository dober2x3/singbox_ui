package scheduler

import (
	"log"
	"sync"
	"time"
)

type Scheduler struct {
	subUpdater   SubscriptionUpdater
	containerMgr ContainerManager
	interval     time.Duration
	stopCh       chan struct{}
	mu           sync.Mutex
	running      bool
}

func New(subUpdater SubscriptionUpdater, containerMgr ContainerManager) *Scheduler {
	return &Scheduler{
		subUpdater:   subUpdater,
		containerMgr: containerMgr,
		interval:     60 * time.Second,
	}
}

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

func (s *Scheduler) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

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

func (s *Scheduler) updateOne(id string) {
	log.Printf("Scheduler: auto-updating subscription %s", id)
	_, err := s.subUpdater.UpdateOne(id)
	if err != nil {
		log.Printf("Scheduler: failed to update subscription %s: %v", id, err)
	}
}
