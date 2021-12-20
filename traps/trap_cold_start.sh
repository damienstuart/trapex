
# SNMP Cold start
snmptrap -v 2c -c public $DEST .1.3.6.1.6.3.1.1.5.3 .1.3.6.1.6.3.1.1.5.3 ifIndex i 2 ifAdminStatus i 1 ifOperStatus i 1

# Currently has error
# Error translating to v1: Invalid sysUptime (varbind0) for v2c trap: {.1.3.6.1.6.3.1.1.4.1.0 ObjectIdentifier .1.3.6.1.6.3.1.1.5.3}

