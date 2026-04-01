# git-llm-guard

A lightweight daemon that acts as a gatekeeper between AI coding agents and git (and GitHub). It lets you run your favorite LLM in a VM without direct git access, while still allowing controlled git operations like pushing to feature branches.

## TLDR

The guest VM has git but no GitHub authentication. When it wants to push, it drops a request file in a shared folder. The host picks it up, checks the rules, and either executes or rejects it.

## Why?

Running AI agents with full git push access is risky. They could force-push to main, overwrite protected branches, or push to places they shouldn't. On GitHub this can be somewhat managed, but needs to be configured on a repository or organisation basis. That's of little help when working on many repositories. git-llm-guard solves this by:

- Running on the **host** machine with GitHub credentials
- Watching a **shared directory** for git command requests from the guest VM
- Validating every command against configurable **allow/deny rules**

## Installation

### Download a release

Grab the latest binary for your platform from [GitHub Releases](https://github.com/git-llm-guard/git-llm-guard/releases) and place it somewhere on your PATH:

```
curl -Lo git-llm-guard https://github.com/git-llm-guard/git-llm-guard/releases/latest/download/git-llm-guard-$(uname -s)-$(uname -m)
chmod +x git-llm-guard
sudo mv git-llm-guard /usr/local/bin/
```

### Or build from source (alternative)

Requires Go 1.22+:

```
go build -o git-llm-guard ./cmd/git-llm-guard
```

### Create a default config

```
./git-llm-guard --init
```

This creates `~/.git-llm-guard.yaml` with sensible defaults: allows pull, fetch, and push to any branch except main/master/develop, and blocks all force-push variants.

### Edit the config

Open `~/.git-llm-guard.yaml` and set `root` to the directory shared between host and guest (VM).

```yaml
root: /home/you/code/shared

rules:
  allow:
    - "git pull *"
    - "git fetch *"
    - "git push origin *"
  deny:
    - "git push origin main"
    - "git push origin master"
    - "git push origin develop"
    - "*--force*"
    - "* -f"
    - "* -f *"
```

Rules use glob patterns where `*` matches any string (including `/`). Deny rules are checked first, then allow rules. Anything that doesn't match an allow rule is denied by default.

### Per-repo overrides

Place a `config.yaml` inside a repo's `.git-llm-guard/` directory to add repo-specific rules:

```yaml
# <repo>/.git-llm-guard/config.yaml
rules:
  allow:
    - "git push origin feat/*"
  deny:
    - "git push origin release/*"
```

Repo rules are merged with global rules (both deny and allow lists are combined).

### Run it

```
./git-llm-guard
```

Or with a custom config path:

```
./git-llm-guard --config /path/to/config.yaml
```

### Run as a systemd service

Create `/etc/systemd/system/git-llm-guard.service`:

```ini
[Unit]
Description=git-llm-guard
After=network.target

[Service]
ExecStart=/usr/local/bin/git-llm-guard --config /home/you/.git-llm-guard.yaml
User=you
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

Then:

```
sudo systemctl enable --now git-llm-guard
```

## How it works

The guest VM creates a request file in `<repo>/.git-llm-guard/pending/` with a timestamp filename (e.g. `2026-04-01T15-30-00.txt`) containing the git command to run.

git-llm-guard detects the new file, validates the command, and moves it to `<repo>/.git-llm-guard/results/` with the outcome appended:

```
git push origin feat/my-feature
---
status: success
output: |
  To github.com:my-org/project.git
     abc1234..def5678  feat/my-feature -> feat/my-feature
```

Possible statuses: `success`, `denied`, `fail`.

## Development

### Prerequisites

- Go 1.22+

### Build

```
go build -o git-llm-guard ./cmd/git-llm-guard
```

### Test

```
# Unit tests
go test ./...

# Including integration tests
go test -tags integration ./...
```

### Build release binaries

Builds for macOS (Intel + Apple Silicon) and Linux (amd64 + arm64):

```
./scripts/build-release.sh
```

Binaries are placed in `dist/`.
