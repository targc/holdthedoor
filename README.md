# HoldTheDoor

Remote shell access platform. Access VM terminals via web browser without SSH.

## How It Works

```
┌─────────────┐         ┌─────────────┐         ┌─────────────┐
│   Browser   │◄──WS───►│   Server    │◄──WS───►│    Agent    │
│  (xterm.js) │         │             │         │  (on VM)    │
└─────────────┘         └─────────────┘         └─────────────┘
```

1. **Agent** runs on your VMs, connects outbound to server (works behind NAT)
2. **Server** authenticates agents, proxies terminal sessions
3. **Browser** shows connected VMs, click to open terminal

## Features

- No SSH required on VMs
- Works behind NAT/firewalls (agent connects outbound)
- 2-way authentication (token + Ed25519 signature)
- Full terminal support (vim, htop, colors, resize)
- Web-based UI with xterm.js

## Install

Download pre-built binaries from [Releases](https://github.com/targc/holdthedoor/releases) or build from source.

## Quick Start

### 1. Generate Keys

```bash
mkdir -p keys
openssl genpkey -algorithm ed25519 -out keys/server.key
openssl pkey -in keys/server.key -pubout -out keys/server.pub
```

### 2. Build

```bash
go build -o bin/server ./server
go build -o bin/agent ./agent
```

### 3. Run Server

```bash
./bin/server \
  --port 8080 \
  --server-key keys/server.key \
  --token YOUR_SECRET_TOKEN
```

### 4. Install Agent (on each VM)

```bash
curl -sL https://raw.githubusercontent.com/targc/holdthedoor/main/install-agent.sh | sudo bash -s -- \
  --server wss://YOUR_SERVER/ws/agent \
  --token YOUR_SECRET_TOKEN
```

> Use `wss://` for HTTPS servers, `ws://` for HTTP.

To uninstall:

```bash
curl -sL https://raw.githubusercontent.com/targc/holdthedoor/main/uninstall-agent.sh | sudo bash
```

This downloads the correct binary, installs it, and sets up a systemd service (Linux).

Or manually:

```bash
./agent \
  --server wss://YOUR_SERVER/ws/agent \
  --server-pubkey server.pub \
  --token YOUR_SECRET_TOKEN \
  --name "my-vm"
```

### 5. Open Browser

Navigate to `http://YOUR_SERVER:8080` - you'll see connected VMs in the sidebar.

## Docker

```bash
docker build -f Dockerfile.server -t holdthedoor-server .

docker run -p 8080:8080 \
  -v $(pwd)/keys:/app/keys:ro \
  holdthedoor-server \
  --server-key /app/keys/server.key \
  --token YOUR_SECRET_TOKEN
```

## Security

| Direction | Method |
|-----------|--------|
| Agent → Server | Static token authentication |
| Server → Agent | Ed25519 signature verification |

The agent sends a random challenge; server signs it with private key. Agent verifies signature using server's public key. This prevents MITM attacks even if an attacker intercepts the token.

## CLI Reference

### Server

| Flag | Required | Description |
|------|----------|-------------|
| `--port` | No | Server port (default: 8080) |
| `--server-key` | Yes | Path to Ed25519 private key |
| `--token` | Yes | Agent authentication token |

### Agent

| Flag | Required | Description |
|------|----------|-------------|
| `--server` | No | Server WebSocket URL (default: ws://localhost:8080/ws/agent) |
| `--server-pubkey` | Yes | Path to server's Ed25519 public key |
| `--token` | Yes | Authentication token |
| `--name` | No | VM display name (default: hostname) |

## License

MIT
