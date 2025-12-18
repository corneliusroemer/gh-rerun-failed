# gh-rerun-failed

A GitHub CLI extension to rerun failed workflow runs across branches, commits, and PRs with powerful filtering.

## Installation

```bash
gh extension install github.com/cli/gh-rerun-failed
```

## Usage

```bash
# Rerun all failed runs on the current branch from the last 24 hours
gh rerun-failed --since 24h

# Rerun failed runs on 'main' branch, limited to 10 runs
gh rerun-failed --branch main --limit 10

# Rerun failed runs from all open PRs
gh rerun-failed --all-prs

# Rerun failed runs for a specific PR
gh rerun-failed --pr 123

# Dry run to see what would be rerun
gh rerun-failed --since 1h --dry-run
```

## Flags

- `-b, --branch string`: Filter runs by branch
- `-L, --limit int`: Limit the number of runs to process
- `-s, --since duration`: Only process runs since this duration (e.g., `24h`, `1h`, `30m`). Uses Go duration format.
- `--pr int`: Filter runs by PR number (fetches failed runs for the PR's head commit)
- `--all-prs`: Process runs for all open PRs
- `--dry-run`: Show what would be done without performing re-runs
- `--failed-only`: Only rerun failed jobs within a run (default `true`)
- `--include-drafts`: Include draft PRs when using `--all-prs` (default `false`)

## Development

### Prerequisites

- [Go](https://golang.org/doc/install)
- [GitHub CLI](https://cli.github.com/)

### Running Locally

```bash
go run main.go --since 1h --dry-run
```

### Testing

```bash
go test ./...
```
