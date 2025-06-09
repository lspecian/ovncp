#!/bin/bash
# Quick start script for OVNCP - single binary deployment

set -e

echo "OVNCP Quick Start"
echo "================="
echo

# Check if running as root
if [ "$EUID" -eq 0 ]; then 
   echo "Warning: Running as root is not recommended."
   echo "Consider creating a dedicated user for OVNCP."
   echo
fi

# Create data directory
DATA_DIR="${DATA_DIR:-./data}"
mkdir -p "$DATA_DIR"
echo "Data directory: $DATA_DIR"

# Set default environment variables
export DB_TYPE="${DB_TYPE:-sqlite}"
export DB_NAME="${DB_NAME:-$DATA_DIR/ovncp.db}"
export AUTH_ENABLED="${AUTH_ENABLED:-true}"
export JWT_SECRET="${JWT_SECRET:-$(openssl rand -base64 32 2>/dev/null || echo 'change-me-in-production-min-32-chars')}"
export LOG_LEVEL="${LOG_LEVEL:-info}"

# Check for OVN configuration
if [ -z "$OVN_NORTHBOUND_DB" ]; then
    echo
    echo "Warning: OVN_NORTHBOUND_DB not set."
    echo "Set it to your OVN northbound database connection string."
    echo "Example: export OVN_NORTHBOUND_DB='tcp:192.168.1.10:6641'"
    echo "Running in demo mode without OVN connection."
    echo
fi

# Display configuration
echo
echo "Configuration:"
echo "  Database: SQLite ($DB_NAME)"
echo "  Auth: Enabled (default: admin/admin)"
echo "  API Port: ${API_PORT:-8080}"
echo "  OVN NB: ${OVN_NORTHBOUND_DB:-Not configured}"
echo

# Check if binary exists
if [ ! -f "./ovncp" ]; then
    echo "Error: ovncp binary not found in current directory."
    echo "Build it with: go build -o ovncp ./cmd/api"
    exit 1
fi

# Start OVNCP
echo "Starting OVNCP..."
echo "Access the web UI at: http://localhost:${API_PORT:-8080}"
echo "Press Ctrl+C to stop"
echo

exec ./ovncp