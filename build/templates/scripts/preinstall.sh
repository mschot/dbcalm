#!/bin/bash
# Pre-installation script - creates user/group BEFORE files are extracted
# This ensures RPM/DEB can set correct file ownership during extraction

project_name="dbcalm"

# Create dbcalm group if it doesn't exist
getent group "$project_name" >/dev/null || groupadd -r "$project_name"

# Create dbcalm user if it doesn't exist
getent passwd "$project_name" >/dev/null || \
    useradd -r -g "$project_name" -d "/var/lib/$project_name" -s /sbin/nologin \
    -c "DBCalm service user" "$project_name"

exit 0
