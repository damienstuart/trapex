# Performance Testing Tool
## Goals
This tool supports load testing of the system, by providing a streaming service
to generate SNMP traps and send them to a traphost. Essentially, this is `spray` (TCP)
or `ApacheBench` (HTTP) for SNMP (UDP) traffic.

Features:

* generates bogus traffic for a specific number of traps (default 100) or forever
* plugins that support generating traffic
   * resend captured traps in a loop
   * fake traffic with some uniqueness
* monitoring hooks to display current performance information
* allows all actions available to trapex
* configuration file allows for multiple destinations, each with own configuration


ab: wrong number of arguments
Usage: ab [options] [http[s]://]hostname[:port]/path
Options are:
    -n requests     Number of requests to perform
    -c concurrency  Number of multiple requests to make at a time
    -t timelimit    Seconds to max. to spend on benchmarking
                    This implies -n 50000
    -s timeout      Seconds to max. wait for each response
                    Default is 30 seconds
    -b windowsize   Size of TCP send/receive buffer, in bytes
    -B address      Address to bind to when making outgoing connections
    -p postfile     File containing data to POST. Remember also to set -T
                    Default is 'text/plain'
    -v verbosity    How much troubleshooting info to print
    -w              Print out results in HTML tables
    -V              Print version number and exit
    -d              Do not show percentiles served table.
    -S              Do not show confidence estimators and warnings.
    -q              Do not show progress when doing more than 150 requests
    -g filename     Output collected data to gnuplot format file.
    -e filename     Output CSV file with percentages served
    -r              Don't exit on socket receive errors.
    -h              Display usage information (this message)
