# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [Unreleased]
### Removed
- `controller.Controller` from `Plugin` interface method parameters

## [1.1.0] - 2017-09-15
### Added
- Implementing new Plugins interface in Kanali v1.1.0
- [LICENSE](./LICENSE) file.
- Configuration items that take advantage of the decentralized Kanali configuration in `v1.1.5`
### Changed
- API key validation will not be preformed on OPTIONS requests

## [1.0.1] - 2017-08-08
### Changed
- Fixed a bug that caused a fatal runtime error when `spec.GranularRule` was `nil`

## [1.0.0] - 2017-07-29
### Added
- Initial Project Commit