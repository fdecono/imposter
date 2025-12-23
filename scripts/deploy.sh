#!/bin/bash
# ===========================================
# Imposter Game - Deployment Script
# ===========================================
# 
# Usage:
#   ./scripts/deploy.sh [host]
#
# Prerequisites:
#   - SSH access to the target host
#   - The 'imposter' user and /opt/imposter directory exist on the host
#   - systemd service is set up
#
# First-time server setup:
#   sudo useradd -r -s /bin/false imposter
#   sudo mkdir -p /opt/imposter/{bin,web}
#   sudo chown -R imposter:imposter /opt/imposter
#   sudo cp deployments/systemd/imposter.service /etc/systemd/system/
#   sudo systemctl daemon-reload
#   sudo systemctl enable imposter

set -e

# Configuration
DEPLOY_HOST="${1:-your-lightsail-ip}"
DEPLOY_PATH="/opt/imposter"
DEPLOY_USER="ubuntu"  # SSH user (imposter runs as different user)

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}═══════════════════════════════════════${NC}"
echo -e "${YELLOW}  Imposter Game - Deployment${NC}"
echo -e "${YELLOW}═══════════════════════════════════════${NC}"
echo ""
echo -e "Target: ${GREEN}${DEPLOY_HOST}${NC}"
echo ""

# Step 1: Build
echo -e "${YELLOW}[1/4]${NC} Building for Linux..."
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/server-linux ./cmd/server
echo -e "${GREEN}✓${NC} Build complete"

# Step 2: Upload binary
echo -e "${YELLOW}[2/4]${NC} Uploading binary..."
rsync -avz --progress \
    bin/server-linux \
    "${DEPLOY_USER}@${DEPLOY_HOST}:/tmp/imposter-server"
echo -e "${GREEN}✓${NC} Binary uploaded"

# Step 3: Upload web assets
echo -e "${YELLOW}[3/4]${NC} Uploading web assets..."
rsync -avz --delete --progress \
    web/ \
    "${DEPLOY_USER}@${DEPLOY_HOST}:${DEPLOY_PATH}/web/"
echo -e "${GREEN}✓${NC} Web assets uploaded"

# Step 4: Install and restart
echo -e "${YELLOW}[4/4]${NC} Installing and restarting service..."
ssh "${DEPLOY_USER}@${DEPLOY_HOST}" << EOF
    set -e
    sudo mv /tmp/imposter-server ${DEPLOY_PATH}/bin/server
    sudo chown imposter:imposter ${DEPLOY_PATH}/bin/server
    sudo chmod +x ${DEPLOY_PATH}/bin/server
    sudo systemctl restart imposter
    sleep 2
    sudo systemctl status imposter --no-pager | head -20
EOF

echo ""
echo -e "${GREEN}═══════════════════════════════════════${NC}"
echo -e "${GREEN}  ✓ Deployment Complete!${NC}"
echo -e "${GREEN}═══════════════════════════════════════${NC}"
echo ""
echo -e "Game is now live at: ${GREEN}https://imposter.yourdomain.com${NC}"

