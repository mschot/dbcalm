#!/bin/bash
# Socket verification script for DBCalm E2E tests
set -e

echo "=== Verifying DBCalm Command Sockets ==="

MARIADB_SOCKET="/var/run/dbcalm/db-cmd.sock"
CMD_SOCKET="/var/run/dbcalm/cmd.sock"
SOCKET_DIR="/var/run/dbcalm"

# Check if socket directory exists
if [ ! -d "$SOCKET_DIR" ]; then
    echo "ERROR: Socket directory does not exist: $SOCKET_DIR"
    echo "Creating directory..."
    mkdir -p "$SOCKET_DIR"
    chmod 755 "$SOCKET_DIR"
fi

# Function to check socket
check_socket() {
    local socket_path=$1
    local service_name=$2
    local max_wait=${3:-30}

    echo "Checking $service_name socket: $socket_path"

    for i in $(seq 1 $max_wait); do
        if [ -S "$socket_path" ]; then
            echo "✓ Socket exists: $socket_path"

            # Check if socket is writable
            if [ -w "$socket_path" ]; then
                echo "✓ Socket is writable"
            else
                echo "⚠ WARNING: Socket exists but is not writable"
                ls -la "$socket_path"
            fi

            return 0
        fi

        echo "  Waiting for socket... ($i/$max_wait)"
        sleep 1
    done

    echo "✗ ERROR: Socket not found after ${max_wait}s: $socket_path"
    return 1
}

# Check if services are running
echo ""
echo "Checking DBCalm services..."
ps aux | grep -E "dbcalm|mariadb-cmd" | grep -v grep || echo "No DBCalm processes found"

echo ""
echo "Socket directory contents:"
ls -la "$SOCKET_DIR" 2>/dev/null || echo "Directory is empty or doesn't exist"

echo ""
# Check MariaDB command socket
if ! check_socket "$MARIADB_SOCKET" "MariaDB Command" 30; then
    echo ""
    echo "=== DIAGNOSTIC INFORMATION ==="
    echo "MariaDB cmd service logs:"
    journalctl -u dbcalm-mariadb-cmd --no-pager -n 20 2>/dev/null || echo "No systemd logs available"

    echo ""
    echo "Checking if mariadb-cmd process is running:"
    ps aux | grep dbcalm-mariadb-cmd | grep -v grep || echo "Process not found"

    echo ""
    echo "Checking socket directory permissions:"
    ls -la /var/run/dbcalm/

    exit 1
fi

echo ""
# Check generic command socket
if ! check_socket "$CMD_SOCKET" "Generic Command" 30; then
    echo ""
    echo "=== DIAGNOSTIC INFORMATION ==="
    echo "Generic cmd service logs:"
    journalctl -u dbcalm-cmd --no-pager -n 20 2>/dev/null || echo "No systemd logs available"

    echo ""
    echo "Checking if cmd process is running:"
    ps aux | grep dbcalm-cmd | grep -v grep | grep -v mariadb || echo "Process not found"

    exit 1
fi

echo ""
echo "=== All sockets verified successfully ==="
