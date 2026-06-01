package subscription

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type FileStore struct {
	baseDir string
}

func NewFileStore(baseDir string) *FileStore {
	if baseDir == "" {
		baseDir, _ = os.Getwd()
	}
	return &FileStore{baseDir: baseDir}
}

func (s *FileStore) filePath() string {
	return filepath.Join(s.baseDir, "subscription.json")
}

func (s *FileStore) Load() (*SubscriptionData, error) {
	path := s.filePath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &SubscriptionData{Subscriptions: []SubscriptionEntry{}}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read subscription file: %w", err)
	}
	var subData SubscriptionData
	if err := json.Unmarshal(data, &subData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal subscription data: %w", err)
	}
	if subData.URL != "" && len(subData.Subscriptions) == 0 {
		subData.Subscriptions = []SubscriptionEntry{
			{
				ID:    generateID(),
				Name:  "Default Subscription",
				URL:   subData.URL,
				Nodes: subData.Nodes,
			},
		}
		subData.URL = ""
		subData.Nodes = nil
		_ = s.Save(subData)
	}
	if subData.Subscriptions == nil {
		subData.Subscriptions = []SubscriptionEntry{}
	}
	return &subData, nil
}

func (s *FileStore) Save(data SubscriptionData) error {
	path := s.filePath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal subscription data: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, jsonData, 0644); err != nil {
		return err
	}
	_ = os.Chmod(tmp, 0644)
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	_ = os.Chmod(path, 0644)
	return nil
}
