# llm-git-firewall

This is a lightweight daemon that acts as a gatekeeper between AI coding agents and git (and GitHub). It lets you run your favorite LLM in a VM without direct git access, while still allowing controlled git operations like pushing to feature branches.

This acts as a proxy for git (and Github) and only lets through allowed operations.

The guest VM has git but no GitHub authentication. When it wants to push, it drops a request file in a shared folder. The host picks it up, checks the rules, and either executes or rejects it.

## Why?

Running AI agents with full git push access is risky. They could force-push to main, overwrite protected branches, or push to places they shouldn't. On GitHub this can be somewhat managed, but needs to be configured on a repository or organisation basis. That's of little help when working on many repositories. llm-git-firewall solves this by:

- Running on the **host** machine with GitHub credentials
- Watching a **shared directory** for git command requests from the guest VM
- Validating every command against configurable **allow/deny rules**

## Requirements

- A VM (guest) where you run your LLM
- A shared directory between your guest and host. Typically this is where all repositories you're working on lives.

## Installation

### Homebrew

```sh
brew tap emiloberg/tap
brew install llm-git-firewall
```

Create and edit the config

```sh
# Create default config
llm-git-firewall --init

# Edit config — set 'root' to your shared directory.
# See further down this README for config
vim ~/.llm-git-firewall.yaml
```

#### Run it

As a service (in background, survives reboot):

```sh
brew services start llm-git-firewall
```

Or as a normal CLI :

```sh
llm-git-firewall
```

If running as a service

```sh
# Check status
brew services info llm-git-firewall

# View logs
tail -f /usr/local/var/log/llm-git-firewall.log

# Stop
brew services stop llm-git-firewall
```

### Or build from source (alternative)

Requires Go 1.22+:

```
go build -o llm-git-firewall ./cmd/llm-git-firewall
```

Create default config as per above

### Edit the config

Open `~/.llm-git-firewall.yaml` and set `root` to the directory shared between host and guest (VM).

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

Place a `config.yaml` inside a repo's `.llm-git-firewall/` directory to add repo-specific rules:

```yaml
# <repo>/.llm-git-firewall/config.yaml
rules:
  allow:
    - "git push origin feat/*"
  deny:
    - "git push origin release/*"
```

Repo rules are merged with global rules (both deny and allow lists are combined).

## Instruct your LLM

Instruct your LLM on how to use this.

Add this to your `AGENTS.md`/`CLAUDE.md`

```
## Special git instructions

You have access to git, but you're not authenticated towards GitHub. You can commit and operate locally, but not fetch, pull and push as you normally would.

When you want to push or pull code upstreams there's a special routine:

Create a file in `<repo-root>/.llm-git-firewall/pending` with the current date/time as name and the command you want to execute as content. E.g.

echo "git push origin feat/1234" > $(date +%Y-%m-%dT%H-%M-%S).txt

A worker will read this, check it against an allow-list and perform it. When done it will create a file, with the same filename, in `<repo-root>/.llm-git-firewall/results` containing the results of the operation.

### Git rules
* Never commit on main, master, staging or any other long lived branches. You will not be able to push them.
* Always create and commit on specific feature branches.

```

## How it works

The guest VM creates a request file in `<repo>/.llm-git-firewall/pending/` with a timestamp filename (e.g. `2026-04-01T15-30-00.txt`) containing the git command to run.

llm-git-firewall detects the new file, validates the command, and moves it to `<repo>/.llm-git-firewall/results/` with the outcome appended:

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
go build -o llm-git-firewall ./cmd/llm-git-firewall
```

### Test

```
# Unit tests
go test ./...

# Including integration tests
go test -tags integration ./...
```
