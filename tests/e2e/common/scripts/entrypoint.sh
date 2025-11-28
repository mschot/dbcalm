#!/bin/bash
# Entrypoint script that starts services and runs E2E tests
set -e

echo "=== E2E Test Entrypoint ==="

# Determine database type (default to mariadb for backward compatibility)
DB_TYPE=${DB_TYPE:-mariadb}
echo "Database type: $DB_TYPE"
export DB_TYPE

# Ensure all scripts are executable (needed for volume-mounted scripts)
chmod +x /tests/scripts/*.sh 2>/dev/null || true

# Initialize database if needed (Rocky/RHEL requires explicit initialization)
if [ ! -d "/var/lib/mysql/mysql" ]; then
    echo "Initializing $DB_TYPE data directory..."

    # Clean up any partial initialization files
    # (mysqld --initialize-insecure fails if directory is not empty)
    if [ "$(ls -A /var/lib/mysql 2>/dev/null)" ]; then
        echo "Cleaning up partial initialization files..."
        rm -rf /var/lib/mysql/*
    fi

    # Ensure data directory exists and has correct ownership
    mkdir -p /var/lib/mysql
    chown -R mysql:mysql /var/lib/mysql
    chmod 750 /var/lib/mysql

    # Initialize with mysqld (works for both MariaDB and MySQL)
    # Use --user=mysql flag to run as mysql user (required by modern MariaDB/MySQL)

    if [ "$DB_TYPE" = "mariadb" ]; then
        echo "Running mariadb --initialize-insecure --user=mysql..."
        mariadb-install-db --user=mysql --datadir=/var/lib/mysql
        echo "Initialization complete"
    else
        echo "Running /usr/libexec/mysqld --initialize-insecure --user=mysql..."
        /usr/libexec/mysqld --initialize-insecure --user=mysql --datadir=/var/lib/mysql
        echo "Initialization complete"
    fi

fi

# Create log file with correct ownership for database error logs
echo "Setting up database log file..."
touch /var/log/db-error.log
chown mysql:mysql /var/log/db-error.log
chmod 664 /var/log/db-error.log

# Start database in the background
if [ "$DB_TYPE" = "mariadb" ]; then
    echo "Starting MariaDB..."
    mysqld --user=mysql --datadir=/var/lib/mysql --log-error=/var/log/db-error.log &
    DB_PID=$!
else
    echo "Starting MySQL..."
    mysqld --user=mysql --datadir=/var/lib/mysql --log-error=/var/log/db-error.log &
    DB_PID=$!
fi

# Wait for database to be ready
echo "Waiting for $DB_TYPE to be ready..."
for i in {1..30}; do
    if mysqladmin ping -h localhost --silent 2>/dev/null; then
        echo "$DB_TYPE is ready!"
        break
    fi
    echo "Waiting for $DB_TYPE... ($i/30)"
    sleep 2
done

if ! mysqladmin ping -h localhost --silent 2>/dev/null; then
    echo "ERROR: $DB_TYPE failed to start"
    exit 1
fi

# Setup database (create user, database, load schema)
if [ "$DB_TYPE" = "mariadb" ]; then
    echo "Setting up MariaDB..."
    /tests/scripts/setup_mariadb.sh
else
    echo "Setting up MySQL..."
    /tests/scripts/setup_mysql.sh
fi

# Install and setup DBCalm
echo "Setting up DBCalm..."
/tests/scripts/setup_dbcalm.sh

# Source credentials and generate API token
if [ -f /tmp/e2e_credentials.env ]; then
    source /tmp/e2e_credentials.env
    export E2E_CLIENT_ID E2E_CLIENT_SECRET
    echo "Loaded API client credentials"

    # Generate API token for tests
    echo "Generating API token..."
    TOKEN_RESPONSE=$(curl -k -s -X POST https://localhost:8335/auth/token \
        -H "Content-Type: application/json" \
        -d '{"grant_type":"client_credentials","client_id":"'"$E2E_CLIENT_ID"'","client_secret":"'"$E2E_CLIENT_SECRET"'"}')

    API_TOKEN=$(echo "$TOKEN_RESPONSE" | grep -o '"access_token":"[^"]*' | cut -d'"' -f4)

    if [ -z "$API_TOKEN" ]; then
        echo "ERROR: Failed to get API token"
        echo "Response: $TOKEN_RESPONSE"
        exit 1
    fi

    export API_TOKEN
    export API_BASE_URL="https://localhost:8335"
    echo "API token generated successfully"
else
    echo "ERROR: Credentials file not found at /tmp/e2e_credentials.env"
    exit 1
fi

# Download Go dependencies
cd /tests
go mod download

# Change to test directory and run tests
cd /tests/tests
echo "=== Running Go tests ==="
go test -v -timeout 30m . 2>&1 | tee /tests/test-results/test-output.log

TEST_EXIT_CODE=${PIPESTATUS[0]}

# Cleanup
echo "=== Cleanup ==="
/tests/scripts/cleanup.sh || true

# Stop database
echo "Stopping $DB_TYPE..."
mysqladmin -u root shutdown 2>/dev/null || kill $DB_PID 2>/dev/null || true

echo "=== Tests completed with exit code: $TEST_EXIT_CODE ==="
exit $TEST_EXIT_CODE
