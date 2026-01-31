#!/bin/bash
set -e

SERVICE_NAME="holdthedoor-agent"
INSTALL_DIR="/usr/local/bin"

echo "Uninstalling HoldTheDoor agent..."

# Stop service
if command -v systemctl &>/dev/null && systemctl is-active --quiet ${SERVICE_NAME} 2>/dev/null; then
    echo "Stopping systemd service..."
    sudo systemctl stop ${SERVICE_NAME}
    sudo systemctl disable ${SERVICE_NAME}
    sudo rm -f /etc/systemd/system/${SERVICE_NAME}.service
    sudo systemctl daemon-reload
else
    # Kill process on macOS/other
    pkill -f "${INSTALL_DIR}/agent" 2>/dev/null && echo "Stopped agent process" || true
fi

# Remove binary
if [[ -f "${INSTALL_DIR}/agent" ]]; then
    sudo rm -f "${INSTALL_DIR}/agent"
    echo "Removed ${INSTALL_DIR}/agent"
fi

# Remove config
if [[ -d "/etc/${SERVICE_NAME}" ]]; then
    sudo rm -rf "/etc/${SERVICE_NAME}"
    echo "Removed /etc/${SERVICE_NAME}"
fi

# Remove log
rm -f /tmp/${SERVICE_NAME}.log 2>/dev/null || true

echo "Done!"
