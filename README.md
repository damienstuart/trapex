# Trapex - A Program for Filtering, Translating, and Forwarding SNMP Traps
Trapex is an SNMP Trap proxy/forwarder.  It can receive, filter, manipulate, 
log, and forward SNMP traps to zero or mulitple destinations.  It can receive 
and process __SNMPv1__, __SNMPv2c__, or __SNMPv3__ traps.  

Presently, __v2c__ and __v3__ traps are converted to __v1__ before they are
logged and/or forwarded.  Support for sending other versions may be added in
a future release.

# Overview
The *trapex* program was created as a replacement for the aging _eHealth
Trap Exploder_ - which is a commercial product, and does not support __SNMPv3__.

*Trapex* has the following features:

* Receive SNMP v1/v2c/v3 traps
* Selectively ignore traps based on their SNMP version.
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
  * Log the trap data in a CSV format (specifically for feeding to a Clickhouse database).
  * Drop the trap and discontinue processing further filters.

This initial version's functionality and configuration options are very
similar to the current *eHealth Trapexploder* program, but there are some
differences, so be sure to carefully read this documentation if you will
be working with this program.

# Notes
This implementation uses external GO modules:

* [gosnmp](https://github.com/gosnmp/gosnmp)
(for all SNMP-based operations)
*  [lumberjack](https://github.com/natefinch/lumberjack)
(for logfile management). 

## Trapex Configuration
Trapex gets its configuration and runtime options from the configuration file.
Some options can be overridden by command-line arguments to _trapex_

Though the out-of-the box configuration file will have resonable defaults
and will have a filter directive for logging any traps, you should edit it
and set any configuration options as needed, and add/edit any other filter
directives. 

See the [trapex.conf file section](#markdown-header-the-trapex-configuration-file) below for details on the
configuration file and its directives

#### Running trapex
For a usage message for running *trapex*, run: `./trapex -h`
and you will get the following usage message:

```
Usage: trapex [-h] [-c <config_file>] [-b <bind_ip>] [-p <listen_port>]
              [-d] [-v]
  -h  - Show this help message and exit.
  -c  - Override the location of the trapex configuration file.
  -b  - Override the bind IP address on which to listen for incoming traps.
  -p  - Override the UDP port on which to listen for incoming traps.
  -d  - Enable debug mode (note: produces very verbose runtime output).
  -v  - Print the version of trapex and exit.
```

*trapex* will stay in the foreground and print information
to STDOUT.  On startup, any *filter* directives that forward a trap will
be printed as they are loaded from the configuration file. Here is an example:

```
Loading configuration for trapex version 0.9.2 from: /opt/trapex/etc/trapex.conf.
 -Added trap destination: 216.203.5.174, port 162
 -Added trap destination: 10.131.125.61, port 162
 -Added trap destination: 10.131.125.62, port 162
 -Added trap destination: 69.164.113.76, port 162
 -Added log destination: /opt/trapex/log/trapex.log
```

Also, any actions triggered by a signal will cause output to be printed to STDOUT as well.

#### Signals
*Trapex* has handlers for the following signals:

* *SIGHUP*

  Sending a SIGHUP to the *trapex* process ID (PID) will cause it to re-read
  the configuration file.  This is useful if you made a change in the
  configuration file and want to start using it without having to stop/start
  the service.

* *SIGUSR1*

  Sending a SIGUSR1 signal will cause *trapex* to dump its current trap
  stats (counters and trap rates) to STDOUT (or the trapex-run.log file).
  Here is a sample of one of these stat dumps:  

```
  Got SIGUSR1
  Trapex stats as of: Fri, 10 Jul 2020 08:35:30 EDT
   - Uptime..............: 0d-15h-41m-0s
   - Traps Received......: 2744275
   - Traps Processed.....: 2744275
   - Traps Dropped.......: 879493
   - Translated from v2c.: 288232
   - Translated from v3..: 497
   - Trap Rates (based on all traps received):
      - Last Minute......: 48
      - Last 5 Minutes...: 50
      - Last 15 Minutes..: 50
      - Last Hour........: 49
      - Last 4 Hours.....: 49
      - Last 8 Hours.....: 49
      - Last 24 Hours....: 32
      - Since Start......: 49
```

The trap rates above are calculated based on all incoming traps before any
filtering or processing takes place.  If the *ignoreVersion* option was used,
there would be a counter for *Traps Ignored* in the list.  Also, if `v2c`
and/or `v3` versions are ignored, the *Translated from vX* line will not be
included in the list.
 
Note that the *Traps Dropped* counter indicates traps that were matched on a
filter that has the `break` action.  Depending on the position of the filter
entry it matched, the trap may or may not have been forwarded or logged. This
is just an indicator that the trap did not traverse the entire filter list. 

* *SIGUSR2*

  Sending a SIGUSR2 signal will cause *trapex* to force a rotation of any
  configured CSV logs.  Since the CSV logs are meant to be used for feeding
  trap data to a database, this mechanism allows for doing a rotation
  on-demand so data can be synced to the database on a schedule.  

----

# The Trapex Configuration File
The _trapex_ configuration file (`trapex.conf`) is used to set the various
runtime options as well as the filtering, forwarding, and logging directives.

Blank lines are allowed, and those that start with `#` are for comments.

There are two types of directives in the `trapex.conf` file:

* Configuration directives:

  These are lines in the file that have *trapex* configuration options and
  their values. In most cases these line will have a parameter/option name
  and its corresponding value. Some directives (like 'debug') do not have 
  a value and are a boolean that is true when that entry is uncommented.

* Filter directives:

  These are the lines that define a filter for matching incoming traps and
  specifying an 'action' for traps that match the filter.  All of these
  directives start with the word "'filter'".


### CONFIGURATION DIRECTIVES
#### _General Options:_
* **debug**

  Enable `debug` mode. This causes *trapex* to print very verbose information
  on the incoming traps as well as the trap log entries to STDOUT.

* **trapexHost `<hostname>`**

  Set/overide the hostname for this trapex instance. If not specified, *trapex*
  will attempt to determine the local hostname and use that. Currently, this
  data is only used for the CSV log data.

* **listenAddress `<bind_ip>`**

  Specify the IP address on which to bind to listen for incoming traps. When
  set to a specific IP, only traps coming in to the network interface that
  has that IP will be received and processed. If not specified here or on
  the command-line, the default is `0.0.0.0` (all IPs).

* **listenPort `<port>`**

  Specify the UDP port on which to listen for incoming traps. If not set
  here or on the command-line, the default is 162.

* **ignoreVersions `<SNMP Version>[,<SNMP Version>]`**

  Specify one or more SNMP versions to ignore. Any traps that have a version
  that matches any listed here will be ignored and dropped by trapex. Valid
  versions are: `v1`, `v2c`, and `v3` (or just `1`, `2`, `3` would suffice as
  well).  Multiple entries are separated by a comma (no spaces). 

  **Note:**  
  Specifying all three versions will cause trapex to complain and exit at startup
  because no traps would be processed at all in that case.

#### _Log File Handling:_

These directive affect only the regular log files. They do not apply to the 
CSV-based logs.

* **logfileMaxSize `<value in MB>`**

  Specify the maximum size in MB the log files can grow before they are
  rotated.

* **logfileMaxBackups `<value>`**

  Specify how many backup (rotated) log files to keep. Older backups beyond
  this number will be removed. The backup log files are renamed with a date-
  time stamp.

* **compressRotatedLogs**

  When uncommented, this option causes backup log files to be compressed
  with gzip after they are rotated.

#### _SNMP v3 Setting:_
These are options for receiving SNMP v3 traps. Note that *trapex* currently
only supports the SNMP v3 *User-based Security Model* (USM).

* **v3msgFlags `<AuthPriv|AuthNoPriv|NoAuthNoPriv>`**

  This specifies the *SNMP v3 Message Flags*. Currently, *Trapex* supports
  only the `Auth` (Authentication) and `Priv` (Privacy) flags. These are 
  set via a single string as follows:
	* *AuthPriv* - Authentication and privacy
	* *AuthNoPriv* - Authentication and no privacy
  * *NoAuthNoPriv* - No authentication, and no privacy

* **v3User `<username>`**

  Set the SNMP v3 username. This is required for v3.

* **v3authProtocol `<MD5|SHA>`**

  Set the SNMP v3 *authentication protocol*. Valid values are `MD5` or
  `SHA` (default).  Note that this parameter is required if the Auth
  *Msg Flag* is set (v3msgFlags = `AuthNoPriv` or `AuthPriv`).

* **v3authPassword `<password>`**

  Set the SNMP v3 authentication password. This is required if Auth
  mode is set.

* **v3privProtocol `<AES|DES>`**

  Set the SNMP v3 *authentication protocol*. Valid values are `AES`
  (default) or `DES`.  Note that this parameter is required if Priv mode
  *Msg Flag* is set (v3msgFlags = `AuthPriv`).

* **v3authPassword `<password>`**

  Set the SNMP v3 privacy password. This is required if Priv mode is set.

### IP SETS
An IP Set is a named list of IP addresses that can be referenced in the
filter entries for the Source IP or Agent IP fields. The format is:
 
```
  ipset <ipset_name> {
      10.1.3.4
      10.1.3.5
      100.3.66.4
  }
```

You can also put multiple (whitespace-separated) IPs on a single line:

```
  ipset <ipset_name2> {
      10.1.3.4 10.1.3.5 100.3.66.4
      192.168.3.4 192.168.3.5 200.4.99.1 200.4.99.26
      10.222.121.7
  }
```

In the filter lines, you can then use "'ipset:<ipset_name>'" in either or
both the 'Source IP' or 'Agent Address' fields.

### FILTER DIRECTIVES
The *trapex* configuration *filter* directives are used for specifying which
traps are processed and what action is taken for traps that match the filter.

Each *filter* line starts with the word `filter` followed by the *filter
expressions*, the *action* for that filter, and for some actions, an option
argument for that action.

#### _Filter Expressions:_
The *filter expression* is a space separated set of 6 filter criteria for trap
data fields in the following order:

* **SNMP Version**

    The SNMP version. Only incoming traps that match this version are
    processed by this filter. Valid values are 'v1', 'v2c', or 'v3'.

* **Source IP**

    The source IP of the incoming trap packet.
    This can be a string match for a single IP address, a subnet in CIDR
    notation, or a regular expression.

* **Agent Address**

    The SNMNP v1 AgentAddr IP address.
    This can be a string match for a single IP address, a subnet in CIDR
    notation, or a regular expression.

* **Generic Type**

    The trap *Generic Type* (integer: 0-6).

* **Specific Type**

    The trap *Specific Type* (integer: 0-n).

* **Enterprise OID**

    The trap *Enterprise OID* value. This uses a regulare expression for
    matching.

An asterisk (`*`) can be used as a wildcard to indicate that any value for
that field matches. For instance, a filter that would match all traps and
forward them to 192.168.1.1 port 162 would look like this:

```
filter * * * * * * forward 192.168.1.1:162
```

If multiple fields are set to a non-wildcard value, then all of them have
to match (logical AND) in order for the trap to match and trigger the action.

#### _Filter Actions_

The *actions* that are currenly supported by *trapex* are:

* **forward <ip_address:port> [break]**

    Forward the trap to the specified IP address and port. *WARNING:* Do
    not specify the trapex host and port as a destination or you will
    create a trap forwarding loop! Note that this action also supports
    an optional second argument: 'break'. This tells trapex to stop
    processing this trap after the forward operation.

* **nat `<ip_address|$SRC_IP>`**

    Set the trap *AgentAddress* value to the specified IP address or use
    `$SRC_IP` to set it to the source IP of the trap packet.

* **log `</path/to/log/file>` [break]**

    Save the trap data to the specified log file. Any files created by log
    actions are subject to the log file handling configuration directives.
    Note that this action also supports an optional second argument: 'break'.
    This tells trapex to stop processing this trap after the log operation.

* **csv `</path/to/csv/file>` [break]**

    Save the trap data to the specified file in a CSV format that is meant
    specifically for feeding directly to a Clickhouse database. This feature
    is specific to the SungardAS snmp_trap table in Sungard's internal
    Clickhouse implementation. 

* **break**

    The *break* action means ignore this trap from this point forward - do
    not forward it or take any other actions - halt further filter processing
    and drop it.

#### _Filter Processing:_
The order of the filter directives in the configuration file is important.

The filters are processed in the order they appear in the configuration
file. When a trap is received, it is checked against each filter in order. If
it matches a filter, the trap data is processed by the *action* for that
filter, and that trap is checked against the next filter, and so on (unless
the action is `break` - where the trap is dropped and ignored from that point
on).

# Filter Examples
These are sample filter entries for various modes of filtering traps and
actions.

#### A sample IP Set
The IP addresses in this list can be referenced in filter directives below.
```
ipset test_ipset {
    10.1.3.4 10.1.3.5 100.3.66.4
    192.168.3.4 192.168.3.5 200.4.99.1
    10.222.121.7
}
```

#### Forward all traps to another host
```
filter * * * * * * forward 192.168.7.7:162
```

#### Log all traps.
Note that this would be all traps that have made it to this filter line
in the configuration file.
```
filter * * * * * * log /opt/trapex/log/trapex.log
```

#### Log all trap data to CSV file.
```
filter * * * * * * csv /opt/trapex/log/trapex.csv
```

#### Log only *coldStart* traps
This filters on a *Generic* trap type of `0` and a specific *Enterprise* OID.
```
filter * * * 0 * ^1\.3\.6\.1\.6\.3\.1\.1\.5 log /var/log/trapex-cold_start.log
```
When filtering on *Enterprise OID*, a *regular expression* (*regex*) is used.
When using a regex in a filter, you should always put a backslash (`\`) in
front of the dots to escape them.  Otherwise you may end up matching values
you did not intend to filter.

#### Filter on Source IP address
This filters a trap based on the SNMP packet source IP address - which may
not be the actual IP of the sending host if it is behind a NAT, or is a trap
that was forwarded from another system.

Here we send any traps from this source IP to a specific destination, then
`break` to drop it here to stop further processing for this trap.
```
filter * 10.66.48.1 * * * * forward 192.168.7.12:162
filter * 10.66.48.1 * * * * break
```

#### Filter on Source IP subnet
This is similar to the filter above accept that it will match all source
IPs in a subnet as specified via CIDR notation. In this case, IPs between
10.66.48.0 and 10.66.48.63
```
filter * 10.66.48.0/26 * * * * forward 192.168.7.12:162
filter * 10.66.48.0/26 * * * * break
```

#### Filter on Agent Address
#### Force agent_addr 0.0.0.0 to be source IP
Some traps might have 0.0.0.0 as the agent address. This filter forces
the agent address to be the same as the trap packet source IP address.
```
filter * * 0.0.0.0 * * * nat $SRC_IP
```

#### Filter on Agent Address
Filtering on Agent Address is just like filtering on the source IP except
that is is checking the *AgentAddr* value in the trap.
```
filter * * 10.66.48.1 * * * forward 192.168.7.12:162
filter * * 10.66.48.1 * * * break

# Or combine the two lines above into a single filter line by adding "break" using
# the optional second argument:
#
filter * * 10.66.48.1 * * * forward 192.168.7.12:162 break
```

#### Filters using an IP SET (defined earlier in the config file)
Here we drop traps for agent_addr IPs that are in the sample ip set: "test_ipset":
```
filter  * * ipset:test_ipset * * * break
```

#### IP filter using a regular expression
*Trapex does support using a regex for IP filters. To do this, the IP
address has to start with a forward slash (`/`) to let the filter
processor know it should be treated as a regex.

Here we want to filter on 4 different Agent IP addresses in the same /24
subnet that end with `.11`, `.22`, `.30`, and `.76`, then forword them
to a trap host that is listening on port 1162.
```
filter * * /^10\.131\.125\.(11|22|30|76)$ * * * forward 10.131.125.200:1162
``` 

### NAT Actions
There are cases where we may want to set the trap's *Agent Address* to
another value.  Common use-cases are changing the agent addr to match an
address that was NAT'ed, or where the agent address is invalid (like 0.0.0.0).

#### Change Agent Addr to another IP based on source IP
Here we want any traps with a packet source IP of 192.168.63.2 to have an
agent address of 10.16.20.2.
```
filter  * 192.168.63.2 * * * * nat 10.16.20.2
```

#### Make the Agent Addr match the Source IP
Here we make the *Agent Address* the same as the packet source IP.
We have one entry for a single IP and one for a subnet.
```
filter * * 10.22.0.98 * * * nat $SRC_IP
filter * * 10.122.0.0/16 * * * nat $SRC_IP
```

### Log Actions
For cases where you would like to log specific traps to a separate log file,
the 'log' action is used.

#### Log incoming v3 traps to a specific log file.
Here we want all incoming SNMPv3 trap to go to a log file called trapex-v3.log.
```
filter  v3 * * * * * log /opt/trapex/log/trapex-v3.log
```

#### Log only cold
Here we want only to log SNMP `cold start` traps from a agent addresses on a 
particular subnet, then ignore and drop the trap after that. This can be
done in a single filter entry by using a second arg of 'break'.
```
filter * * 10.22.0.0/16 0 * ^1\.3\.6\.1\.6\.3\.1\.1\.5 log /var/log/trapex-10.22-cold_start.log break
```

# Author

Trapex was writen by Damien Stuart - (<dstuart@dstuart.org>)
