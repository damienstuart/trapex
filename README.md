trapex
======

Trapex is an SNMP Trap proxy/forwarder.  It can receive, filter, manipulate, 
log, and forward SNMP traps to zero or mulitple destinations.  It can receive 
and process __SNMPv1__, __SNMPv2c__, or __SNMPv3__ traps.  

Presently, __v2c__ and __v3__ traps are converted to __v1__ before they are
logged and/or forwarded.  Support for sending other versions may be added in
a future release.

Overview
========
**damienstuart/trapex** was created as a replacement for the aging _eHealth 
Trap Exploder_ - which a commercial product, and does not support __SNMPv3__.

Trapex has the following features:

* Receive SNMP v1/v2c/v3 traps
* Filter traps based on one or more criteria:
  * Source IP address of the trap packet
  * Trap AgentAddress
  * Generic trap type
  * Specific trap type
  * Trap Enterprise OID
* Perform one of the followint actions based on a filter match:
  * Forward the trap
  * Change the AgentAddress value (_nat_ function)
  * Log the trap to a specified file
  * Drop the trap and discontinue processing further filters.
  
