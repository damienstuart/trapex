
# Changelog
Follows the advice from [Keep A Changelog](https://keepachangelog.com/en/1.0.0/)

## Unreleased

### Added
* RPM creation (make rpm)
* Example AWS CloudFormation for CodeBuild
* Build support for Windows
* Docker container
* Prometheus exporter

### Changed
* Replaced bad configuration error reporting from panic() to fmt.Println() for saner error reporting
* Configuration files changed to YAML format

### Known Issues
* Filter entries that specify an ipset that don't exist does not raise an error
* SNMPv3 Auth protocol of AES is not supported

