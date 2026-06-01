package singbox

// NamedConfigInfo contains information about a named configuration instance.
type NamedConfigInfo struct {
	Name    string `json:"name" example:"my-config" description:"Configuration name"`
	Running bool   `json:"running" example:"false" description:"Whether the container is running"`
	Config  string `json:"config" example:"{\"log\":{}}" description:"Raw configuration JSON"`
}

// VersionResponse sing-box version response
type VersionResponse struct {
	Version string `json:"version" example:"1.10.0"`
}

// StatusResponse container status response
type StatusResponse struct {
	Running     bool   `json:"running" example:"true"`
	ContainerID string `json:"containerId,omitempty" example:"abc123def456"`
}

// LogResponse container logs response
type LogResponse struct {
	Logs string `json:"logs" example:"2026/06/01 12:00:00 starting..."`
}

// NamedInstanceResponse response for named container operations
type NamedInstanceResponse struct {
	Message     string `json:"message" example:"Operation completed"`
	Name        string `json:"name" example:"my-config"`
	ContainerID string `json:"containerId,omitempty" example:"abc123def456"`
}

// CheckConfigResponse validation result response
type CheckConfigResponse struct {
	Valid   bool   `json:"valid" example:"true"`
	Message string `json:"message" example:"Config is valid"`
}
