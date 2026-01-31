#!/bin/bash
set -e

SERVER_PUBKEY='-----BEGIN PUBLIC KEY-----
MCowBQYDK2VwAyEAxCneWbYo/xvASGVH1GNdg1RzxTnuXSFc9i5bfyAvEK8=
-----END PUBLIC KEY-----'

REPO="targc/holdthedoor"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="holdthedoor-agent"
SERVICE_NAME="holdthedoor-agent"

usage() {
    echo "Usage: $0 --server <ws://host:port/ws/agent> --token <token>"
    exit 1
}

while [[ $# -gt 0 ]]; do
    case $1 in
        --server) SERVER_URL="$2"; shift 2 ;;
        --token) TOKEN="$2"; shift 2 ;;
        *) usage ;;
    esac
done

[[ -z "$SERVER_URL" || -z "$TOKEN" ]] && usage

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $ARCH in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

case $OS in
    linux|darwin) ;;
    *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

echo "Detected: ${OS}/${ARCH}"

LATEST=$(curl -sI "https://github.com/${REPO}/releases/latest" | grep -i "^location:" | sed 's/.*tag\///' | tr -d '\r\n')
[[ -z "$LATEST" ]] && { echo "Failed to get latest release"; exit 1; }
echo "Latest release: $LATEST"

BINARY_URL="https://github.com/${REPO}/releases/download/${LATEST}/agent-${OS}-${ARCH}"
TMP_BIN=$(mktemp)
echo "Downloading agent..."
curl -sL "$BINARY_URL" -o "$TMP_BIN"
chmod +x "$TMP_BIN"

echo "Installing to ${INSTALL_DIR}/${BINARY_NAME}..."
sudo mv "$TMP_BIN" "${INSTALL_DIR}/${BINARY_NAME}"

PUBKEY_PATH="/etc/${SERVICE_NAME}/server.pub"
sudo mkdir -p "$(dirname "$PUBKEY_PATH")"
echo "$SERVER_PUBKEY" | sudo tee "$PUBKEY_PATH" > /dev/null

if [[ "$OS" == "linux" ]] && command -v systemctl &>/dev/null; then
    echo "Setting up systemd service..."
    sudo tee /etc/systemd/system/${SERVICE_NAME}.service > /dev/null <<EOF
[Unit]
Description=HoldTheDoor Agent
After=network.target

[Service]
Type=simple
ExecStart=${INSTALL_DIR}/${BINARY_NAME} --server ${SERVER_URL} --server-pubkey ${PUBKEY_PATH} --token ${TOKEN}
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

    sudo systemctl daemon-reload
    sudo systemctl enable ${SERVICE_NAME}
    sudo systemctl restart ${SERVICE_NAME}
    echo "Service started. Check status: systemctl status ${SERVICE_NAME}"
elif [[ "$OS" == "darwin" ]]; then
    echo "Starting agent..."
    nohup ${INSTALL_DIR}/${BINARY_NAME} --server ${SERVER_URL} --server-pubkey ${PUBKEY_PATH} --token ${TOKEN} > /tmp/${SERVICE_NAME}.log 2>&1 &
    echo "Agent started (PID: $!). Log: /tmp/${SERVICE_NAME}.log"
else
    echo "Run agent manually:"
    echo "  ${BINARY_NAME} --server ${SERVER_URL} --server-pubkey ${PUBKEY_PATH} --token ${TOKEN}"
fi

echo "Done!"
