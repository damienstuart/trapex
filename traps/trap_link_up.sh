
# SNMP cold start
snmptrap -v 2c -c public $DEST '' .1.3.6.1.6.3.1.1.5.4 ifIndex i 3 ifAdminStatus i 1 ifOperStatus i 1


