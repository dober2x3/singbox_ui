package scheduler

import "singbox-config-service/internal/subscription"

// SubscriptionUpdater is implemented by the subscription service for auto-updates.
type SubscriptionUpdater interface {
	LoadAll() ([]subscription.SubscriptionEntry, error)
	UpdateOne(id string) (*subscription.SubscriptionEntry, error)
}

// ContainerManager manages container lifecycle for scheduler.
type ContainerManager interface {
	UpdateAndRestart(name string, configData []byte) error
	Status(name string) (running bool, containerID string)
}
