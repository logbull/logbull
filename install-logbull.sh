#!/bin/bash

# Check if script is run as root
if [ "$(id -u)" -ne 0 ]; then
    echo "Error: This script must be run as root (sudo ./install-logbull.sh)" >&2
    exit 1
fi

# Set up logging
LOG_FILE="/var/log/logbull-install.log"
INSTALL_DIR="/opt/logbull"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

# Create log file if doesn't exist
touch "$LOG_FILE"
log "Starting Log Bull installation..."

# Create installation directory
log "Creating installation directory..."
if [ ! -d "$INSTALL_DIR" ]; then
    mkdir -p "$INSTALL_DIR"
    log "Created directory: $INSTALL_DIR"
else
    log "Directory already exists: $INSTALL_DIR"
fi

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    log "Docker not found. Installing Docker..."
    
    # Install Docker
    apt-get update
    apt-get remove -y docker docker-engine docker.io containerd runc
    apt-get install -y ca-certificates curl gnupg lsb-release
    mkdir -p /etc/apt/keyrings
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
    apt-get update
    apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
    log "Docker installed successfully"
else
    log "Docker already installed"
fi

# Check if docker-compose is installed
if ! command -v docker-compose &> /dev/null && ! command -v docker compose &> /dev/null; then
    log "Docker Compose not found. Installing Docker Compose..."
    apt-get update
    apt-get install -y docker-compose-plugin
    log "Docker Compose installed successfully"
else
    log "Docker Compose already installed"
fi

# Write docker-compose.yml
log "Writing docker-compose.yml to $INSTALL_DIR"
cat > "$INSTALL_DIR/docker-compose.yml" << 'EOF'
version: "3"

services:
  logbull:
    container_name: logbull
    image: logbull/logbull:latest
    ports:
      - "4005:4005"
    volumes:
      - ./logbull-data:/logbull-data
    restart: unless-stopped
EOF
log "docker-compose.yml created successfully"

# Start LogBull
log "Starting LogBull..."
cd "$INSTALL_DIR"
docker compose up -d
log "Log Bull started successfully"

log "Log Bull installation completed successfully!"
log "-------------------------------------------"
log "To launch:"
log "> cd $INSTALL_DIR && docker compose up -d"
log "Access Log Bull at: http://localhost:4005"