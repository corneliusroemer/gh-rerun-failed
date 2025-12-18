#!/bin/bash
set -e

echo "Setting up environment for Jules..."

# Ensure we are in the root of the repo (where go.mod is)
if [ ! -f "go.mod" ]; then
    echo "Error: go.mod not found. Please run this script from the repository root."
    exit 1
fi

# 1. Install gh CLI
if ! command -v gh &> /dev/null; then
    echo "Installing gh CLI..."
    # Dependencies for gh installation
    (type -p wget >/dev/null || (sudo apt update && sudo apt-get install wget -y))
    sudo mkdir -p -m 755 /etc/apt/keyrings
    wget -qO- https://cli.github.com/packages/githubcli-archive-keyring.gpg | sudo tee /etc/apt/keyrings/githubcli-archive-keyring.gpg > /dev/null
    sudo chmod go+r /etc/apt/keyrings/githubcli-archive-keyring.gpg
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | sudo tee /etc/apt/sources.list.d/github-cli.list > /dev/null
    sudo apt update
    sudo apt install gh -y
    echo "gh CLI installed successfully."
else
    echo "gh CLI is already installed."
fi

# 2. Install pre-commit
if ! command -v pre-commit &> /dev/null; then
    echo "Installing pre-commit..."
    # Try pipx first as it's cleaner for tools
    if command -v pipx &> /dev/null; then
        pipx install pre-commit
    else
        echo "pipx not found, trying pip..."
        pip install pre-commit || echo "Failed to install pre-commit with pip. Please install it manually."
    fi
    # Add to PATH if needed (pipx usually needs ~/.local/bin)
    export PATH="$HOME/.local/bin:$PATH"
else
    echo "pre-commit is already installed."
fi

# 3. Install golangci-lint
if ! command -v golangci-lint &> /dev/null; then
    echo "Installing golangci-lint..."
    # Install binary to $(go env GOPATH)/bin
    # Version v1.64.5 matches .pre-commit-config.yaml
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.64.5
    echo "golangci-lint installed."
else
    echo "golangci-lint is already installed."
fi

# 4. Go dependencies
echo "Downloading Go dependencies..."
go mod download

# 5. Install pre-commit hooks
if command -v pre-commit &> /dev/null; then
    echo "Installing pre-commit hooks..."
    pre-commit install
else
    echo "Warning: pre-commit command not found, skipping hook installation."
fi

echo "Environment setup complete!"
echo "You can now run 'go build' to build the project, or 'pre-commit run --all-files' to run linters."
