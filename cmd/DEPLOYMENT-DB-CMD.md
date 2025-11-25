# Deployment Guide

## Prerequisites

1. Build the binary:
```bash
cd /home/martijn/projects/dbcalm/go-backend
make build
```

2. Install the binary:
```bash
sudo cp dbcalm-db-cmd /usr/local/bin/
sudo chmod +x /usr/local/bin/dbcalm-db-cmd
```

## Systemd Service

Create `/etc/systemd/system/dbcalm-db-cmd.service`:

```ini
[Unit]
Description=DBCalm Database Command Server
After=network.target

[Service]
Type=simple
User=mysql
Group=mysql
ExecStart=/usr/local/bin/dbcalm-db-cmd
Restart=on-failure
RestartSec=5s

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/run/dbcalm /var/backups/dbcalm /var/lib/dbcalm

[Install]
WantedBy=multi-user.target
```

## Enable and Start Service

```bash
sudo systemctl daemon-reload
sudo systemctl enable dbcalm-db-cmd
sudo systemctl start dbcalm-db-cmd
sudo systemctl status dbcalm-db-cmd
```

## Verify

Check socket exists:
```bash
ls -l /var/run/dbcalm/db-cmd.sock
```

Test with Python client:
```bash
cd /home/martijn/projects/dbcalm/backend
python3 << EOF
from dbcalm_mariadb_cmd_client import Client
client = Client()
# This should connect without errors
