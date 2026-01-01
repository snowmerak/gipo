package sshconfig

import (
	"fmt"
	"os"
	"strings"
)

// Entry describes an ssh config host entry managed by gitprofiles
type Entry struct {
	Alias        string
	HostName     string
	User         string
	IdentityFile string
}

// AddOrReplaceEntry adds or replaces a managed block for alias in configPath.
// If the file doesn't exist it is created.
func AddOrReplaceEntry(configPath string, e Entry) error {
	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		configPath = fmt.Sprintf("%s/.ssh/config", home)
	}

	b, err := os.ReadFile(configPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	content := string(b)

	begin := fmt.Sprintf("# BEGIN GITPROFILES %s", e.Alias)
	end := fmt.Sprintf("# END GITPROFILES %s", e.Alias)
	blockLines := []string{
		begin,
		fmt.Sprintf("Host %s", e.Alias),
		fmt.Sprintf("    HostName %s", e.HostName),
		fmt.Sprintf("    User %s", e.User),
		fmt.Sprintf("    IdentityFile %s", e.IdentityFile),
		"    IdentitiesOnly yes",
		end,
	}
	block := strings.Join(blockLines, "\n") + "\n"

	if idx := strings.Index(content, begin); idx != -1 {
		// replace existing block
		endIdx := strings.Index(content[idx:], end)
		if endIdx == -1 {
			// malformed, append block
			content = content + "\n" + block
		} else {
			endIdx = idx + endIdx + len(end)
			// include following newline if present
			if endIdx < len(content) && content[endIdx] == '\n' {
				endIdx++
			}
			content = content[:idx] + block + content[endIdx:]
		}
	} else {
		if !strings.HasSuffix(content, "\n") && len(content) > 0 {
			content = content + "\n"
		}
		content = content + block
	}

	return os.WriteFile(configPath, []byte(content), 0o600)
}

// RemoveEntry removes a managed block for alias from configPath. No-op if not found.
func RemoveEntry(configPath, alias string) error {
	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		configPath = fmt.Sprintf("%s/.ssh/config", home)
	}

	b, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	content := string(b)

	begin := fmt.Sprintf("# BEGIN GITPROFILES %s", alias)
	end := fmt.Sprintf("# END GITPROFILES %s", alias)
	if idx := strings.Index(content, begin); idx != -1 {
		endIdx := strings.Index(content[idx:], end)
		if endIdx == -1 {
			// malformed, just remove from begin to end of file
			content = content[:idx]
		} else {
			endIdx = idx + endIdx + len(end)
			if endIdx < len(content) && content[endIdx] == '\n' {
				endIdx++
			}
			content = content[:idx] + content[endIdx:]
		}
		return os.WriteFile(configPath, []byte(content), 0o600)
	}
	return nil
}

// ListEntries parses the config file and returns managed entries
func ListEntries(configPath string) ([]Entry, error) {
	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		configPath = fmt.Sprintf("%s/.ssh/config", home)
	}

	b, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	content := string(b)

	var out []Entry
	lines := strings.Split(content, "\n")
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "# BEGIN GITPROFILES ") {
			alias := strings.TrimPrefix(line, "# BEGIN GITPROFILES ")
			var e Entry
			e.Alias = alias
			// parse following lines until END marker
			for j := i + 1; j < len(lines); j++ {
				l := strings.TrimSpace(lines[j])
				if strings.HasPrefix(l, "HostName ") {
					e.HostName = strings.TrimSpace(strings.TrimPrefix(l, "HostName "))
				} else if strings.HasPrefix(l, "User ") {
					e.User = strings.TrimSpace(strings.TrimPrefix(l, "User "))
				} else if strings.HasPrefix(l, "IdentityFile ") {
					e.IdentityFile = strings.TrimSpace(strings.TrimPrefix(l, "IdentityFile "))
				} else if strings.HasPrefix(l, "# END GITPROFILES ") {
					out = append(out, e)
					i = j
					break
				}
			}
		}
	}
	return out, nil
}
