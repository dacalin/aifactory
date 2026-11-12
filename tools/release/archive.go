package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"os"
	"sort"
)

func fileNames(files map[string][]byte) []string {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func writeTar(name string, files map[string][]byte) (err error) {
	file, err := os.Create(name)
	if err != nil {
		return err
	}
	compressed := gzip.NewWriter(file)
	archive := tar.NewWriter(compressed)
	defer func() { err = errors.Join(err, archive.Close(), compressed.Close(), file.Close()) }()
	for _, name := range fileNames(files) {
		mode := int64(0755)
		if name == "README.md" {
			mode = 0644
		}
		data := files[name]
		if err := archive.WriteHeader(&tar.Header{Name: name, Mode: mode, Size: int64(len(data))}); err != nil {
			return err
		}
		if _, err := archive.Write(data); err != nil {
			return err
		}
	}
	return nil
}

func writeZip(name string, files map[string][]byte) (err error) {
	file, err := os.Create(name)
	if err != nil {
		return err
	}
	archive := zip.NewWriter(file)
	defer func() { err = errors.Join(err, archive.Close(), file.Close()) }()
	for _, name := range fileNames(files) {
		header := &zip.FileHeader{Name: name, Method: zip.Deflate}
		header.SetMode(0755)
		if name == "README.md" {
			header.SetMode(0644)
		}
		entry, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}
		if _, err := entry.Write(files[name]); err != nil {
			return err
		}
	}
	return nil
}
