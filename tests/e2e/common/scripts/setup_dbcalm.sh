#!/bin/bash
set -e

echo "=== Setting up DBCalm for E2E tests ==="

# Determine distro (default to debian for backward compatibility)
DISTRO=${DISTRO:-debian}
echo "Distribution: $DISTRO"

# For Go version, copy binaries from mounted artifacts directory
echo "Installing DBCalm binaries from /tests/artifacts/..."
if [ ! -f "/tests/artifacts/dbcalm" ] || [ ! -f "/tests/artifacts/dbcalm-db-cmd" ] || [ ! -f "/tests/artifacts/dbcalm-cmd" ]; then
    echo "ERROR: Required binaries not found in /tests/artifacts/"
    ls -la /tests/artifacts/
    exit 1
fi

# Copy binaries to /usr/bin
cp /tests/artifacts/dbcalm /usr/bin/dbcalm
cp /tests/artifacts/dbcalm-db-cmd /usr/bin/dbcalm-db-cmd
cp /tests/artifacts/dbcalm-cmd /usr/bin/dbcalm-cmd

# Make them executable
chmod +x /usr/bin/dbcalm
chmod +x /usr/bin/dbcalm-db-cmd
chmod +x /usr/bin/dbcalm-cmd

echo "Binaries installed successfully"

# Create necessary directories and users
echo "Creating DBCalm directories and users..."
mkdir -p /etc/dbcalm
mkdir -p /var/lib/dbcalm
mkdir -p /var/log/dbcalm
mkdir -p /var/run/dbcalm
mkdir -p /var/backups/dbcalm

# Create dbcalm user and group if they don't exist
if ! id -u dbcalm >/dev/null 2>&1; then
    useradd -r -s /bin/false dbcalm
fi

# Set up permissions
chown -R mysql:mysql /var/run/dbcalm
chown -R dbcalm:dbcalm /var/lib/dbcalm
chown -R dbcalm:dbcalm /var/log/dbcalm
chmod 755 /var/lib/dbcalm
chmod 755 /var/log/dbcalm

# Configure db_type based on DB_TYPE environment variable
DB_TYPE=${DB_TYPE:-mariadb}
echo "Configuring DBCalm for database type: $DB_TYPE..."

# Create basic config file
cat > /etc/dbcalm/config.yml <<EOF
db_type: $DB_TYPE
backup_dir: /var/backups/dbcalm
db_socket: /var/run/dbcalm/dbcalm-db.sock
cmd_socket: /var/run/dbcalm/dbcalm-cmd.sock
database_path: /var/lib/dbcalm/db.sqlite3
log_file: /var/log/dbcalm/dbcalm.log
tls:
  enabled: true
  cert_file: /etc/dbcalm/cert.pem
  key_file: /etc/dbcalm/key.pem
server:
  host: 0.0.0.0
  port: 8335
EOF

# Generate self-signed certificate for HTTPS
echo "Generating self-signed TLS certificate..."
openssl req -x509 -newkey rsa:4096 -nodes -keyout /etc/dbcalm/key.pem -out /etc/dbcalm/cert.pem -days 365 -subj "/CN=localhost" 2>/dev/null
chmod 600 /etc/dbcalm/key.pem
chown dbcalm:dbcalm /etc/dbcalm/key.pem /etc/dbcalm/cert.pem

# Create credentials file for xtrabackup
cat > /etc/dbcalm/credentials.cnf <<EOF
[client-dbcalm]
user=root
password=
EOF
chmod 600 /etc/dbcalm/credentials.cnf

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

# Save credentials to file for pytest
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
ps aux | grep -E "(dbcalm|dbcalm-cmd|dbcalm-mariadb-cmd)" | grep -v grep || echo "Warning: Some services may not be running"

# Verify command sockets
echo ""
# Ensure verify_sockets.sh is executable
chmod +x /tests/scripts/verify_sockets.sh 2>/dev/null || true
/tests/scripts/verify_sockets.sh
