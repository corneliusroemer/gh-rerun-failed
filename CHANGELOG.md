# Changelog

All notable changes to this project will be documented in this file.

## [0.3.2] - 2025-12-18

### Added
- **Parallel Fetching**: Fetch multiple pages of workflow runs concurrently for faster branch-level scans.
- **Parallel PR Scanning**: Fetch workflow runs for multiple Pull Requests in parallel (10-concurrency limit).
- **Commit Fallback**: Automatically fetch individual commit info if not found in the initial batch, ensuring commit messages are always displayed.

### Changed
- Improved PR fetching to prioritize most recent ones (descending order).
- Refined dry-run table formatting with dynamic terminal width detection.

### Fixed
- Fixed bug where `FetchWorkflowRunsForSha` could miss failed runs due to restricted `per_page` settings.
- Corrected GraphQL query handling using the `Do` method for raw strings.

## [0.3.1] - 2025-12-18

### Added
- Matrix awareness: Identify and display specific failed matrix entries (e.g., `(integration-tests)`).
- Early termination for `--all-prs`: Stop scanning PRs after 5 consecutive empty results to save API budget.

## [0.3.0] - 2025-12-18

### Fixed
- Robust GraphQL query handling using the `Do` method in `go-gh`.

## [0.1.0] - 2025-12-18

- Initial release with core rerun functionality and basic filtering.
