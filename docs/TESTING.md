# Testing

## Current test coverage

The repository currently includes a parser test for PassWall2 UCI output:

- `internal/config/uci_parser_test.go`

It verifies that sample `uci show passwall2` output is normalized into the expected proxy node records.

## Run tests

```bash
go test ./... -count=1
```

For the Makefile test target:

```bash
make test
```

## Test data

The fixture used by the parser test is:

- `testdata/uci_show_passwall2.txt`

## What to test next

Good follow-up coverage areas are:

- SSH command execution against a mock server
- config loading precedence
- parser handling for more PassWall2 node variants
- output formatting for the report helpers
