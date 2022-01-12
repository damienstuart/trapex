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

%build
# Ummm..... We'll do that outside for the moment

%install
if [ -n "$CODEBUILD_SRC_DIR" ] ; then
    # AWS CodeBuild source directory
    cd $CODEBUILD_SRC_DIR
else
    cd ~/go/src/trapex
fi

mkdir -p %{buildroot}%{_sysconfdir}/systemd/system
install -m 750 tools/%{name}.service %{buildroot}%{_sysconfdir}/systemd/system

mkdir -p %{buildroot}/opt/%{name}/bin
install -m 644 README.md %{buildroot}/opt/%{name}
install -m 750 trapex %{buildroot}/opt/%{name}/bin
install -m 750 tools/process_csv_data.sh %{buildroot}/opt/%{name}/bin
install -m 750 cmds/traplay/traplay %{buildroot}/opt/%{name}/bin
install -m 750 cmds/trapbench/trapbench %{buildroot}/opt/%{name}/bin

mkdir -p %{buildroot}/opt/%{name}/etc
install -m 644 tools/trapex.yml %{buildroot}/opt/%{name}/etc

mkdir -p %{buildroot}/opt/%{name}/log

mkdir -p %{buildroot}/opt/%{name}/plugins/actions
for plugin in `ls -1 plugins/actions/*/*.so`; do
    install -m 750 $pluginsh %{buildroot}/opt/%{name}/plugins/actions
done

mkdir -p %{buildroot}/opt/%{name}/plugins/generators
for plugin in `ls -1 plugins/generators/*/*.so`; do
    install -m 750 $pluginsh %{buildroot}/opt/%{name}/plugins/generators
done

%files
%defattr(-,root,root)
%{_sysconfdir}/systemd/system/%{name}.service
%dir /opt/%{name}
%dir /opt/%{name}/bin
%dir /opt/%{name}/etc
%dir /opt/%{name}/log
%dir /opt/%{name}/clickhouse
%dir /opt/%{name}/clickhouse/exported
%dir /opt/%{name}/plugins
%dir /opt/%{name}/plugins/actions
%dir /opt/%{name}/plugins/generators
%dir /opt/%{name}/captured
/opt/%{name}/bin/trapex
%config(noreplace) /opt/%{name}/etc/trapex.yml
/opt/%{name}/README.md
/opt/%{name}/plugins/actions/*.so
/opt/%{name}/plugins/generators/*.so

%pre
# Check for upgrades
if [[ $1 -eq 1 || $1 -eq 2 ]]; then
    /usr/bin/systemctl daemon-reload
    /usr/bin/systemctl start %{name}.service
fi

%preun
%systemd_preun %{name}.service

