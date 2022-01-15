
# Changelog
Follows the advice from [Keep A Changelog](https://keepachangelog.com/en/1.0.0/)

## Unreleased

### Added
* RPM creation (make rpm)
* Example AWS CloudFormation for CodeBuild
* Docker container
* Prometheus exporter

### Changed
* Replaced bad configuration error reporting from panic() to fmt.Println() for saner error reporting
* Configuration files changed to YAML format
* Use plugin architecture

### Known Issues
* Filter entries that specify an ipset that don't exist does not raise an error
* SNMPv3 Auth protocol of AES is not supported
* Filter lines that specify log directories that don't exist break, rather than creating the log directory
* Rate tracking can be slightly off due to race condition with ticker and getRate() calls

