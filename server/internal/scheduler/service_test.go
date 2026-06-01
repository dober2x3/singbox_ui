package scheduler

import (
	"testing"
	"time"

	"singbox-config-service/internal/subscription"
)

type mockSubUpdater struct {
	entries []subscription.SubscriptionEntry
	updated map[string]bool
}

func (m *mockSubUpdater) LoadAll() ([]subscription.SubscriptionEntry, error) {
	return m.entries, nil
}

func (m *mockSubUpdater) UpdateOne(id string) (*subscription.SubscriptionEntry, error) {
	if m.updated == nil {
		m.updated = make(map[string]bool)
	}
	m.updated[id] = true
	return &m.entries[0], nil
}

type mockContainerManager struct{}

func (m *mockContainerManager) UpdateAndRestart(name string, configData []byte) error {
	return nil
}

func (m *mockContainerManager) Status(name string) (running bool, containerID string) {
	return false, ""
}

func TestScheduler_autoUpdateTrigger(t *testing.T) {
	subMock := &mockSubUpdater{
		entries: []subscription.SubscriptionEntry{
			{
				ID: "test", AutoUpdate: true, UpdateInterval: 1,
				LastUpdated: time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
			},
		},
	}
	containerMock := &mockContainerManager{}
	sched := New(subMock, containerMock)

	sched.checkAndAutoUpdateSubscriptions()

	if !subMock.updated["test"] {
		t.Error("Subscription should have been auto-updated")
	}
}

func TestScheduler_skipIfNotDue(t *testing.T) {
	subMock := &mockSubUpdater{
		entries: []subscription.SubscriptionEntry{
			{
				ID: "test", AutoUpdate: true, UpdateInterval: 24,
				LastUpdated: time.Now().Format(time.RFC3339),
			},
		},
	}
	sched := New(subMock, &mockContainerManager{})
	sched.checkAndAutoUpdateSubscriptions()
	if subMock.updated["test"] {
		t.Error("Subscription should NOT be updated if not due")
	}
}

func TestScheduler_skipIfAutoUpdateDisabled(t *testing.T) {
	subMock := &mockSubUpdater{
		entries: []subscription.SubscriptionEntry{
			{
				ID: "test", AutoUpdate: false,
			},
		},
	}
	sched := New(subMock, &mockContainerManager{})
	sched.checkAndAutoUpdateSubscriptions()
	if subMock.updated["test"] {
		t.Error("Subscription should NOT be updated if AutoUpdate is false")
	}
}

func TestScheduler_startStop(t *testing.T) {
	subMock := &mockSubUpdater{}
	sched := New(subMock, &mockContainerManager{})
	sched.Start()
	if !sched.IsRunning() {
		t.Error("Scheduler should be running after Start()")
	}
	sched.Stop()
	if sched.IsRunning() {
		t.Error("Scheduler should not be running after Stop()")
	}
}
