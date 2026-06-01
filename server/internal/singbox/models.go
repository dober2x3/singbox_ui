package singbox

type NamedConfigInfo struct {
	Name    string `json:"name"`
	Running bool   `json:"running"`
	Config  string `json:"config"`
}
