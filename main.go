package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/snowmerak/GitProfiles/backup"
	"github.com/snowmerak/GitProfiles/key"
)

const envDir = "GITPROFILES_DIR"

// Init creates the base directory structure under baseDir
func Init(baseDir string) error {
	if baseDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		baseDir = filepath.Join(home, ".ssh", "git_profiles")
	}

	dirs := []string{
		filepath.Join(baseDir, "keys"),
		filepath.Join(baseDir, "meta"),
		filepath.Join(baseDir, "backups"),
		filepath.Join(baseDir, "gpg"),
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return err
		}
	}

	// ensure keys.json exists
	keysMeta := filepath.Join(baseDir, "meta", "keys.json")
	if _, err := os.Stat(keysMeta); os.IsNotExist(err) {
		if err := os.WriteFile(keysMeta, []byte("{}"), 0o600); err != nil {
			return err
		}
	}

	return nil
}

// Add generates a key using given algo and stores it under baseDir
func Add(baseDir, algo, name, email, host string) (privatePath, publicPath string, err error) {
	if baseDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", "", err
		}
		baseDir = filepath.Join(home, ".ssh", "git_profiles")
	}

	if algo == "" || name == "" || email == "" {
		return "", "", errors.New("algo, name and email are required")
	}

	gen, err := key.GetKeyGenerator(algo)
	if err != nil {
		return "", "", err
	}

	priv, pub, err := gen.Generate(name, email)
	if err != nil {
		return "", "", err
	}

	keysDir := filepath.Join(baseDir, "keys")
	if err := os.MkdirAll(keysDir, 0o700); err != nil {
		return "", "", err
	}

	baseName := fmt.Sprintf("%s_id_%s", name, strings.ReplaceAll(algo, "-", "_"))
	privatePath = filepath.Join(keysDir, baseName)
	publicPath = privatePath + ".pub"

	if err := os.WriteFile(privatePath, []byte(priv), 0o600); err != nil {
		return "", "", err
	}
	if err := os.WriteFile(publicPath, []byte(pub), 0o644); err != nil {
		return "", "", err
	}

	// update meta
	metaPath := filepath.Join(baseDir, "meta", "keys.json")
	metaBytes, err := os.ReadFile(metaPath)
	if err != nil {
		return privatePath, publicPath, err
	}
	var meta map[string]map[string]string
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		return privatePath, publicPath, fmt.Errorf("failed to parse keys.json: %w", err)
	}

	meta[name] = map[string]string{
		"algo":    algo,
		"private": privatePath,
		"public":  publicPath,
		"email":   email,
		"host":    host,
	}

	out, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return privatePath, publicPath, err
	}
	if err := os.WriteFile(metaPath, out, 0o600); err != nil {
		return privatePath, publicPath, err
	}

	return privatePath, publicPath, nil
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}

	sub := os.Args[1]
	switch sub {
	case "init":
		initCmd := flag.NewFlagSet("init", flag.ExitOnError)
		initCmd.Usage = func() {
			fmt.Fprintf(initCmd.Output(), "Usage: gitprofiles init [flags]\n\nInitialize the gitprofiles directory structure.\n\nFlags:\n")
			initCmd.PrintDefaults()
		}
		base := initCmd.String("base", os.Getenv(envDir), "base directory for gitprofiles (overrides HOME)")
		initCmd.Parse(os.Args[2:])
		if err := Init(*base); err != nil {
			fmt.Fprintln(os.Stderr, "init error:", err)
			os.Exit(1)
		}
		fmt.Println("initialized")
	case "add":
		addCmd := flag.NewFlagSet("add", flag.ExitOnError)
		addCmd.Usage = func() {
			fmt.Fprintf(addCmd.Output(), "Usage: gitprofiles add [flags]\n\nCreate a new git profile with an SSH key.\n\nFlags:\n")
			addCmd.PrintDefaults()
		}
		algo := addCmd.String("algo", "ed25519", "algorithm (ed25519, rsa2048, rsa4096, p256, p384, p521)")
		name := addCmd.String("name", "", "profile name (required)")
		email := addCmd.String("email", "", "email/identity (required)")
		host := addCmd.String("host", "", "host to use in ssh config (e.g. github.com)")
		base := addCmd.String("base", os.Getenv(envDir), "base directory for gitprofiles (overrides HOME)")
		addCmd.Parse(os.Args[2:])
		if *name == "" || *email == "" {
			fmt.Fprintln(os.Stderr, "error: name and email are required")
			addCmd.Usage()
			os.Exit(2)
		}
		priv, pub, err := Add(*base, *algo, *name, *email, *host)
		if err != nil {
			fmt.Fprintln(os.Stderr, "add error:", err)
			os.Exit(1)
		}
		fmt.Printf("private: %s\npublic: %s\n", priv, pub)
	case "ssh-config":
		if len(os.Args) < 3 {
			fmt.Println("usage: gitprofiles ssh-config <status|sync> [flags]")
			os.Exit(2)
		}
		sub := os.Args[2]
		switch sub {
		case "status":
			statusCmd := flag.NewFlagSet("ssh-config status", flag.ExitOnError)
			statusCmd.Usage = func() {
				fmt.Fprintf(statusCmd.Output(), "Usage: gitprofiles ssh-config status [flags]\n\nPreview changes to SSH config.\n\nFlags:\n")
				statusCmd.PrintDefaults()
			}
			cfgPath := statusCmd.String("config", os.ExpandEnv("$HOME/.ssh/config"), "ssh config file path")
			base := statusCmd.String("base", os.Getenv(envDir), "base directory for gitprofiles (overrides HOME)")
			prune := statusCmd.Bool("prune", true, "show entries that would be removed if prune is enabled")
			statusCmd.Parse(os.Args[3:])
			adds, removes, err := PreviewSSHConfig(*base, *cfgPath, *prune)
			if err != nil {
				fmt.Fprintln(os.Stderr, "ssh-config status error:", err)
				os.Exit(1)
			}
			if len(adds) == 0 && len(removes) == 0 {
				fmt.Println("ssh-config is up to date")
				return
			}
			if len(adds) > 0 {
				fmt.Println("Entries to add/update:")
				for _, e := range adds {
					fmt.Printf("  - alias: %s host: %s identity: %s\n", e.Alias, e.HostName, e.IdentityFile)
				}
			}
			if len(removes) > 0 {
				fmt.Println("Entries to remove:")
				for _, a := range removes {
					fmt.Printf("  - alias: %s\n", a)
				}
			}
		case "sync":
			syncCmd := flag.NewFlagSet("ssh-config sync", flag.ExitOnError)
			syncCmd.Usage = func() {
				fmt.Fprintf(syncCmd.Output(), "Usage: gitprofiles ssh-config sync [flags]\n\nApply changes to SSH config.\n\nFlags:\n")
				syncCmd.PrintDefaults()
			}
			cfgPath := syncCmd.String("config", os.ExpandEnv("$HOME/.ssh/config"), "ssh config file path")
			base := syncCmd.String("base", os.Getenv(envDir), "base directory for gitprofiles (overrides HOME)")
			prune := syncCmd.Bool("prune", true, "remove stale managed entries not present in meta")
			syncCmd.Parse(os.Args[3:])
			if err := SyncSSHConfig(*base, *cfgPath, *prune); err != nil {
				fmt.Fprintln(os.Stderr, "ssh-config sync error:", err)
				os.Exit(1)
			}
			fmt.Println("ssh-config synced")
		default:
			fmt.Println("unknown ssh-config subcommand")
			os.Exit(2)
		}
	case "backup":
		bCmd := flag.NewFlagSet("backup", flag.ExitOnError)
		bCmd.Usage = func() {
			fmt.Fprintf(bCmd.Output(), "Usage: gitprofiles backup [flags]\n\nCreate an encrypted backup of profiles.\n\nFlags:\n")
			bCmd.PrintDefaults()
		}
		out := bCmd.String("out", "", "output encrypted backup file (required)")
		base := bCmd.String("base", os.Getenv(envDir), "base directory for gitprofiles (overrides HOME)")
		pass := bCmd.String("pass", "", "passphrase for encryption (optional; prompt if empty)")
		bCmd.Parse(os.Args[2:])
		if *out == "" {
			fmt.Fprintln(os.Stderr, "error: output file is required")
			bCmd.Usage()
			os.Exit(2)
		}
		if *base == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				fmt.Fprintln(os.Stderr, "error getting home dir:", err)
				os.Exit(1)
			}
			*base = filepath.Join(home, ".ssh", "git_profiles")
		}
		var passBytes []byte
		if *pass != "" {
			passBytes = []byte(*pass)
		} else {
			fmt.Fprint(os.Stderr, "Passphrase: ")
			p, err := readPassword()
			if err != nil {
				fmt.Fprintln(os.Stderr, "passphrase error:", err)
				os.Exit(1)
			}
			passBytes = p
		}
		if err := backup.Backup(*base, *out, passBytes); err != nil {
			fmt.Fprintln(os.Stderr, "backup error:", err)
			os.Exit(1)
		}
		fmt.Println("backup written to", *out)
	case "restore":
		rCmd := flag.NewFlagSet("restore", flag.ExitOnError)
		rCmd.Usage = func() {
			fmt.Fprintf(rCmd.Output(), "Usage: gitprofiles restore [flags]\n\nRestore profiles from an encrypted backup.\n\nFlags:\n")
			rCmd.PrintDefaults()
		}
		in := rCmd.String("in", "", "input encrypted backup file (required)")
		base := rCmd.String("base", os.Getenv(envDir), "target base directory for restore (overrides HOME)")
		pass := rCmd.String("pass", "", "passphrase for decryption (optional; prompt if empty)")
		rCmd.Parse(os.Args[2:])
		if *in == "" {
			fmt.Fprintln(os.Stderr, "error: input file is required")
			rCmd.Usage()
			os.Exit(2)
		}
		if *base == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				fmt.Fprintln(os.Stderr, "error getting home dir:", err)
				os.Exit(1)
			}
			*base = filepath.Join(home, ".ssh", "git_profiles")
		}
		var passBytes []byte
		if *pass != "" {
			passBytes = []byte(*pass)
		} else {
			fmt.Fprint(os.Stderr, "Passphrase: ")
			p, err := readPassword()
			if err != nil {
				fmt.Fprintln(os.Stderr, "passphrase error:", err)
				os.Exit(1)
			}
			passBytes = p
		}
		if err := backup.Restore(*in, *base, passBytes); err != nil {
			fmt.Fprintln(os.Stderr, "restore error:", err)
			os.Exit(1)
		}
		fmt.Println("restore completed")
	case "clone":
		cloneCmd := flag.NewFlagSet("clone", flag.ExitOnError)
		cloneCmd.Usage = func() {
			fmt.Fprintf(cloneCmd.Output(), "Usage: gitprofiles clone [flags] <repo>\n\nClone a repository using a specific profile and configure local git settings.\n\nArguments:\n  <repo>      Repository to clone (e.g. owner/repo)\n\nFlags:\n")
			cloneCmd.PrintDefaults()
		}
		profile := cloneCmd.String("profile", "", "profile name to use (required)")
		base := cloneCmd.String("base", os.Getenv(envDir), "base directory for gitprofiles (overrides HOME)")
		cloneCmd.Parse(os.Args[2:])

		args := cloneCmd.Args()
		if len(args) < 1 {
			fmt.Fprintln(os.Stderr, "error: repository argument is required")
			cloneCmd.Usage()
			os.Exit(2)
		}
		repo := args[0]

		if *profile == "" {
			fmt.Fprintln(os.Stderr, "error: profile name is required")
			cloneCmd.Usage()
			os.Exit(2)
		}

		if err := Clone(*base, *profile, repo); err != nil {
			fmt.Fprintln(os.Stderr, "clone error:", err)
			os.Exit(1)
		}
		fmt.Println("clone and configuration completed")
	default:
		printUsage()
		os.Exit(2)
	}
}

func printUsage() {
	fmt.Println("GitProfiles - Manage multiple git profiles and SSH keys")
	fmt.Println("\nUsage:")
	fmt.Println("  gitprofiles <command> [flags]")
	fmt.Println("\nCommands:")
	fmt.Println("  init        Initialize the gitprofiles directory structure")
	fmt.Println("  add         Create a new git profile with an SSH key")
	fmt.Println("  backup      Create an encrypted backup of profiles")
	fmt.Println("  restore     Restore profiles from an encrypted backup")
	fmt.Println("  clone       Clone a repository using a specific profile")
	fmt.Println("  ssh-config  Manage SSH configuration (status/sync)")
	fmt.Println("\nUse 'gitprofiles <command> -h' for more information about a command.")
}
