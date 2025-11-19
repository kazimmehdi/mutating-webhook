# Contributing to Kubernetes Mutating Webhook

Thank you for your interest in contributing! This document provides guidelines for contributing to this project.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/yourusername/k8s-mutating-webhook.git`
3. Create a branch: `git checkout -b feature/my-feature`
4. Make your changes
5. Run tests: `make test`
6. Commit your changes: `git commit -am 'Add new feature'`
7. Push to your fork: `git push origin feature/my-feature`
8. Create a Pull Request

## Development Setup

### Prerequisites
- Go 1.21+
- Docker
- Kubernetes cluster (Minikube recommended for local development)
- Helm 3+

### Local Development

```bash
# Install dependencies
go mod download

# Build the project
make build

# Run tests
make test

# Build Docker image for Minikube
make docker-build-minikube

# Install to local cluster
make helm-install
```

## Code Style

- Follow standard Go formatting: `go fmt`
- Run linters: `golangci-lint run`
- Write meaningful commit messages
- Add tests for new features
- Update documentation

## Testing

```bash
# Run all tests
go test -v ./...

# Test the webhook in Kubernetes
make test-webhook
```

## Pull Request Process

1. Update the README.md with details of changes if needed
2. Update the CHANGELOG.md following the Keep a Changelog format
3. Ensure all tests pass
4. Request review from maintainers

## Reporting Bugs

Please use GitHub Issues to report bugs. Include:
- Description of the issue
- Steps to reproduce
- Expected behavior
- Actual behavior
- Environment details (Kubernetes version, etc.)

## Feature Requests

Feature requests are welcome! Please open an issue describing:
- The problem you're trying to solve
- Your proposed solution
- Any alternatives you've considered

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
