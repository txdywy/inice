# Configuration

`inice` reads configuration from three sources, with the following priority:

1. CLI flags
2. Environment variables and config file values
3. Built-in defaults

## Config file

By default, `inice` looks for `~/.inice.yaml`.

Example:

```yaml
router:
  host: 192.168.1.1
  port: 22
  user: root
  password: ""
  key_file: ""

shadow:
  preferred_core: auto
  base_port: 20000

testing:
  concurrency: 10
  timeout: 3s
  warmup_probes: 3
  measurement_probes: 5

output:
  mode: static
  format: table
```

## Environment variables

- `INICE_ROUTER_HOST`
- `INICE_ROUTER_PORT`
- `INICE_ROUTER_USER`
- `INICE_SSH_PASSWORD`
- `INICE_SSH_KEY_FILE`

## CLI flags

- `--router` — router hostname or IP
- `--port` — SSH port
- `--user` — SSH username
- `--password` — SSH password
- `--key-file` — SSH private key path
- `--config` — config file path

## Defaults

- router port: `22`
- router user: `root`
- shadow preferred core: `auto`
- shadow base port: `20000`
- test concurrency: `10`
- test timeout: `3s`
- warmup probes: `3`
- measurement probes: `5`
- output mode: `static`
- output format: `table`

## Notes

If no password or key file is supplied, the CLI prompts for the password interactively in the terminal.

The current default workflow is read-only and does not modify the router.
