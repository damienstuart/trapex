# Table structure in a Clickhouse database:

CREATE TABLE snmp_traps
(
    `TrapDate` Date,
    `TrapTimestamp` DateTime,
    `TrapHost` LowCardinality(String),
    `TrapNumber` UInt32,
    `TrapSourceIP` IPv4,
    `TrapAgentAddress` IPv4,
    `TrapGenericType` UInt8,
    `TrapSpecificType` UInt32,
    `TrapEnterpriseOID` String,
    `TrapVarBinds.ObjID` Array(String),
    `TrapVarBinds.Value` Array(String)
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(TrapDate)
ORDER BY (TrapTimestamp, TrapHost, TrapAgentAddress, TrapEnterpriseOID, TrapGenericType)
TTL TrapDate + toIntervalYear(3)
SETTINGS index_granularity = 8192
