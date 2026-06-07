package resourcecheck

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"singbox-config-service/internal/pkg/types"
)

// ProberConfig holds configuration for the service.
type ProberConfig struct {
	ResourcesPath string
	DBPath        string
}

// Service orchestrates resource availability checks.
type Service struct {
	checker      *Checker
	store        *Store
	nodeProvider NodeProvider

	resourcesPath string
	dbPath        string
	resources     []ResourceConfig
	resourcesMu   sync.RWMutex

	status     CheckStatus
	statusMu   sync.Mutex
	cancel     context.CancelFunc

	schedulerCtx    context.Context
	schedulerCancel context.CancelFunc
	schedulerMu     sync.Mutex
	wg              sync.WaitGroup
}

// NewService creates a new resource check service.
func NewService(checker *Checker, nodeProvider NodeProvider, cfg ProberConfig) *Service {
	return &Service{
		checker:       checker,
		nodeProvider:  nodeProvider,
		resourcesPath: cfg.ResourcesPath,
		dbPath:        cfg.DBPath,
		status:        CheckStatus{Status: "idle"},
	}
}

// Init loads resources from YAML and opens the database.
func (s *Service) Init() error {
	store, err := NewStore(s.dbPath)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	s.store = store

	resources, err := LoadResources(s.resourcesPath)
	if err != nil {
		return fmt.Errorf("load resources: %w", err)
	}
	s.resourcesMu.Lock()
	s.resources = resources
	s.resourcesMu.Unlock()

	if len(resources) == 0 {
		log.Println("resourcecheck: no resources configured (resources.yaml missing or empty)")
	} else {
		log.Printf("resourcecheck: loaded %d resources from %s", len(resources), s.resourcesPath)
	}
	return nil
}

// ReloadResources reloads resources.yaml without restarting the service.
func (s *Service) ReloadResources() error {
	resources, err := LoadResources(s.resourcesPath)
	if err != nil {
		return err
	}
	s.resourcesMu.Lock()
	s.resources = resources
	s.resourcesMu.Unlock()
	log.Printf("resourcecheck: reloaded %d resources from %s", len(resources), s.resourcesPath)
	return nil
}

// GetResources returns the current resource list.
func (s *Service) GetResources() []ResourceConfig {
	s.resourcesMu.RLock()
	defer s.resourcesMu.RUnlock()
	result := make([]ResourceConfig, len(s.resources))
	copy(result, s.resources)
	return result
}

// RunAll checks all resources through all nodes.
func (s *Service) RunAll(ctx context.Context) error {
	allNodes := s.nodeProvider.GetAllNodes()
	if len(allNodes) == 0 {
		return fmt.Errorf("no nodes available")
	}
	s.resourcesMu.RLock()
	resources := make([]ResourceConfig, len(s.resources))
	copy(resources, s.resources)
	s.resourcesMu.RUnlock()
	if len(resources) == 0 {
		return fmt.Errorf("no resources configured")
	}
	return s.runChecks(ctx, allNodes, resources)
}

// RunForTag checks all resources through a specific node tag.
func (s *Service) RunForTag(ctx context.Context, tag string) error {
	allNodes := s.nodeProvider.GetAllNodes()
	var node *types.ProxyNode
	for i := range allNodes {
		n := &allNodes[i]
		if nodeOutboundTag(n) == tag {
			node = n
			break
		}
	}
	if node == nil {
		return fmt.Errorf("node with tag %q not found", tag)
	}
	s.resourcesMu.RLock()
	resources := make([]ResourceConfig, len(s.resources))
	copy(resources, s.resources)
	s.resourcesMu.RUnlock()
	if len(resources) == 0 {
		return fmt.Errorf("no resources configured")
	}
	return s.runChecks(ctx, []types.ProxyNode{*node}, resources)
}

// runChecks iterates nodes and resources, performing checks.
func (s *Service) runChecks(ctx context.Context, nodes []types.ProxyNode, resources []ResourceConfig) error {
	s.setStatusRunning(len(nodes), len(resources))
	defer s.setStatusIdle()

	for i := range nodes {
		node := &nodes[i]
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		tag := nodeOutboundTag(node)
		s.updateProgress(tag, "", i, len(nodes))

		results, err := s.checker.CheckNodeResources(ctx, node, resources)
		if err != nil {
			log.Printf("resourcecheck: check failed for node %s: %v", tag, err)
			continue
		}

		for _, r := range results {
			if err := s.store.SaveResult(r); err != nil {
				log.Printf("resourcecheck: save result error: %v", err)
			}
		}

		s.updateProgress(tag, "", i+1, len(nodes))
	}

	return nil
}

// GetLatestResults returns all latest results from the store.
func (s *Service) GetLatestResults() ([]CheckResult, error) {
	if s.store == nil {
		return nil, nil
	}
	return s.store.GetLatestResults()
}

// GetResultsForTag returns latest results for a specific tag.
func (s *Service) GetResultsForTag(tag string) ([]CheckResult, error) {
	if s.store == nil {
		return nil, nil
	}
	return s.store.GetResultsForTag(tag)
}

// GetHistory returns check history for a (resource, tag) pair.
func (s *Service) GetHistory(resource, tag string, limit int) ([]CheckResult, error) {
	if s.store == nil {
		return nil, nil
	}
	return s.store.GetHistory(resource, tag, limit)
}

// GetStatus returns a copy of the current status.
func (s *Service) GetStatus() CheckStatus {
	s.statusMu.Lock()
	defer s.statusMu.Unlock()
	return s.status
}

// Stop cancels a running check operation.
func (s *Service) Stop() {
	s.statusMu.Lock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	s.statusMu.Unlock()
}

// StartScheduler starts periodic checks with the given interval.
func (s *Service) StartScheduler(intervalSec int) {
	s.StopScheduler()
	if intervalSec <= 0 {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())

	s.schedulerMu.Lock()
	s.schedulerCtx = ctx
	s.schedulerCancel = cancel
	s.schedulerMu.Unlock()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(time.Duration(intervalSec) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				runCtx, runCancel := context.WithCancel(context.Background())
				s.statusMu.Lock()
				if s.cancel != nil {
					s.cancel()
				}
				s.cancel = runCancel
				s.statusMu.Unlock()

				if err := s.RunAll(runCtx); err != nil {
					log.Printf("resourcecheck: scheduled run error: %v", err)
				}
			}
		}
	}()
	log.Printf("resourcecheck: scheduler started with interval %ds", intervalSec)
}

// StopScheduler stops the periodic check scheduler.
func (s *Service) StopScheduler() {
	s.schedulerMu.Lock()
	cancel := s.schedulerCancel
	s.schedulerCancel = nil
	s.schedulerCtx = nil
	s.schedulerMu.Unlock()

	if cancel != nil {
		cancel()
	}
	s.wg.Wait()
}

// Close shuts down the service.
func (s *Service) Close() error {
	s.StopScheduler()
	s.Stop()
	if s.store != nil {
		return s.store.Close()
	}
	return nil
}

func (s *Service) setStatusRunning(totalNodes, totalChecks int) {
	s.statusMu.Lock()
	s.status = CheckStatus{
		Running:     true,
		Status:      "running",
		TotalNodes:  totalNodes,
		TotalChecks: totalChecks,
	}
	s.statusMu.Unlock()
}

func (s *Service) setStatusIdle() {
	s.statusMu.Lock()
	s.status.Running = false
	s.status.Status = "idle"
	s.status.CompletedNodes = 0
	s.status.Progress = 0
	s.statusMu.Unlock()
}

func (s *Service) updateProgress(tag, resource string, completed, total int) {
	s.statusMu.Lock()
	s.status.Tag = tag
	s.status.Resource = resource
	s.status.CompletedNodes = completed
	if total > 0 {
		s.status.Progress = completed * 100 / total
	}
	s.statusMu.Unlock()
}
