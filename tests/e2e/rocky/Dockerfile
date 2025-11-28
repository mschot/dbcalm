# E2E Test Environment for DBCalm RPM packages
# Uses Rocky Linux 9 to test RPM installation and functionality

FROM rockylinux:9

# Build argument to select database type (mariadb or mysql)
ARG DB_TYPE=mariadb

ENV TZ=UTC

# Install base dependencies including Go
RUN dnf install -y --allowerasing \
    golang \
    curl \
    openssl \
    sudo \
    procps \
    file \
    && dnf clean all

# Install Percona repository and XtraBackup
RUN dnf install -y https://repo.percona.com/yum/percona-release-latest.noarch.rpm \
    && percona-release enable-only tools release \
    && dnf install -y percona-xtrabackup-80 \
    && dnf clean all

# Install database server based on DB_TYPE build arg
RUN if [ "$DB_TYPE" = "mariadb" ]; then \
        dnf install -y mariadb-server mariadb mariadb-backup; \
    else \
        dnf install -y mysql-server mysql; \
    fi \
    && dnf clean all

# Create test directories with proper permissions
RUN mkdir -p /tests/fixtures \
    /tests/scripts \
    /tests/artifacts \
    /tests/logs \
    /tests/logs/dbcalm \
    /tests/test-results \
    /etc/cron.d && \
    chmod -R 777 /tests/logs /tests/test-results /tests/artifacts

# Set working directory
WORKDIR /tests

# Run entrypoint script
ENTRYPOINT ["/tests/scripts/entrypoint.sh"]
