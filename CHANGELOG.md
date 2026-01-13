# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-01-12

### Added

- **Unified HTTP Trigger API**: Provides a unified RESTful API interface for triggering CI builds
- **API Key Authentication**: Supports multiple API Key configuration, providing minimum security boundary
- **Jenkins Integration**:
  - Parameterized triggering of Jenkins Pipeline/Job
  - Jenkins Token encapsulation, not exposed externally
  - CSRF protection support
  - Configurable request timeout
- **Audit Logging**:
  - Records all trigger requests and results
  - Provides audit log query API
  - SQLite storage, supports smooth upgrade to PostgreSQL
- **Storage System**:
  - SQLite database support
  - Automatic database migration
  - Audit log persistent storage
- **Logging System**:
  - Structured logging based on Go official slog library
  - Supports log level configuration (debug, info, warn, error)
  - JSON format output
- **Configuration Management**:
  - YAML configuration file support
  - Environment variable override
  - Configuration validation and default value settings
- **Docker Support**:
  - Dockerfile multi-stage build
  - Docker Compose configuration
  - Production environment deployment support
- **Test Coverage**:
  - Unit tests (authentication, configuration, storage)
  - Integration tests (API, Jenkins integration)
  - End-to-end tests (complete trigger flow)
- **Documentation**:
  - Complete README documentation
  - API documentation
  - Configuration guide
  - Development guide

### Technical Details

- **Language**: Golang 1.21+
- **Database**: SQLite 3+ (can be smoothly upgraded to PostgreSQL)
- **Authentication**: API Key
- **Logging**: slog (Go official logging library)
- **Deployment**: Docker

### Installation

- Build and install from source
- Run directly from source
- Docker containerized deployment

### Breaking Changes

None (initial version)

### Security

- API Key authentication mechanism
- Jenkins Token secure encapsulation
- Input parameter validation

### Performance

- Lightweight design
- Efficient database operations
- Supports concurrent request handling

---

## [1.0.1] - 2026-01-13

### Fixed

- Fixed security vulnerabilities and improved code quality
- Fixed golangci-lint errors in unit tests
- Fixed CI workflow duplicate step and restored code quality checks
- Fixed linter config error and unignore main package
- Resolved typecheck error with explicit yaml import alias
- Fixed GitHub Actions CI errors

### Changed

- Improved error handling and code quality
- Enhanced CI workflow for Go builds
- Updated Codecov action to v5 and added token support
- Updated golangci-lint configuration

### Test

- Improved test coverage and fixed test isolation issues
- Improved test reliability with mock Jenkins server and constants
- Fixed unit test golangci-lint errors

### CI/CD

- Added automated release workflow for Docker image publishing
- Enhanced CI workflow with better error handling

---

## [Unreleased]

### Planned

- Support for more CI engines (GitLab CI, GitHub Actions, CircleCI, Travis CI)
- Web UI management interface
- Distributed deployment support
- Log rotation functionality
- Monitoring and alerting mechanisms

[1.0.1]: https://github.com/nesnilnehc/triggermesh/releases/tag/v1.0.1
[1.0.0]: https://github.com/nesnilnehc/triggermesh/releases/tag/v1.0.0
