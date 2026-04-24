# Getting Started

## Prerequisites

- Go 1.26 or newer
- SSH access to an iStoreOS/OpenWrt router with PassWall2 installed

## Run from source

```bash
go run . --router 192.168.1.1 --user root
```

If you do not provide `--password` or `--key-file`, `inice` prompts for the password in the terminal.

## Build locally

```bash
make build-amd64
```

Or build a universal macOS binary:

```bash
make build-universal
```

## What the command does

- connects to the router with SSH
- reads `uci show passwall2`
- parses proxy nodes
- prints a read-only inventory

## Example output

```text
Connecting to 192.168.1.1:22 as root ...
Reading PassWall2 configuration...
Found 4 proxy nodes
- HK-VMess-WS | Xray | vmess | hk1.example.com:443
- US-Trojan | sing-box | trojan | us1.example.com:8443
```
