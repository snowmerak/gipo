package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"text/tabwriter"
)

// ListProfiles prints all registered profiles in a tabular format.
func ListProfiles(baseDir string) error {
	if baseDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		baseDir = filepath.Join(home, ".ssh", "git_profiles")
	}

	metaPath := filepath.Join(baseDir, "meta", "keys.json")
	metaBytes, err := os.ReadFile(metaPath)
	if err != nil {
		// If file doesn't exist, just say no profiles
		if os.IsNotExist(err) {
			fmt.Println("No profiles found.")
			return nil
		}
		return fmt.Errorf("failed to read profiles: %w", err)
	}

	var meta map[string]map[string]string
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		return fmt.Errorf("failed to parse profiles: %w", err)
	}

	if len(meta) == 0 {
		fmt.Println("No profiles found.")
		return nil
	}

	// Use tabwriter for aligned output
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tEMAIL\tHOST\tALGO")

	// Sort by name
	var names []string
	for k := range meta {
		names = append(names, k)
	}
	sort.Strings(names)

	for _, name := range names {
		info := meta[name]
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", name, info["email"], info["host"], info["algo"])
	}
	w.Flush()

	return nil
}
