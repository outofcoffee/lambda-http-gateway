# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).


## [0.2.0] - 2026-03-21
### Added
- feat: add subdomain-based routing and function prefix support (#1)

### Changed
- build: adds since release config
- build: ignore local agent settings
- chore(ci): bump actions to latest stable
- docs: adds changelog

## [0.1.9] - 2022-01-07
### Added
- feat: replaces bespoke stats endpoint with Prometheus metrics.

## [0.1.8] - 2022-01-05
### Fixed
- fix: checks environment variable instead of cached var for stats URL.

## [0.1.7] - 2022-01-05
### Added
- feat: allows stats recording and reporting to be enabled independently.
- feat: buffers hit channel to reduce likelihood of blocking caller.

## [0.1.6] - 2022-01-05
### Added
- feat: adds stats recorder.
- feat: adds stats reporter.

### Changed
- build: improves dependency caching.

## [0.1.5] - 2022-01-04
### Fixed
- fix: improves handling of empty environment variables.

## [0.1.4] - 2022-01-04
### Added
- feat: adds healthcheck endpoint.

## [0.1.3] - 2022-01-03
### Added
- feat: adds Docker image.

## [0.1.2] - 2022-01-03
### Added
- feat: improves error response status codes.
- feat: records proxy duration.

## [0.1.1] - 2022-01-03
### Changed
- build: adds goreleaser config.
- ci: adds GitHub Actions config.

## [0.1.0] - 2022-01-03
### Added
- feat: allows region to be configured.
- feat: improves logging.

### Other
- initial commit.
