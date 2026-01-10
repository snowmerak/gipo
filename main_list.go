package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

	meta, err := LoadProfiles(baseDir)
	if err != nil {
		// If file doesn't exist (e.g. init not run or empty), LoadProfiles returns error.
		// Check if it is a path error or just empty.
		// Actually LoadProfiles uses os.ReadFile which returns PathError if not found.
		if os.IsNotExist(err) {
			fmt.Println("No profiles found.")
			return nil
		}
		// If meta/keys.json doesn't exist, we can treat it as empty.
		// os.ReadFile returns error specific to the file.
		// Let's rely on error message or explicit check if needed.
		// For simplicity, just return error unless it is file not found.
		if strings.Contains(err.Error(), "The system cannot find") || strings.Contains(err.Error(), "no such file") {
			fmt.Println("No profiles found.")
			return nil
		}
		return fmt.Errorf("failed to read profiles: %w", err)
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
