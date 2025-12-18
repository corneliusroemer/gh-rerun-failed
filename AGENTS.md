# Jules Agents Configuration

This repository contains a GitHub CLI extension for re-running failed workflows.

## Environment Setup

To set up the development environment, run the following script:

```bash
./scripts/setup_jules.sh
```

This script will:
1.  Install `gh` CLI (needed for running the extension).
2.  Install `pre-commit` and configured hooks.
3.  Install `golangci-lint`.
4.  Download Go dependencies.

## Common Tasks

### Running Tests

```bash
go test ./...
```

### Linting

You can run linters using `golangci-lint`:

```bash
golangci-lint run
```

Or run all pre-commit hooks:

```bash
pre-commit run --all-files
```

### Building

```bash
go build
```

## Project Documentation

-   `GEMINI.md`: Development log and project status.
-   `IDEA.md`: Detailed research and technical specification.
