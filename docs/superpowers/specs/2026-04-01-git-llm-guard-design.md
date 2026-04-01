# git-llm-guard Design Spec

## Overview

A Go daemon that runs on the host machine, watches a shared directory for git command requests from a guest VM (where Claude Code runs), validates them against a configurable allow-list, executes approved commands, and writes results back for the guest to read.

## Problem

Claude Code runs in a guest VM with bypass permissions but should not have direct GitHub access. The guest and host share a directory (`~/code/shared` or similar). The guest has git but no GitHub authentication. We need a controlled way for the guest to request git operations (primarily push) that the host executes after validation.

## Architecture

Single Go binary with three packages:

- **watcher** — filesystem monitoring with fsnotify
- **guard** — command validation and execution
- **config** — YAML configuration parsing

Flow: `watcher detects new file → guard.Validate() → guard.Execute() → move file to results/`

## Configuration

### Global config

Located at `~/.git-llm-guard.yaml` (or specified via `--config` flag):

```yaml
root: /home/user/code/shared
rules:
  allow:
    - "git push origin *"
  deny:
    - "git push origin main"
    - "git push origin master"
    - "git push origin develop"
    - "git push * --force"
    - "git push * --force-with-lease"
```

### Per-repo override

Located at `<repo>/.git-llm-guard/config.yaml`:

```yaml
rules:
  allow:
    - "git push origin feat/*"
    - "git push origin fix/*"
  deny:
    - "git push origin release/*"
```

### Rule evaluation

1. Deny rules are checked first. A match means the command is rejected.
2. Allow rules are checked second. A match means the command is approved.
3. No match means the command is rejected (default deny).
4. Per-repo rules are merged with global rules: repo deny rules are added to global deny rules, repo allow rules are added to global allow rules.
5. Glob patterns use `*` to match any string including `/` (e.g., `feat/*` matches both `feat/my-feature` and `feat/sub/branch`).

## Request/Result File Format

### Request file

- Location: `<repo>/.git-llm-guard/pending/<timestamp>.txt`
- Filename: ISO 8601-ish timestamp, e.g., `2026-04-01T15-30-00.txt`
- Content: the git command to execute, e.g., `git push origin feat/my-feature`

### Result file

The request file is moved from `pending/` to `results/` with the result appended below a `---` separator.

**Success:**

```
git push origin feat/my-feature
---
status: success
output: |
  To github.com:my-org/project.git
     abc1234..def5678  feat/my-feature -> feat/my-feature
```

**Denied:**

```
git push origin main
---
status: denied
reason: matches deny rule "git push origin main"
```

**Fail (git error):**

```
git push origin feat/broken
---
status: fail
output: |
  error: failed to push some refs to 'github.com:my-org/project.git'
```

## Watcher

- Uses `fsnotify` for filesystem event monitoring.
- At startup: watches all directories 1-2 levels deep under root.
- When any watched directory gains a `.git-llm-guard/pending/` subdirectory (either existing at startup or created later), an fsnotify watch is added on that `pending/` directory.
- New repos appearing under root are detected and watched automatically via the same mechanism.
- On `CREATE` event for a file in a `pending/` directory: pass to guard for validation and execution.

## Guard

- `Validate(cmd string, repoPath string) (bool, string)` — checks command against deny rules (global + repo), then allow rules. Returns approved/denied and reason.
- `Execute(cmd string, repoPath string) (string, error)` — runs the command using `os/exec` with the repo root as working directory. The repo root is derived from the path: if the pending file is at `<root>/org/project/.git-llm-guard/pending/file.txt`, the cwd is `<root>/org/project/`.
- After validation and execution, the request file is moved from `pending/` to `results/` with the result appended.

## CLI

```
git-llm-guard --config ~/.git-llm-guard.yaml
```

- `--config`: path to global config file. Defaults to `~/.git-llm-guard.yaml`.
- Runs in the foreground, logs to stdout.
- Designed to be wrapped by systemd/launchd if desired.

## Error Handling

- Invalid or empty request file: status `fail`, reason explains the error.
- Config file missing or unparseable: exit with clear error message.
- fsnotify errors: log to stdout, continue running.
- Git command returns non-zero exit code: status `fail`, output contains stderr/stdout.

## Testing

- Unit tests for glob pattern matching (allow/deny rules, including patterns like `feat/*`).
- Unit tests for config parsing (global config, per-repo config, merge behavior).
- Unit tests for rule evaluation order (deny-first, default-deny).
- Integration test: create a pending file, verify result file is created with correct status and content.
