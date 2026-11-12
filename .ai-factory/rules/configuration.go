package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// checkerDirectories follows factory.yaml unless an explicit CLI override is set.
func checkerDirectories(root, dataDir, rulesDir string) (string, string, error) {
	var settings struct {
		Dir       string `yaml:"dir"`
		RulesDir  string `yaml:"rules_dir"`
		RuleAgent struct {
			Dir string `yaml:"dir"`
		} `yaml:"rule-agent"`
	}
	data, err := os.ReadFile(filepath.Join(root, "factory.yaml"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", "", err
	}
	if err == nil {
		if err := yaml.Unmarshal(data, &settings); err != nil {
			return "", "", fmt.Errorf("parse factory.yaml: %w", err)
		}
	}
	if dataDir == "" {
		dataDir = settings.Dir
		if dataDir == "" && settings.RulesDir != "" {
			dataDir = filepath.Dir(settings.RulesDir)
		}
		if dataDir == "" {
			dataDir = ".ai-factory"
		}
	}
	if rulesDir == "" {
		rulesDir = settings.RulesDir
		if rulesDir == "" {
			subdirectory := settings.RuleAgent.Dir
			if subdirectory == "" {
				subdirectory = "rules"
			}
			if !filepath.IsLocal(subdirectory) {
				return "", "", fmt.Errorf("rule-agent.dir must be project-relative")
			}
			rulesDir = filepath.Join(dataDir, subdirectory)
		}
	}
	return filepath.Clean(dataDir), filepath.Clean(rulesDir), nil
}
