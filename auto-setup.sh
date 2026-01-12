#!/bin/bash
# Automated deployment script with password handling
# This script will guide you through the process

set -e

SERVER="115.190.239.146"
USER="root"
PASS="GGsuperman1211"
REMOTE_DIR="/opt/lurus"

echo "╔════════════════════════════════════════════════════════════╗"
echo "║       Lurus Switch Automated Deployment                   ║"
echo "║       Server: $SERVER                              ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo ""

# Check if we're in Git Bash
if [[ "$OSTYPE" != "msys" && "$OSTYPE" != "cygwin" ]]; then
    echo "Warning: This script should be run in Git Bash"
fi

# Step 1: Setup SSH key
echo "Step 1/10: Setting up SSH key authentication..."
echo ""
echo "IMPORTANT: When prompted for password, enter: $PASS"
echo ""
echo "Press Enter to continue..."
read

# Copy SSH key
cat ~/.ssh/id_rsa.pub | ssh -o StrictHostKeyChecking=no $USER@$SERVER "mkdir -p ~/.ssh && chmod 700 ~/.ssh && cat >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys"

# Verify SSH key setup
echo ""
echo "Verifying SSH connection..."
if ssh -o BatchMode=yes -o ConnectTimeout=5 $USER@$SERVER "echo OK" 2>/dev/null | grep -q "OK"; then
    echo "✓ SSH key configured successfully!"
else
    echo "✗ SSH key configuration failed. Please try manually:"
    echo "cat ~/.ssh/id_rsa.pub | ssh $USER@$SERVER \"mkdir -p ~/.ssh && cat >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys\""
    exit 1
fi

# Now run the deployment
echo ""
echo "Step 2/10: Starting deployment..."
bash deploy-now.sh

echo ""
echo "╔════════════════════════════════════════════════════════════╗"
echo "║              Deployment Process Complete!                 ║"
echo "╚════════════════════════════════════════════════════════════╝"
