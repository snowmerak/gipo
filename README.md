# gipo (Git Profiles)

`gipo` is a CLI tool designed to manage multiple Git profiles (SSH keys, user identities) easily. It simplifies the process of working with different Git identities (e.g., personal, work, client projects) on the same machine by automating SSH key generation, SSH config management, and repository cloning.

## Features

- **Profile Management**: Create and manage multiple Git profiles, each with its own SSH key and user identity (name, email).
- **Automatic SSH Key Generation**: Supports various algorithms (ed25519, rsa, p256, etc.) to generate secure keys for your profiles.
- **SSH Config Sync**: Automatically updates your `~/.ssh/config` file with unique aliases for each profile, keeping your SSH configuration clean and organized.
- **Smart Cloning**: The `clone` command uses the profile's SSH alias to clone repositories and automatically configures the local repository's `user.name` and `user.email`.
- **Secure Backup & Restore**: Create encrypted backups of your profiles and keys to easily migrate to another machine.
- **Preview Changes**: `status` command lets you see what changes will be made to your SSH config before applying them.

## Installation

### From Source

Ensure you have Go installed (1.25+ recommended).

```bash
go install github.com/snowmerak/gipo@latest
```

Or clone the repository and build:

```bash
git clone https://github.com/snowmerak/gipo.git
cd gipo
go build -o gipo .
```

## Usage

### 1. Initialize

First, initialize the storage directory (default: `~/.ssh/git_profiles`).

```bash
gipo init
```

### 2. Add a Profile

Create a new profile. This will generate an SSH key pair.

```bash
gipo add --name work --email me@company.com --host github.com
```

- `--name`: A unique name for the profile (e.g., `work`, `personal`).
- `--email`: The email address associated with the Git identity.
- `--host`: The Git host (e.g., `github.com`, `gitlab.com`).
- `--algo`: (Optional) Key algorithm (default: `ed25519`).

### 3. Sync SSH Config

Apply the changes to your `~/.ssh/config` file. This creates aliases like `git-work-github-com`.

```bash
# Preview changes
gipo status

# Apply changes
gipo sync
```

### 4. Clone a Repository

Use `gipo clone` to clone a repository using a specific profile.

```bash
gipo clone --profile work owner/repo
```

This command does two things:
1.  Clones the repo using the SSH alias (e.g., `git@git-work-github-com:owner/repo.git`).
2.  Sets the local git config (`user.name` and `user.email`) for that repository to match the profile.

### 5. List Profiles

View all registered profiles.

```bash
gipo list
```

### 6. Backup & Restore

Backup your profiles to an encrypted file.

```bash
gipo backup --out profiles.enc
```

Restore from a backup.

```bash
gipo restore --in profiles.enc
```

## How it Works

`gipo` works by creating a dedicated SSH config entry for each profile. For example, if you add a profile named `work` for `github.com`, `gipo` generates an SSH key and adds an entry to `~/.ssh/config` like this:

```ssh
Host git-work-github-com
  HostName github.com
  User git
  IdentityFile /Users/you/.ssh/git_profiles/keys/work_id_ed25519
```

When you run `gipo clone --profile work owner/repo`, it translates the clone URL to `git@git-work-github-com:owner/repo.git`, forcing Git to use the correct SSH key. After cloning, it locally sets `user.name` and `user.email` in the new repository.

## License

Mozilla Public License Version 2.0
