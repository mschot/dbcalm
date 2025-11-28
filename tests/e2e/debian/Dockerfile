FROM ubuntu:22.04

# Build argument to select database type (mariadb or mysql)
ARG DB_TYPE=mariadb

# Prevent interactive prompts during package installation
ENV DEBIAN_FRONTEND=noninteractive
ENV TZ=UTC

# Install base dependencies
RUN apt-get update && apt-get install -y \
    software-properties-common \
    wget \
    lsb-release \
    gnupg \
    curl \
    ca-certificates \
    golang-1.21 \
    sudo \
    procps \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Add Go to PATH
ENV PATH="/usr/lib/go-1.21/bin:${PATH}"

# Install Percona repository and XtraBackup
RUN wget https://repo.percona.com/apt/percona-release_latest.$(lsb_release -sc)_all.deb \
    && dpkg -i percona-release_latest.$(lsb_release -sc)_all.deb \
    && percona-release enable-only tools release \
    && apt-get update \
    && apt-get install -y percona-xtrabackup-80 \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* percona-release_latest.*.deb

# Install database server based on DB_TYPE build arg
RUN apt-get update && \
    if [ "$DB_TYPE" = "mariadb" ]; then \
        apt-get install -y mariadb-server mariadb-client mariadb-backup; \
    else \
        apt-get install -y mysql-server mysql-client; \
    fi \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Create test directories with proper permissions
RUN mkdir -p /tests/fixtures \
    /tests/scripts \
    /tests/artifacts \
    /tests/logs \
    /tests/logs/dbcalm \
    /tests/test-results && \
    chmod -R 777 /tests/logs /tests/test-results /tests/artifacts

# Configure MariaDB/MySQL for testing
RUN mkdir -p /var/run/mysqld && \
    chown mysql:mysql /var/run/mysqld

# Expose ports
EXPOSE 8335 3306

# Set working directory
WORKDIR /tests

# Run entrypoint script (mounted as volume from common/)
ENTRYPOINT ["/tests/scripts/entrypoint.sh"]
