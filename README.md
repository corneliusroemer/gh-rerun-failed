# gh-rerun-failed

A GitHub CLI extension to rerun failed workflow runs across branches, commits, and PRs with powerful filtering and detailed insights.

## Installation

```bash
gh extension install corneliusroemer/gh-action-rerun-failed-extension
```

## Usage

```bash
# Rerun failed runs on the current branch from the last 24 hours
gh rerun-failed --since 24h

# Rerun failed runs on 'main' branch, limited to 10 runs
gh rerun-failed --branch main --limit 10

# Rerun failed runs for a specific PR
gh rerun-failed --pr 123

# Rerun failed runs from all open PRs
gh rerun-failed --all-prs

# Dry run to see a detailed table of what would be rerun
gh rerun-failed --since 1h --dry-run
```

## Management

### Update to the latest version
```bash
gh extension upgrade rerun-failed
```

### List installed extensions
```bash
gh extension list
```

### Remove the extension
```bash
gh extension remove rerun-failed
```

## Flags

- `-R, --repo string`: Select another repository using the `[HOST/]OWNER/REPO` format
- `-b, --branch string`: Filter runs by branch
- `-L, --limit int`: Limit the number of runs to process
- `-s, --since duration`: Only process runs since this duration (e.g., `24h`, `1h`). Uses Go duration format.
- `--pr int`: Filter runs by PR number (fetches failed runs for the PR's head commit)
- `--all-prs`: Process runs for all open PRs
- `--dry-run`: Show a detailed summary table without performing re-runs
- `--failed-only`: Only rerun failed jobs within a run (default `true`)
- `--include-cancelled`: Also process cancelled runs (default `false`)
- `--include-timed-out`: Also process timed-out runs (default `false`)
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
