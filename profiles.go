package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
)

// LoadProfiles reads and parses keys.json from the baseDir using the Profile struct.
// It normalizes the file paths for private/public keys to match the current OS execution environment.
// This ensures that backups restored from Linux to Windows (or vice versa) work correctly.
func LoadProfiles(baseDir string) (map[string]map[string]string, error) {
	metaPath := filepath.Join(baseDir, "meta", "keys.json")
	metaBytes, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, err
	}

	var meta map[string]map[string]string
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		return nil, fmt.Errorf("failed to parse keys.json: %w", err)
	}

	// Normalize paths for the current environment
	for name, info := range meta {
		if priv, ok := info["private"]; ok && priv != "" {
			info["private"] = fixKeyPath(baseDir, priv)
		}
		if pub, ok := info["public"]; ok && pub != "" {
			info["public"] = fixKeyPath(baseDir, pub)
		}
		meta[name] = info
	}

	return meta, nil
}

// fixKeyPath takes a potentially foreign path (e.g. from Linux json on Windows)
// and returns a valid absolute path for the current system, assuming the file lives in <baseDir>/keys/
func fixKeyPath(baseDir, oldPath string) string {
	// 1. Normalize separators to forward slashes to handle all OS paths uniformly
	slashPath := filepath.ToSlash(oldPath)

	// 2. Extract the file name.
	// We use path.Base (not filepath.Base) because we've converted to slashes,
	// and path package handles forward slashes consistently across platforms.
	fileName := path.Base(slashPath)

	// 3. Reconstruct absolute path for current system
	return filepath.Join(baseDir, "keys", fileName)
}
