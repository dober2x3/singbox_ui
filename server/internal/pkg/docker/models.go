package docker

// ContainerInfo holds summary information about a Docker container.
type ContainerInfo struct {
	Name        string `json:"name"`
	ContainerID string `json:"container_id"`
	State       string `json:"state"`
	Status      string `json:"status"`
	Created     int64  `json:"created"`
}
