# Development and Testing Guide

This document outlines the development, testing, and dependency management strategy for the Go plugins in this repository. This project uses a **Go Workspace** model, which requires a specific workflow for all commands.

## The Go Workspace Model

The `plugins/` directory is a Go workspace, defined by the `go.work` file at its root. This workspace includes all the individual plugin modules (e.g., `collaborator-plugin`, `teamrepo-plugin`) and the shared `pkg` module.

The key principle of this model is that the `go.work` file is the **single source of truth** for resolving dependencies between local modules. The individual `go.mod` files for each plugin do **not** contain `replace` directives to find the local `pkg` module. This makes the workspace setup clean but requires all commands to be run in a workspace-aware context.

## The Workspace Root: The Golden Rule

To ensure the Go toolchain can see and use the `go.work` file, **all `go` commands MUST be executed from the workspace root directory.**

```sh
# All commands should be run from this location:
/path/to/project/plugins/
```

Running commands from within a subdirectory (e.g., `plugins/cmd/collaborator-plugin/`) will fail, as the Go toolchain will not have the workspace context and will be unable to find the local `pkg` module.

## Dependency Management

### Synchronizing Dependencies (Correct)

When you add or update dependencies in any of the modules, you should synchronize the entire workspace. This is the equivalent of `go mod tidy` for a workspace.

**Terminal Location:** `plugins/`
```sh
go work sync
```

### Tidying Individual Modules (Incorrect)

Running `go mod tidy` inside a specific plugin's directory **will fail**.

```sh
# From plugins/cmd/collaborator-plugin/
go mod tidy # <-- This will fail!
```

This is expected behavior. Without the `go.work` context, the command tries to find the shared `pkg` module on the internet, which is not where it's located.

## Testing

### Running All Tests

To run all tests for every module in the workspace, use the following command.

**Terminal Location:** `plugins/`
```sh
go test -v -cover ./pkg/... ./cmd/collaborator-plugin/... ./cmd/teamrepo-plugin/...
```

### Running Tests for a Specific Module

You can still run tests for a single module, but the command must still be executed from the workspace root.

**Terminal Location:** `plugins/`
```sh
# Example: Run tests only for the collaborator-plugin
go test -v -cover ./cmd/collaborator-plugin/...
```

## Building Binaries

### Building a Single Plugin

To compile a single plugin, run the `go build` command from the workspace root, specifying the path to the plugin's `main.go` file.

**Terminal Location:** `plugins/`
```sh
# Example: Build the collaborator-plugin
go build ./cmd/collaborator-plugin

# Example: Build the teamrepo-plugin
go build ./cmd/teamrepo-plugin
```
This will produce a binary in the `plugins/` directory.

### Building Docker Images

The `Dockerfile` at the root of the `plugins/` directory is also workspace-aware. It copies the entire workspace context to correctly build the target plugin. Builds must be initiated with the `plugins/` directory as the Docker context.

**Terminal Location:** `plugins/`
```sh
# Example: Build the collaborator-plugin Docker image
docker build --build-arg PLUGIN_NAME=collaborator-plugin -t collaborator-plugin:latest .
```

Note that at the root of the `github-provider-kog` repository, there are GitHub Actions workflows that automate the building and publishing of these Docker images.
