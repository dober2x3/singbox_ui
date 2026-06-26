package clashapi

import (
	"encoding/json"
	"os"
	"sync"
)

type PortManager struct {
	basePort int
	assigned map[string]int
	mu       sync.Mutex
}

func NewPortManager(basePort int) *PortManager {
	return &PortManager{
		basePort: basePort,
		assigned: make(map[string]int),
	}
}

func (pm *PortManager) Assign(instanceName string) int {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if port, ok := pm.assigned[instanceName]; ok {
		return port
	}

	used := make(map[int]bool)
	for _, p := range pm.assigned {
		used[p] = true
	}
	port := pm.basePort
	for used[port] {
		port++
	}

	pm.assigned[instanceName] = port
	return port
}

func (pm *PortManager) Release(instanceName string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	delete(pm.assigned, instanceName)
}

func (pm *PortManager) Get(instanceName string) (int, bool) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	port, ok := pm.assigned[instanceName]
	return port, ok
}

func (pm *PortManager) List() map[string]int {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	cp := make(map[string]int, len(pm.assigned))
	for k, v := range pm.assigned {
		cp[k] = v
	}
	return cp
}

func (pm *PortManager) Save(path string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	data, err := json.MarshalIndent(pm.assigned, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (pm *PortManager) Load(path string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &pm.assigned)
}
