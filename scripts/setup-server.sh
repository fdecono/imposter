#!/bin/bash
# ===========================================
# Imposter Game - First-Time Server Setup
# ===========================================
#
# Run this on your AWS Lightsail instance to set up the environment.
# 
# Usage:
#   ssh your-server
#   curl -sSL https://raw.githubusercontent.com/youruser/imposter/main/scripts/setup-server.sh | sudo bash
#
# Or copy and run manually:
#   sudo bash setup-server.sh

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${YELLOW}═══════════════════════════════════════${NC}"
echo -e "${YELLOW}  Imposter Game - Server Setup${NC}"
echo -e "${YELLOW}═══════════════════════════════════════${NC}"
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "Please run as root (sudo)"
    exit 1
fi

# Step 1: Create system user
echo -e "${YELLOW}[1/5]${NC} Creating system user..."
if id "imposter" &>/dev/null; then
    echo "User 'imposter' already exists, skipping..."
else
    useradd -r -s /bin/false imposter
    echo -e "${GREEN}✓${NC} User created"
fi

# Step 2: Create directories
echo -e "${YELLOW}[2/5]${NC} Creating directories..."
mkdir -p /opt/imposter/{bin,web/static}
chown -R imposter:imposter /opt/imposter
echo -e "${GREEN}✓${NC} Directories created"

# Step 3: Create environment file template
echo -e "${YELLOW}[3/5]${NC} Creating environment file..."
if [ ! -f /opt/imposter/.env ]; then
    cat > /opt/imposter/.env << 'EOF'
# Imposter Game Configuration
PORT=8080
HOST=0.0.0.0
ENV=production

# Game Settings
MIN_PLAYERS=4
MAX_PLAYERS=10
VOTING_DURATION_SECONDS=20

# Logging
LOG_LEVEL=info
LOG_FORMAT=json
EOF
    chown imposter:imposter /opt/imposter/.env
    chmod 600 /opt/imposter/.env
    echo -e "${GREEN}✓${NC} Environment file created"
else
    echo "Environment file already exists, skipping..."
fi

# Step 4: Install systemd service
echo -e "${YELLOW}[4/5]${NC} Installing systemd service..."
cat > /etc/systemd/system/imposter.service << 'EOF'
[Unit]
Description=Imposter Game Server
After=network.target

[Service]
Type=simple
User=imposter
Group=imposter
WorkingDirectory=/opt/imposter
ExecStart=/opt/imposter/bin/server
Restart=on-failure
RestartSec=5
EnvironmentFile=/opt/imposter/.env
StandardOutput=journal
StandardError=journal
SyslogIdentifier=imposter
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/imposter

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable imposter
echo -e "${GREEN}✓${NC} systemd service installed"

# Step 5: Show next steps
echo -e "${YELLOW}[5/5]${NC} Setup complete!"
echo ""
echo -e "${GREEN}═══════════════════════════════════════${NC}"
echo -e "${GREEN}  ✓ Server Setup Complete!${NC}"
echo -e "${GREEN}═══════════════════════════════════════${NC}"
echo ""
echo "Next steps:"
echo "  1. Deploy the application: ./scripts/deploy.sh YOUR_SERVER_IP"
echo "  2. Set up Nginx or Caddy for HTTPS (see deployments/ folder)"
echo "  3. Configure DNS to point to this server"
echo "  4. Obtain SSL certificate: sudo certbot --nginx -d imposter.yourdomain.com"
echo ""
echo "Useful commands:"
echo "  sudo systemctl status imposter    # Check status"
echo "  sudo journalctl -u imposter -f    # View logs"
echo "  sudo systemctl restart imposter   # Restart service"

