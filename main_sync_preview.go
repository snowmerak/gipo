package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/snowmerak/gipo/sshconfig"
)

// PreviewSSHConfig returns entries to add/update and aliases to remove (if prune==true).
// It compares the desired state (from keys.json) with the current state (from ssh config file).
// baseDir is the root directory of gitprofiles (default: ~/.ssh/git_profiles).
// cfgPath is the path to the ssh config file (default: ~/.ssh/config).
// prune indicates whether to remove managed entries that are no longer in keys.json.
func PreviewSSHConfig(baseDir, cfgPath string, prune bool) (adds []sshconfig.Entry, removes []string, err error) {
	if baseDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, nil, err
		}
		baseDir = filepath.Join(home, ".ssh", "git_profiles")
	}
	if cfgPath == "" {
		home, _ := os.UserHomeDir()
		cfgPath = filepath.Join(home, ".ssh", "config")
	}

	meta, err := LoadProfiles(baseDir)
	if err != nil {
		return nil, nil, err
	}

	desired := make(map[string]sshconfig.Entry)
	for name, info := range meta {
		host := info["host"]
		priv := info["private"]
		if host == "" || priv == "" {
			continue
		}
		alias := fmt.Sprintf("git-%s-%s", name, strings.ReplaceAll(host, ".", "-"))
		desired[alias] = sshconfig.Entry{Alias: alias, HostName: host, User: "git", IdentityFile: priv}
	}

	existing, err := sshconfig.ListEntries(cfgPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, nil, err
	}

	// compute adds/updates
	for alias, de := range desired {
		found := false
		for _, ex := range existing {
			if ex.Alias == alias {
				found = true
				// if any field differs, mark for add/update
				if ex.HostName != de.HostName || ex.User != de.User || ex.IdentityFile != de.IdentityFile {
					adds = append(adds, de)
				}
				break
			}
		}
		if !found {
			adds = append(adds, de)
		}
	}

	if prune {
		for _, ex := range existing {
			if strings.HasPrefix(ex.Alias, "git-") {
				if _, ok := desired[ex.Alias]; !ok {
					removes = append(removes, ex.Alias)
				}
			}
		}
	}

	return adds, removes, nil
}

// SyncSSHConfig applies the changes calculated by PreviewSSHConfig to the ssh config file.
// It adds or updates entries for profiles and removes stale entries if prune is true.
func SyncSSHConfig(baseDir, cfgPath string, prune bool) error {
	adds, removes, err := PreviewSSHConfig(baseDir, cfgPath, prune)
	if err != nil {
		return err
	}
	for _, e := range adds {
		if err := sshconfig.AddOrReplaceEntry(cfgPath, e); err != nil {
			return err
		}
	}
	for _, a := range removes {
		if err := sshconfig.RemoveEntry(cfgPath, a); err != nil {
			return err
		}
	}
	return nil
}
