# RPM Spec File for DBCalm

Name:           dbcalm
Version:        0.0.1
Release:        1%{?dist}
Summary:        Database backup and restore manager with web API

License:        BUSL-1.1
URL:            https://github.com/mschot/dbcalm-open-backend

# Runtime dependencies
# Note: mariadb-server or mysql-server should be installed separately based on user's preference
Requires:       openssl
Requires:       systemd

# Build dependencies (only needed for rpmbuild scriptlets, not for compiling)
BuildRequires:  systemd-rpm-macros

%description
DBCalm is a tool for managing database backups and restores with a web API
interface. It provides zero-downtime backups, point-in-time recovery, and
automated backup testing for MySQL and MariaDB databases.

%build
# Binaries are pre-built by build-rpm.sh using PyInstaller before rpmbuild
# No build steps needed here

%install
# Copy pre-built files from staging directory
# Files were built by build-rpm.sh using PyInstaller and staged in ../staging/
cp -r ../staging/* %{buildroot}/

%files
%{_bindir}/%{name}
%{_bindir}/%{name}-cmd
%{_bindir}/%{name}-db-cmd
%{_unitdir}/%{name}-api.service
%{_unitdir}/%{name}-cmd.service
%{_unitdir}/%{name}-db-cmd.service
%dir %{_sysconfdir}/%{name}
%config(noreplace) %{_sysconfdir}/%{name}/config.yml
%config(noreplace) %{_sysconfdir}/%{name}/credentials.cnf
%dir %{_localstatedir}/log/%{name}
%dir %{_localstatedir}/lib/%{name}
%dir %{_localstatedir}/run/%{name}
%dir %{_localstatedir}/backups/%{name}
%dir %{_datadir}/%{name}
%dir %{_datadir}/%{name}/scripts
%{_datadir}/%{name}/scripts/common-setup.sh

%post
# Post-installation script
project_name="dbcalm"

# Call shared common setup script
/usr/share/%{name}/scripts/common-setup.sh "$project_name"

# Reload systemd daemon
%systemd_post %{name}-api.service
%systemd_post %{name}-cmd.service
%systemd_post %{name}-db-cmd.service

%preun
# Pre-uninstallation script
%systemd_preun %{name}-api.service
%systemd_preun %{name}-cmd.service
%systemd_preun %{name}-db-cmd.service

%postun
# Post-uninstallation script
%systemd_postun_with_restart %{name}-api.service
%systemd_postun_with_restart %{name}-cmd.service
%systemd_postun_with_restart %{name}-db-cmd.service

# Only remove user/group on complete removal (not on upgrade)
if [ $1 -eq 0 ]; then
    userdel dbcalm 2>/dev/null || true
    groupdel dbcalm 2>/dev/null || true
fi

