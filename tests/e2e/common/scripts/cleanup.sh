#!/bin/bash
set -e

echo "=== Cleaning up E2E test environment ==="

# Clear only test database (preserve MariaDB system files)
echo "Clearing test database..."
if mysqladmin ping -h localhost --silent 2>/dev/null; then
    mysql -u root <<'MYSQL_EOF' 2>/dev/null || true
DROP DATABASE IF EXISTS testdb;
DELETE FROM mysql.user WHERE User = 'backupuser';
FLUSH PRIVILEGES;
MYSQL_EOF
fi

# Clear backup directory
echo "Clearing backup directory..."
if [ -d /var/backups/dbcalm ]; then
    rm -rf /var/backups/dbcalm/*
fi

# Clear DBCalm database
echo "Clearing DBCalm database..."
if [ -f /var/lib/dbcalm/db.sqlite3 ]; then
    rm -f /var/lib/dbcalm/db.sqlite3
fi

echo "Cleanup complete!"
