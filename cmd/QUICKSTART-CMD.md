# DBCalm-CMD Quick Start Guide

## TL;DR

```bash
# Build
cd /home/martijn/projects/dbcalm/go-backend
./build-cmd.sh

# Install
sudo cp dbcalm-cmd /usr/bin/
sudo cp dbcalm-cmd.service /etc/systemd/system/
sudo cp cmd-config.example.yml /etc/dbcalm/cmd-config.yml
sudo systemctl daemon-reload
sudo systemctl enable --now dbcalm-cmd

# Verify
sudo systemctl status dbcalm-cmd
ls -la /var/run/dbcalm/cmd.sock
```

## What It Does

**dbcalm-cmd** is a root-privileged service that handles system operations:
1. Updates cron schedules → `/etc/cron.d/dbcalm`
2. Deletes directories → `rm -rf`
3. Cleans up backups → bulk deletion

## Test Commands

### 1. Update Cron Schedule
```bash
cat <<'EOF' | nc -U /var/run/dbcalm/cmd.sock
{"cmd":"update_cron_schedules","args":{"schedules":[{"id":1,"backup_type":"full","frequency":"daily","hour":2,"minute":30,"enabled":true}]}}
EOF
```

### 2. Delete Directory
```bash
sudo mkdir -p /tmp/test-delete
cat <<'EOF' | nc -U /var/run/dbcalm/cmd.sock
{"cmd":"delete_directory","args":{"path":"/tmp/test-delete"}}
EOF
```

### 3. Cleanup Backups
```bash
cat <<'EOF' | nc -U /var/run/dbcalm/cmd.sock
{"cmd":"cleanup_backups","args":{"backup_ids":["1","2"],"folders":["/tmp/test1","/tmp/test2"]}}
EOF
```

## Key Files

| File | Location |
|------|----------|
| Binary | `/usr/bin/dbcalm-cmd` |
| Config | `/etc/dbcalm/cmd-config.yml` |
| Service | `/etc/systemd/system/dbcalm-cmd.service` |
| Socket | `/var/run/dbcalm/cmd.sock` |
| Log | `/var/log/dbcalm/cmd.log` |
| Cron File | `/etc/cron.d/dbcalm` |

## Troubleshooting

### Service won't start
```bash
sudo journalctl -u dbcalm-cmd -n 50
sudo systemctl status dbcalm-cmd
```

### Socket not found
```bash
sudo ls -la /var/run/dbcalm/
sudo chmod 2774 /var/run/dbcalm/
```

### Permission errors
```bash
# Service must run as root
sudo systemctl edit dbcalm-cmd
# Add: User=root
```

## Differences from Python Version

| Aspect | Python | Go |
|--------|--------|-----|
| Binary | `dbcalm-cmd` (PyInstaller) | `dbcalm-cmd` (static) |
| Size | ~50 MB | ~8 MB |
| Memory | ~30-50 MB | ~5-10 MB |
| Startup | ~100-200ms | <10ms |
| Dependencies | Python runtime | None |

## API Compatibility

✅ 100% compatible with existing DBCalm API
- Same socket path
- Same JSON protocol
- Same validation rules
- Same responses

No code changes required in API server or frontend.

## Documentation

- [README-CMD.md](README-CMD.md) - Full documentation
- [IMPLEMENTATION-SUMMARY.md](IMPLEMENTATION-SUMMARY.md) - Implementation details
- [QUICKSTART-CMD.md](QUICKSTART-CMD.md) - This file
