# Development

## Tooling

- `go test ./... -count=1`
- `make test`
- `make build-amd64`
- `make build-arm64`
- `make build-universal`

## Key packages

- `cmd/root.go` — CLI wiring and read-only SSH flow
- `internal/config/user_config.go` — config loading and environment binding
- `internal/config/uci_parser.go` — PassWall2 UCI parsing and node normalization
- `internal/ssh/client.go` — SSH authentication and command execution
- `internal/report/renderer.go` — terminal formatting helpers

## Build notes

The Makefile builds macOS binaries with `CGO_ENABLED=0` and a configurable `MACOSX_DEPLOYMENT_TARGET`.

## CI

GitHub Actions runs tests on pushes to `main`, pull requests, and manual dispatches, then builds binaries for:

- linux/amd64
- linux/arm64
- windows/amd64
- darwin/amd64
- darwin/arm64
- freebsd/amd64
- netbsd/amd64
- openbsd/amd64

## Working with the parser

The parser fixture lives in `testdata/uci_show_passwall2.txt` and is covered by `internal/config/uci_parser_test.go`.

That test is the best place to extend coverage when PassWall2 fields change.
