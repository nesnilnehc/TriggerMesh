# Contributing to TriggerMesh

Thank you for your interest in the TriggerMesh project! We welcome contributions of all kinds.

## How to Contribute

### Reporting Bugs

If you've found a bug, please report it via GitHub Issues:

1. Check if the issue already exists
2. If not, create a new issue
3. Provide the following information:
   - A clear description of the bug
   - Steps to reproduce
   - Expected behavior
   - Actual behavior
   - Environment information (Go version, OS, etc.)
   - Relevant logs or error messages

### Feature Requests

If you have a feature suggestion:

1. Check if there's already a related issue
2. Create a new issue describing your idea
3. Explain the use case and expected outcome

### Code Contributions

#### Development Environment Setup

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/your-username/triggermesh.git
   cd triggermesh
   ```
3. Install dependencies:
   ```bash
   go mod tidy
   ```
4. Create a feature branch:
   ```bash
   git checkout -b feature/your-feature-name
   ```

#### Code Style

- Follow the official Go code style guidelines
- Use `go fmt` to format code
- Use `go vet` to check code
- Ensure all tests pass

#### Commit Guidelines

Commit messages should clearly describe the changes:

```
[type] Short description

Detailed description (optional)

- Change point 1
- Change point 2
```

Types include:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation update
- `style`: Code style (no functional changes)
- `refactor`: Refactoring
- `test`: Test related
- `chore`: Build/tool related

#### Testing Requirements

- Add tests for new features
- Ensure all tests pass: `go test ./...`
- Run test coverage: `make coverage`
- Ensure core functionality test coverage ≥ 80%

#### Pull Request Process

1. Ensure code passes all tests and checks
2. Update relevant documentation (if needed)
3. Submit PR with:
   - Clear PR title and description
   - Related issue number (e.g., `Fixes #123`)
   - Test instructions
4. Wait for code review
5. Make changes based on feedback

#### Code Review

- All PRs require code review
- Maintainers will provide feedback in PRs
- Please respond to review comments promptly

## Development Guide

### Project Structure

```
triggermesh/
├── cmd/triggermesh/     # Application entry point
├── internal/            # Internal packages
│   ├── api/            # API handling
│   ├── config/         # Configuration management
│   ├── engine/         # CI engine abstraction layer
│   ├── logger/         # Logging system
│   └── storage/        # Storage layer
├── tests/              # Test files
└── docs/               # Documentation
    ├── README.md       # Documentation index
    └── tutorial.md     # Detailed Chinese tutorial
```

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make coverage

# Run tests for a specific package
go test ./internal/config/...
```

### Building the Project

```bash
# Build binary
make build

# Run the project
make run
```

### Using Docker

```bash
# Build Docker image
make docker-build

# Run container
make docker-run
```

## Code of Conduct

Please follow the project's [Code of Conduct](CODE_OF_CONDUCT.md) and respect other contributors.

## Questions

If you have any questions, please contact us via:

- Create a GitHub Issue
- Participate in project discussions

Thank you for your contributions!
