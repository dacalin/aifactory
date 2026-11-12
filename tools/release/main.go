package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

type target struct{ OS, Arch, Label string }

var targets = []target{
	{"darwin", "amd64", "amd64"}, {"darwin", "arm64", "arm64"},
	{"linux", "386", "386"}, {"linux", "amd64", "amd64"},
	{"linux", "arm", "armv7"}, {"linux", "arm64", "arm64"},
	{"windows", "386", "386"}, {"windows", "amd64", "amd64"}, {"windows", "arm64", "arm64"},
}

func main() {
	version := flag.String("version", "v0.1.0", "Release version")
	output := flag.String("out", "dist", "Output directory")
	flag.Parse()
	if err := buildRelease(*version, *output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func buildRelease(version, output string) error {
	if version == "" || filepath.Base(version) != version || version == "." || version == ".." {
		return fmt.Errorf("version must be a filename-safe release identifier")
	}
	if err := os.MkdirAll(output, 0755); err != nil {
		return err
	}
	var manifest string
	for _, platform := range targets {
		archive, err := buildTarget(version, output, platform)
		if err != nil {
			return err
		}
		sum, err := checksum(archive)
		if err != nil {
			return err
		}
		manifest += fmt.Sprintf("%s  %s\n", sum, filepath.Base(archive))
		fmt.Println(filepath.Base(archive))
	}
	return os.WriteFile(filepath.Join(output, "SHA256SUMS"), []byte(manifest), 0644)
}
