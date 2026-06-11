//go:build docker

package singbox

import (
	"fmt"

	"singbox-config-service/internal/pkg/config"
)

func NewRuntime(cfg *config.AppConfig) (Runtime, error) {
	return nil, fmt.Errorf("Docker runtime is not available in this build")
}
