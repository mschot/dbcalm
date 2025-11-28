#!/bin/bash
set -e

echo "=== Setting up DBCalm for E2E tests ==="

# Determine distro (default to debian for backward compatibility)
DISTRO=${DISTRO:-debian}
echo "Distribution: $DISTRO"

# Find and install the package based on distro
if [ "$DISTRO" = "debian" ]; then
    PACKAGE_FILE=$(ls /tests/artifacts/*.deb 2>/dev/null | head -1)
    if [ -z "$PACKAGE_FILE" ]; then
        echo "ERROR: No .deb package found in /tests/artifacts/"
        exit 1
    fi
    echo "Installing DBCalm from $PACKAGE_FILE..."
    dpkg -i "$PACKAGE_FILE" || apt-get install -f -y
elif [ "$DISTRO" = "rocky" ]; then
    PACKAGE_FILE=$(ls /tests/artifacts/*.rpm 2>/dev/null | head -1)
    if [ -z "$PACKAGE_FILE" ]; then
        echo "ERROR: No .rpm package found in /tests/artifacts/"
        exit 1
    fi
    echo "Installing DBCalm from $PACKAGE_FILE..."
    dnf install -y "$PACKAGE_FILE" || yum install -y "$PACKAGE_FILE"
else
    echo "ERROR: Unsupported distro: $DISTRO"
    exit 1
fi

# Re-run post-install setup to fix permissions
# (needed because volume mounts may overlay files created during package install)
echo "Running post-install setup to fix permissions..."
/usr/share/dbcalm/scripts/common-setup.sh dbcalm

# Configure db_type based on DB_TYPE environment variable
DB_TYPE=${DB_TYPE:-mariadb}
echo "Configuring DBCalm for database type: $DB_TYPE..."
sed -i "s/^db_type:.*/db_type: $DB_TYPE/" /etc/dbcalm/config.yml

# Start DBCalm services manually (no systemd in container)
echo "Starting DBCalm services..."
sudo -u root /usr/bin/dbcalm-cmd &
sleep 2
sudo -u mysql /usr/bin/dbcalm-db-cmd &
sleep 2
sudo -u dbcalm /usr/bin/dbcalm server &
sleep 5

# Wait for API server to be ready
echo "Waiting for API server to be ready..."
for i in {1..30}; do
    if curl -k -s https://localhost:8335/docs >/dev/null 2>&1; then
        echo "API server is ready!"
        break
    fi
    echo "Waiting for API server... ($i/30)"
    sleep 2
done

if ! curl -k -s https://localhost:8335/docs >/dev/null 2>&1; then
    echo "ERROR: API server failed to start"
    echo "Checking API server logs..."
    ps aux | grep dbcalm
    exit 1
fi

# Check if dbcalm command is available
if ! command -v dbcalm &> /dev/null; then
    echo "ERROR: dbcalm command not found after installation"
    exit 1
fi

# Create API client for E2E tests
echo "Creating API client for E2E tests..."
output=$(dbcalm clients add "e2e-test-client" 2>&1) || {
    echo "ERROR: dbcalm clients add failed with exit code $?"
    echo "Output: $output"
    exit 1
}

# Extract client_id and client_secret from output
client_id=$(echo "$output" | grep "Client ID:" | awk '{print $3}')
client_secret=$(echo "$output" | grep "Client Secret:" | awk '{print $3}')

if [ -z "$client_id" ] || [ -z "$client_secret" ]; then
    echo "ERROR: Failed to extract client credentials"
    echo "Output was: $output"
    exit 1
fi

# Save credentials to file for Go tests
echo "E2E_CLIENT_ID=$client_id" > /tmp/e2e_credentials.env
echo "E2E_CLIENT_SECRET=$client_secret" >> /tmp/e2e_credentials.env

# Also export for current session
export E2E_CLIENT_ID="$client_id"
export E2E_CLIENT_SECRET="$client_secret"

echo "DBCalm setup complete!"
echo "Client ID: $client_id"
echo "Credentials saved to: /tmp/e2e_credentials.env"

# Verify services are running (check processes instead of systemctl)
echo "Verifying DBCalm services..."
ps aux | grep -E "(dbcalm|dbcalm-cmd|dbcalm-db-cmd)" | grep -v grep || echo "Warning: Some services may not be running"

# Verify command sockets
echo ""
# Ensure verify_sockets.sh is executable
chmod +x /tests/scripts/verify_sockets.sh 2>/dev/null || true
if [ -f /tests/scripts/verify_sockets.sh ]; then
    /tests/scripts/verify_sockets.sh
fi
