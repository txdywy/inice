# inice

`inice` is a Go CLI for reading PassWall2 proxy inventory from an iStoreOS/OpenWrt router over SSH.

It currently works in read-only mode:
- connects to the router with SSH
- reads `uci show passwall2`
- parses proxy nodes from the UCI output
- prints a plain inventory of configured nodes

## Features

- SSH connection with password or private key authentication
- Interactive password prompt when no credential is provided
- Narrow read-only SSH command path limited to `uci show passwall2`
- UCI parser for PassWall2 node sections
- Normalized proxy node model covering Xray, sing-box, Shadowsocks, Hysteria2, TUIC, and NaiveProxy records
- GitHub Actions workflow for multi-platform builds

## Current scope

The command does not modify router configuration, upload files, or start remote proxy processes.

## Quick start

```bash
go run . --router 192.168.1.1 --user root
```

If no password or key file is provided, `inice` prompts for the SSH password in the terminal.

## Build

```bash
make build-amd64
make build-arm64
make build-universal
```

## Test

```bash
go test ./... -count=1
```

## Repository layout

- `cmd/` — Cobra CLI entry point
- `internal/config/` — config loading and UCI parsing
- `internal/ssh/` — SSH connection and read-only remote command helper
- `internal/model/` — shared domain types
- `internal/engine/` — probe and runner code
- `internal/report/` — terminal rendering helpers
- `.github/workflows/` — CI configuration
- `testdata/` — parser fixtures
