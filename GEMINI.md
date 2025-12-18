# gh-rerun-failed Development Log

## Project Goal
Build a user-friendly GitHub CLI extension to rerun failed actions across commits, branches, and PRs with powerful filters like `--limit 10` and `--since 24h`.

## Architecture
- **Language**: Go
- **CLI Framework**: Cobra
- **GitHub API**: `cli/go-gh/v2` (REST and GraphQL)
- **Structure**:
    - `main.go`: Entry point, flag definitions.
    - `internal/gh`: GitHub API client wrapper and interface.
    - `internal/rerunner`: Core business logic for fetching and triggering reruns.

## Key Features Implemented
- [x] Fetch failed, cancelled, and timed-out runs.
- [x] Concurrent execution of reruns (max 5) to respect rate limits.
- [x] Parallel fetching of different statuses to improve performance.
- [x] PR-level targeting (`--pr`, `--all-prs`).
- [x] Matrix job awareness: shows exactly which matrix entries failed.
- [x] Robust GraphQL: fixed raw query handling using `Do` method.
- [x] Metadata in logs: Run #, Branch, SHA, CreatedAt, URL.
- [x] Rate limit accounting and trace logging.
- [x] Repository override via `--repo`.

## Performance & Rate Limits
- **Parallelism**: Fetching `failure`, `cancelled`, and `timed_out` runs is now done in parallel.
- **Rate Limit Budget**: The tool reports rate limit consumption at the end.
- **Large Repositories**: Pagination is handled up to 1000 runs per status.

## Future Considerations & Live Testing Ideas
1. **Interactive Mode**: Allow users to select which runs to rerun from a list (e.g., using `gum` or `bubbletea`).
2. **Exponential Backoff**: Improve error handling for 403/429 responses.
3. **Better Filtering**: Add filtering by workflow name or job name.
4. **Live Testing**:
    - Use a dedicated test repository with intentionally failing actions.
    - Test against large public repos (like `loculus-project/loculus`) in `--dry-run` mode to verify performance.

## Usage Examples
```bash
# Recent failures on current branch
gh-rerun-failed --since 2h

# Batch rerun for all open PRs (dry run)
gh-rerun-failed --all-prs --dry-run

# Target specific repo
gh-rerun-failed --repo owner/repo --branch main
```
