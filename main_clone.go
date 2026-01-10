package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Clone clones a repository using the specified profile and configures local git settings.
func Clone(baseDir, profileName, repoArg string) error {
	if baseDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		baseDir = filepath.Join(home, ".ssh", "git_profiles")
	}

	// Load profile
	meta, err := LoadProfiles(baseDir)
	if err != nil {
		return fmt.Errorf("failed to read profiles: %w", err)
	}

	profile, ok := meta[profileName]
	if !ok {
		return fmt.Errorf("profile '%s' not found", profileName)
	}

	host := profile["host"]
	email := profile["email"]
	if host == "" {
		return fmt.Errorf("profile '%s' has no host defined", profileName)
	}

	// Construct SSH config alias
	// This must match the logic in PreviewSSHConfig
	alias := fmt.Sprintf("git-%s-%s", profileName, strings.ReplaceAll(host, ".", "-"))

	// Construct Clone URL: git@alias:repo.git
	cloneURL := fmt.Sprintf("git@%s:%s.git", alias, repoArg)

	fmt.Printf("Cloning %s...\n", cloneURL)

	cmd := exec.Command("git", "clone", cloneURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}

	// Determine directory name
	// repoArg is typically "owner/repo", so we take the last part
	parts := strings.Split(repoArg, "/")
	dirName := parts[len(parts)-1]
	dirName = strings.TrimSuffix(dirName, ".git")

	// Check if directory exists (it should after clone)
	if _, err := os.Stat(dirName); os.IsNotExist(err) {
		return fmt.Errorf("cloned directory '%s' not found", dirName)
	}

	fmt.Printf("Configuring local git config for '%s'...\n", dirName)
	fmt.Printf("  user.name: %s\n", profileName)
	fmt.Printf("  user.email: %s\n", email)

	// git config --local user.name <profileName>
	configNameCmd := exec.Command("git", "config", "--local", "user.name", profileName)
	configNameCmd.Dir = dirName
	if err := configNameCmd.Run(); err != nil {
		return fmt.Errorf("failed to set user.name: %w", err)
	}

	// git config --local user.email <email>
	configEmailCmd := exec.Command("git", "config", "--local", "user.email", email)
	configEmailCmd.Dir = dirName
	if err := configEmailCmd.Run(); err != nil {
		return fmt.Errorf("failed to set user.email: %w", err)
	}

	return nil
}
