package singbox

// NamedConfigInfo contains information about a named configuration instance.
type NamedConfigInfo struct {
	Name    string `json:"name"`
	Running bool   `json:"running"`
	Config  string `json:"config"`
}
