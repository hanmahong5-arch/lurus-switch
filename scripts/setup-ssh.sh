#!/bin/bash
# Setup SSH key authentication for remote server
# Usage: ./setup-ssh.sh

SERVER_IP="115.190.239.146"
SERVER_USER="root"
PASSWORD="GGsuperman1211"

echo "=== Setting up SSH key authentication ==="

# Install sshpass if needed (optional)
if ! command -v sshpass &> /dev/null; then
    echo "Note: sshpass not found. You may need to enter password manually."
    echo "To install sshpass on Cygwin: apt-cyg install sshpass"
fi

# Copy SSH public key to server
if [ -f ~/.ssh/id_rsa.pub ]; then
    echo "Copying SSH public key to $SERVER_IP..."

    # Method 1: Using sshpass (if available)
    if command -v sshpass &> /dev/null; then
        sshpass -p "$PASSWORD" ssh-copy-id -o StrictHostKeyChecking=no ${SERVER_USER}@${SERVER_IP}
    else
        # Method 2: Manual copy (requires password input)
        echo "Please enter password when prompted: $PASSWORD"
        ssh-copy-id -o StrictHostKeyChecking=no ${SERVER_USER}@${SERVER_IP}
    fi

    echo "SSH key setup complete!"
else
    echo "Error: No SSH key found at ~/.ssh/id_rsa.pub"
    echo "Generating new SSH key..."
    ssh-keygen -t rsa -b 4096 -f ~/.ssh/id_rsa -N ""
    echo "SSH key generated. Please run this script again."
    exit 1
fi

# Test connection
echo ""
echo "Testing SSH connection..."
if ssh -o BatchMode=yes -o ConnectTimeout=5 ${SERVER_USER}@${SERVER_IP} "echo 'Connection successful!'" 2>/dev/null; then
    echo "✓ SSH key authentication working!"
else
    echo "✗ SSH key authentication failed. Please check the setup."
    exit 1
fi
