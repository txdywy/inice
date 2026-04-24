# Architecture

`inice` is organized around a read-only SSH inventory flow.

## Runtime flow

1. Load local configuration from `~/.inice.yaml` and environment variables.
2. Apply CLI flag overrides.
3. Prompt for an SSH password if no password or key file is available.
4. Connect to the router over SSH.
5. Run `uci show passwall2`.
6. Parse the UCI output into normalized proxy node records.
7. Print the discovered nodes as a read-only inventory.

## Package overview

### `cmd`
The Cobra entry point. It wires configuration, SSH authentication, signal handling, and the main read-only command flow.

### `internal/config`
Handles local config loading and parsing of PassWall2 UCI output.

### `internal/model`
Defines the normalized node model, test result types, and user configuration structures shared across the project.

### `internal/ssh`
Wraps SSH connection management and terminal password prompting. The package also contains an SFTP helper for future extension.

### `internal/engine`
Contains probe helpers and a runner for proxy testing workflows. These packages are currently not part of the default CLI path.

### `internal/report`
Contains render helpers for terminal output.

## Data model

PassWall2 nodes are normalized into `model.ProxyNode`, which abstracts over the differences between Xray-style fields and sing-box-style fields in UCI output.

## CI / builds

The project uses GitHub Actions to run tests and build binaries for multiple operating systems and architectures, including older macOS deployment targets.

The local Makefile mirrors the build targets for macOS binaries and universal output.
