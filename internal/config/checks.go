package config

import (
	"fmt"
	"strings"
)

// ChecksConfig declares commands, independent of the checker language or filename.
type ChecksConfig struct {
	Build []string `yaml:"build,omitempty"`
	Run   []string `yaml:"run,omitempty"`
}

func (checks ChecksConfig) validate() error {
	for name, command := range map[string][]string{"build": checks.Build, "run": checks.Run} {
		if len(command) > 0 && strings.TrimSpace(command[0]) == "" {
			return fmt.Errorf("factory.yaml: checks.%s must start with an executable", name)
		}
	}
	return nil
}
