#!/bin/bash
# Common post-installation setup script for both DEBIAN and RPM packages
# This script runs AFTER files are extracted to set permissions and configure the system
# Note: User/group creation is handled by preinstall.sh (runs before file extraction)

project_name="dbcalm"

# Add mysql user to dbcalm group
if id mysql >/dev/null 2>&1; then
    if ! id -nG mysql | grep -qw "$project_name"; then
        usermod -a -G "$project_name" mysql
    fi
fi

# Set permissions on directories
chown -R mysql:$project_name /var/backups/$project_name/
chmod -R 770 /var/backups/$project_name/

mkdir -p /var/run/$project_name
chown -R mysql:$project_name /var/run/$project_name/
chmod -R 2770 /var/run/$project_name/

mkdir -p /var/log/$project_name
touch /var/log/$project_name/$project_name.log
chown -R mysql:$project_name /var/log/$project_name/
# Set setgid bit (2770) so new files inherit dbcalm group ownership
chmod -R 2770 /var/log/$project_name/
ls -la /var/log/$project_name/


chown -R mysql:$project_name /var/lib/$project_name/
# Set setgid bit (2775) so new files inherit dbcalm group ownership
chmod -R 2775 /var/lib/$project_name/

# Pre-create empty database file with correct permissions
# This ensures mysql and dbcalm users can both write to it
touch /var/lib/$project_name/db.sqlite3
chown mysql:$project_name /var/lib/$project_name/db.sqlite3
chmod 664 /var/lib/$project_name/db.sqlite3

# Set binary permissions
chown mysql:$project_name /usr/bin/$project_name
chmod 750 /usr/bin/$project_name

chown root:$project_name /usr/bin/$project_name-cmd
chmod 750 /usr/bin/$project_name-cmd

chown mysql:$project_name /usr/bin/$project_name-db-cmd
chmod 750 /usr/bin/$project_name-db-cmd

# Create SSL directory and generate self-signed certificate if not exists
ssl_dir="/etc/$project_name/ssl"
ssl_cert="$ssl_dir/fullchain-cert.pem"
ssl_key="$ssl_dir/private-key.pem"

mkdir -p "$ssl_dir"
chown $project_name:$project_name "$ssl_dir"
chmod 750 "$ssl_dir"

if [ ! -f "$ssl_cert" ] || [ ! -f "$ssl_key" ]; then
    echo "Generating self-signed SSL certificate for $project_name.localhost..."
    openssl req -x509 -newkey rsa:4096 -nodes \
        -keyout "$ssl_key" \
        -out "$ssl_cert" \
        -days 365 -subj "/CN=$project_name.localhost" 2>/dev/null

    chown $project_name:$project_name "$ssl_cert" "$ssl_key"
    chmod 640 "$ssl_cert" "$ssl_key"
    echo "Self-signed SSL certificate generated successfully."
fi

# Handle config.yml - generate JWT secret if placeholder exists
config_file="/etc/$project_name/config.yml"
if [ -f "$config_file" ]; then
    if grep -q "jwt_secret_key: replace_with_key" "$config_file"; then
        echo "Generating JWT secret key..."
        jwt_secret=$(openssl rand -hex 32)
        sed -i "s/jwt_secret_key: replace_with_key/jwt_secret_key: $jwt_secret/" "$config_file"
        echo "JWT secret key generated successfully."
    fi

    chown $project_name:$project_name "$config_file"
    chmod 640 "$config_file"
fi

# Create MariaDB credentials file for backups if it doesn't exist
credentials_file="/etc/$project_name/credentials.cnf"
if [ ! -f "$credentials_file" ]; then
    echo "Creating MariaDB credentials file for backups..."
    cat > "$credentials_file" <<'EOF'
# MariaDB credentials for DBCalm backups
# This file is used by mariabackup and mysqladmin
# The -dbcalm suffix allows using the same file for both tools

[client-dbcalm]
user=backupuser
password=changeme
host=localhost
EOF

    chown mysql:$project_name "$credentials_file"
    chmod 640 "$credentials_file"
    echo "MariaDB credentials file created at $credentials_file"
    echo "IMPORTANT: Update the password in $credentials_file before running backups!"
fi

exit 0
