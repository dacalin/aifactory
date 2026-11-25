package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const defaultGlobalYAML = "listen_address: 127.0.0.1:8787\n"

type Global struct {
	ListenAddress string `yaml:"listen_address"`
	DataDir       string `yaml:"-"`
	DatabasePath  string `yaml:"-"`
}

func LoadGlobal() (Global, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Global{}, fmt.Errorf("resolve home directory: %w", err)
	}
	dataDir := filepath.Join(home, ".ai-factory")
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return Global{}, fmt.Errorf("create application data directory: %w", err)
	}
	configPath := filepath.Join(dataDir, "config.yaml")
	data, err := readOrCreateGlobalConfig(configPath)
	if err != nil {
		return Global{}, err
	}
	var global Global
	if err := yaml.Unmarshal(data, &global); err != nil {
		return Global{}, fmt.Errorf("parse global config: %w", err)
	}
	global.ListenAddress, err = validateListenAddress(global.ListenAddress)
	if err != nil {
		return Global{}, err
	}
	global.DataDir = dataDir
	global.DatabasePath = filepath.Join(dataDir, "factory.db")
	return global, nil
}

func readOrCreateGlobalConfig(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err == nil {
		return data, nil
	}
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read global config: %w", err)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if os.IsExist(err) {
		return readGlobalConfig(path)
	}
	if err != nil {
		return nil, fmt.Errorf("create global config: %w", err)
	}
	_, writeErr := file.WriteString(defaultGlobalYAML)
	closeErr := file.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		return nil, fmt.Errorf("write global config: %w", err)
	}
	return []byte(defaultGlobalYAML), nil
}

func readGlobalConfig(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read global config: %w", err)
	}
	return data, nil
}

func validateListenAddress(address string) (string, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		address = "127.0.0.1:8787"
	}
	host, port, err := net.SplitHostPort(address)
	portNumber, portErr := strconv.Atoi(port)
	if err != nil || portErr != nil || portNumber < 1 || portNumber > 65535 || (host != "127.0.0.1" && host != "localhost") {
		return "", fmt.Errorf("global config listen_address must use a loopback host and explicit port")
	}
	return address, nil
}
