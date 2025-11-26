#!/bin/bash
set -e

echo "=== Setting up MySQL for E2E tests ==="

# MySQL should already be running (started by entrypoint.sh)
# Wait a bit more to ensure it's fully ready
sleep 2

# Configure root user for password-less access (for testing)
# In MySQL on Ubuntu 22.04, root user uses unix_socket auth by default
# We need to set it to use native password with empty password for tests
echo "Configuring root user..."
mysql -u root <<'MYSQL_EOF' || true
-- Switch root from unix_socket to native password with empty password
ALTER USER 'root'@'localhost' IDENTIFIED WITH mysql_native_password BY '';
FLUSH PRIVILEGES;
MYSQL_EOF

# Create backup user with proper permissions
echo "Creating backup user..."
mysql -u root <<'MYSQL_EOF'
CREATE USER IF NOT EXISTS 'backupuser'@'localhost' IDENTIFIED BY 's0m3p455w0rd';
GRANT BACKUP_ADMIN, RELOAD, PROCESS, REPLICATION CLIENT, LOCK TABLES ON *.* TO 'backupuser'@'localhost';
GRANT SELECT, INSERT, UPDATE, DELETE, CREATE, DROP, INDEX, ALTER ON *.* TO 'backupuser'@'localhost';
CREATE DATABASE IF NOT EXISTS testdb;
FLUSH PRIVILEGES;
MYSQL_EOF

echo "Loading test schema..."
mysql -u root testdb < /tests/fixtures/test_schema.sql

echo "MySQL setup complete!"
echo "Test database: testdb"
echo "Backup user: backupuser"
