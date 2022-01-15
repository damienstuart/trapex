
# Changelog
Follows the advice from [Keep A Changelog](https://keepachangelog.com/en/1.0.0/)

## Unreleased

### Added
* RPM creation (make rpm)
* Example AWS CloudFormation for CodeBuild
* Example Dockerfile defintion
* Prometheus exporter
* Basic secret support (filename or env variable retrieval of arguments that start with filename: or env:)
* Added 'capture' action to save traps to a file
* traplay command to replay traps to a destination
* trapbench command to replay a count of traps (or forever) for performance benchmarking purposes

### Changed
* Replaced bad configuration error reporting from panic() to fmt.Println() for saner error reporting
* Configuration files changed to YAML format
* Actions and counter reporting (ie metrics) now use a plugin architecture

### Known Issues
* Filter entries that specify an ipset that don't exist do not raise errors (ugh!)
* SNMPv3 Auth protocol of AES is not supported
* Filter lines that specify log directories that don't exist break, rather than creating the log directory
* Rate tracking can be slightly off due to race condition with ticker and getRate() calls

