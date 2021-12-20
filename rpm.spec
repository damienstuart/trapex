Name: trapex
Version: 0.9.6
Release: 1
License: MIT License
Summary: SNMP trap receiver and forwarder to multiple destinations
URL: https://github.com/damienstuart/trapex
BuildRequires: systemd

%description
Trapex is an SNMP Trap proxy/forwarder.  It can receive, filter, manipulate, 
log, and forward SNMP traps to zero or mulitple destinations.  It can receive 
and process SNMP v1, v2c, or v3 traps.  

Presently, v2c and v3 traps are converted to v1 before they are
logged and/or forwarded.  Support for sending other versions may be added in
a future release.

%build
# Ummm..... We'll do that outside for the moment

%install
# Use the default assumed Golang directories
cd ~/go/src/trapex

# systemd-fu
mkdir -p %{buildroot}%{_sysconfdir}/systemd/system
install -m 750 %{name}.service %{buildroot}%{_sysconfdir}/systemd/system

mkdir -p %{buildroot}/opt/%{name}/bin
install -m 750 trapex %{buildroot}/opt/%{name}/bin

%files
%defattr(-,root,root)
install -m 750
%{buildroot}%{_sysconfdir}/systemd/system/%{name}.service
%dir /opt/%{name}
%dir /opt/%{name}/bin
%dir /opt/%{name}/etc
/opt/%{name}/bin/trapex
/opt/%{name}/etc/trapex.conf
/opt/%{name}/README.md

%pre

# Check for upgrades
if [ $1 -eq 1 ]; then
    /usr/bin/systemctl daemon-reload
    /usr/bin/systemctl start %{name}.service
fi
if [ $1 -eq 2 ]; then
    /usr/bin/systemctl daemon-reload
    /usr/bin/systemctl start %{name}.service
fi

%preun
%systemd_preun %{name}.service

