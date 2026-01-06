#!/bin/bash

# ============================================
# IMPOSTER GAME - Deployment Script
# ============================================

set -e  # Exit on any error

# ============================================
# LOAD ENVIRONMENT VARIABLES FROM .env.deploy
# ============================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="${SCRIPT_DIR}/.env"

if [ -f "${ENV_FILE}" ]; then
    echo "Loading configuration from .env..."
    # Export variables from .env file (ignore comments and empty lines)
    set -a
    source "${ENV_FILE}"
    set +a
else
    echo "Warning: .env.deploy file not found at ${ENV_FILE}"
    echo "Using default values."
fi

# ============================================
# CONFIGURATION (with fallback defaults)
# ============================================

APP_NAME="${APP_NAME:-imposter}"
SSH_KEY="${SSH_KEY:-~/.ssh/lightsail}"
SSH_USER="${SSH_USER:-ubuntu}"
SSH_HOST="${SSH_HOST:-YOUR_SERVER_IP}"
REMOTE_DIR="${REMOTE_DIR:-imposter}"
LOCAL_DIR="${LOCAL_DIR:-$(pwd)}"
HOST_PORT="${HOST_PORT:-8082}"
CONTAINER_PORT="${CONTAINER_PORT:-8080}"
DOMAIN="${DOMAIN:-imposter.fdecono.com}"

# Expand tilde in SSH_KEY path
SSH_KEY="${SSH_KEY/#\~/$HOME}"

# ============================================
# COLORS FOR OUTPUT
# ============================================

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# ============================================
# HELPER FUNCTIONS
# ============================================

print_step() {
    echo -e "\n${BLUE}==>${NC} ${GREEN}$1${NC}"
}

print_info() {
    echo -e "${YELLOW}    $1${NC}"
}

print_error() {
    echo -e "${RED}ERROR: $1${NC}"
}

# ============================================
# DEPLOYMENT STEPS
# ============================================

echo -e "${CYAN}"
echo "============================================"
echo "  🎭 DEPLOYING IMPOSTER GAME"
echo "============================================"
echo -e "${NC}"

print_info "SSH Key: ${SSH_KEY}"
print_info "Server: ${SSH_USER}@${SSH_HOST}"
print_info "Remote Dir: ~/${REMOTE_DIR}"
print_info "Port Mapping: ${HOST_PORT}:${CONTAINER_PORT}"

# Step 1: Build Docker image
print_step "Step 1/8: Building Docker image..."
docker build --platform linux/amd64 -t "${APP_NAME}" .

# Step 2: Save and compress the image
print_step "Step 2/8: Saving and compressing image..."
docker save "${APP_NAME}" | gzip > "${APP_NAME}.tar.gz"
print_info "Created ${APP_NAME}.tar.gz ($(du -h ${APP_NAME}.tar.gz | cut -f1))"

# Step 3: Copy to server
print_step "Step 3/8: Copying image to server..."
scp -i "${SSH_KEY}" "${APP_NAME}.tar.gz" "${SSH_USER}@${SSH_HOST}:~/"

# Step 4-8: Run remote commands via SSH
print_step "Step 4/8: Connecting to server and deploying..."

ssh -i "${SSH_KEY}" "${SSH_USER}@${SSH_HOST}" << REMOTE_COMMANDS
    set -e
    
    echo "  → Creating app directory if needed..."
    mkdir -p ~/${REMOTE_DIR}
    
    echo "  → Backing up old image..."
    cd ~/${REMOTE_DIR}
    if [ -f "${APP_NAME}.tar.gz" ]; then
        mv "${APP_NAME}.tar.gz" "${APP_NAME}_old.tar.gz"
        echo "    Renamed old image to ${APP_NAME}_old.tar.gz"
    fi
    
    echo "  → Moving new image to app directory..."
    cd ~
    mv "${APP_NAME}.tar.gz" "${REMOTE_DIR}/"
    
    echo "  → Loading new Docker image..."
    cd ~/${REMOTE_DIR}
    docker load < "${APP_NAME}.tar.gz"
    
    echo "  → Stopping old container..."
    docker stop "${APP_NAME}" 2>/dev/null || true
    
    echo "  → Removing old container..."
    docker rm "${APP_NAME}" 2>/dev/null || true
    
    echo "  → Starting new container..."
    docker run -d \
        --name "${APP_NAME}" \
        -p ${HOST_PORT}:${CONTAINER_PORT} \
        --restart unless-stopped \
        "${APP_NAME}"
    
    echo ""
    echo "============================================"
    echo "  CONTAINER STATUS"
    echo "============================================"
    docker ps --filter "name=${APP_NAME}"
    
    echo ""
    echo "============================================"
    echo "  CONTAINER LOGS"
    echo "============================================"
    docker logs "${APP_NAME}" 2>&1 | tail -20
REMOTE_COMMANDS

# Step 9: Clean up local tar.gz
print_step "Step 8/8: Cleaning up local files..."
rm -f "${APP_NAME}.tar.gz"
print_info "Removed local ${APP_NAME}.tar.gz"

# Done!
echo -e "\n${CYAN}"
echo "============================================"
echo "  🎭 ✅ DEPLOYMENT COMPLETE!"
echo "============================================"
echo -e "${NC}"
echo -e "Visit: ${BLUE}https://${DOMAIN}${NC}"
echo ""

