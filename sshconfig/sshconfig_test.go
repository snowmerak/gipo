package sshconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAddReplaceRemove(t *testing.T) {
	d := t.TempDir()
	cfg := filepath.Join(d, "config")

	entry := Entry{
		Alias:        "git-acct1",
		HostName:     "github.com",
		User:         "git",
		IdentityFile: "/home/user/.ssh/git_profiles/keys/acct1",
	}

	// Add
	if err := AddOrReplaceEntry(cfg, entry); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !stringsContains(string(b), entry.Alias) {
		t.Fatalf("config doesn't contain alias: %s", string(b))
	}

	// Replace with different HostName
	entry.HostName = "github.enterprise"
	if err := AddOrReplaceEntry(cfg, entry); err != nil {
		t.Fatal(err)
	}
	b, err = os.ReadFile(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !stringsContains(string(b), entry.HostName) {
		t.Fatalf("config not updated: %s", string(b))
	}

	// List
	list, err := ListEntries(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Alias != entry.Alias {
		t.Fatalf("list entries mismatch: %#v", list)
	}

	// Remove
	if err := RemoveEntry(cfg, entry.Alias); err != nil {
		t.Fatal(err)
	}
	b, err = os.ReadFile(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if stringsContains(string(b), entry.Alias) {
		t.Fatalf("alias still present after remove: %s", string(b))
	}
}

func stringsContains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || (s != "" && (len(sub) == 0 || len(s) >= len(sub) && (s != "" && (indexOf(s, sub) >= 0)))))
}

func indexOf(s, sub string) int {
	for i := range s {
		if i+len(sub) <= len(s) && s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
