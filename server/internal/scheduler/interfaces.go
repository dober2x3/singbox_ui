package scheduler

type SubscriptionUpdater interface {
	LoadAll() ([]SubscriptionEntry, error)
	UpdateOne(id string) (*SubscriptionEntry, error)
}

type SubscriptionEntry struct {
	ID             string
	URL            string
	AutoUpdate     bool
	UpdateInterval int
	LastUpdated    string
}

type ContainerManager interface {
	UpdateAndRestart(name string, configData []byte) error
	Status(name string) (running bool, containerID string)
}
