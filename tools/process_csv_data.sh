#!/bin/bash
#
# process_csv_data
#
# Takes trapex CSV logs and loads them into the clickhouse database.
#
##############################################################################
#
TRAPEX_LOG_DIR="/opt/trapex/log"
CSV_ERROR_DIR="${TRAPEX_LOG_DIR}/CSV_error"
CSV_DONE_DIR="${TRAPEX_LOG_DIR}/CSV_done"

CSV_LOG_PATTERN="trapex-20??-*.csv"

# Set the Clickhouse database host
#
CH_HOST=${1:-"clickhouse_host"}
CH_PORT=${2:-"9000"}
CH_DB=${3:-"snmp_traps"}

# The trapex log directory must exist.
[[ -d "$TRAPEX_LOG_DIR" ]] || {
    echo "Trapex log directory: $TRAPEX_LOG_DIR, not found.  Aborting!" >&2
    exit 1
}

# Make sure the CVS directories exist.  Create them if they don't
[[ -d "$CSV_ERROR_DIR" ]] || mkdir -p "$CSV_ERROR_DIR"
[[ -d "$CSV_DONE_DIR" ]] || mkdir -p "$CSV_DONE_DIR"

# Force a rotation
/sbin/service trapex rotatecsv > /dev/null 2>&1

sleep 2

cd "$TRAPEX_LOG_DIR"
RCSVLIST=$(ls $CSV_LOG_PATTERN)
[[ -z "$RCSVLIST" ]] && return
for CSV in $RCSVLIST
do
    echo "Processing trapex CSV file: $CSV"
    cat $CSV | clickhouse-client --host=$CH_HOST --port=$CH_PORT --query="INSERT INTO $CH_DB FORMAT CSV" && {
        /bin/mv $CSV "${CSV_DONE_DIR}/"
        gzip -9 "${CSV_DONE_DIR}/$CSV" &
    } || {
        /bin/mv $CSV "${CSV_ERROR_DIR}/"
        echo " ** Error processing CSV file - moved to: ${CSV_ERROR_DIR}/$CSV"
    }
done
