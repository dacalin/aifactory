package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func buildTarget(version, output string, platform target) (string, error) {
	workspace, err := os.MkdirTemp(output, "build-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(workspace)
	name := "aifactory"
	if platform.OS == "windows" {
		name += ".exe"
	}
	binary := filepath.Join(workspace, name)
	flags := "-s -w -X github.com/dacalin/aifactory/internal/commands.Version=" + strings.TrimPrefix(version, "v")
	process := exec.Command("go", "build", "-trimpath", "-buildvcs=false", "-ldflags", flags, "-o", binary, ".")
	process.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS="+platform.OS, "GOARCH="+platform.Arch, "GOARM=7", "GOAMD64=v1")
	process.Stdout, process.Stderr = os.Stdout, os.Stderr
	if err := process.Run(); err != nil {
		return "", fmt.Errorf("build %s/%s: %w", platform.OS, platform.Arch, err)
	}
	readme, err := os.ReadFile("README.md")
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(binary)
	if err != nil {
		return "", err
	}
	base := filepath.Join(output, "aifactory_"+version+"_"+platform.OS+"_"+platform.Label)
	files := map[string][]byte{name: data, "README.md": readme}
	if platform.OS == "windows" {
		return base + ".zip", writeZip(base+".zip", files)
	}
	return base + ".tar.gz", writeTar(base+".tar.gz", files)
}

func checksum(name string) (string, error) {
	file, err := os.Open(name)
	if err != nil {
		return "", err
	}
	defer file.Close()
	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}
