# Contributing to kube-changejob

Thank you for your interest in contributing to kube-changejob! This document provides guidelines and instructions for contributing to the project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [How to Contribute](#how-to-contribute)
- [Coding Guidelines](#coding-guidelines)
- [Testing](#testing)
- [Submitting Changes](#submitting-changes)
- [Release Process](#release-process)
- [Getting Help](#getting-help)

## Code of Conduct

This project adheres to a Code of Conduct that all contributors are expected to follow. Please read [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) before contributing.

## Getting Started

### Prerequisites

Before you begin, ensure you have the following installed:

- **Go**: Version 1.24 or later (check `go.mod` for the exact version)
- **Docker**: For building container images
- **kubectl**: Kubernetes command-line tool
- **Kind** or **Minikube**: For local Kubernetes testing
- **Make**: For running build commands
- **Git**: For version control

### Fork and Clone

1. Fork the repository on GitHub
2. Clone your fork locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/kube-changejob.git
   cd kube-changejob
   ```
3. Add the upstream repository:
   ```bash
   git remote add upstream https://github.com/nusnewob/kube-changejob.git
   ```

## Development Setup

### Install Dependencies

```bash
# Download Go dependencies
go mod download

# Verify dependencies
go mod verify
```

### Run Locally

```bash
# Install CRDs into your Kubernetes cluster
make install

# Run the controller locally (against the configured Kubernetes cluster)
make run
```

### Build

```bash
# Build the binary
make build

# Build the Docker image
make docker-build
```

## How to Contribute

### Reporting Bugs

- Check if the bug has already been reported in [Issues](https://github.com/nusnewob/kube-changejob/issues)
- If not, create a new issue using the Bug Report template
- Provide as much detail as possible, including:
  - Steps to reproduce
  - Expected vs actual behavior
  - Kubernetes version and platform
  - Operator version
  - Relevant logs and manifests

### Suggesting Features

- Check if the feature has already been suggested
- Create a new issue using the Feature Request template
- Clearly describe the use case and proposed solution
- Be open to discussion and feedback

### Code Contributions

1. **Find or create an issue** - Make sure there's an issue for what you're working on
2. **Discuss the approach** - Comment on the issue to discuss your implementation approach
3. **Create a branch** - Use a descriptive branch name:
   ```bash
   git checkout -b feature/add-new-trigger-type
   git checkout -b fix/pod-cleanup-issue
   ```
4. **Make your changes** - Follow the coding guidelines below
5. **Test thoroughly** - Add tests and ensure all tests pass
6. **Submit a pull request** - Use the PR template and link to related issues

## Coding Guidelines

### Go Style

- Follow the official [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `gofmt` to format your code (included in the Makefile)
- Follow Go best practices and idioms
- Write clear, self-documenting code

### Code Quality

```bash
# Run linting
make lint

# Run code formatting
make fmt

# Run vet
make vet
```

### Project Structure

- `api/` - API definitions (CRDs)
- `cmd/` - Main applications
- `internal/controller/` - Controller implementations
- `config/` - Kubernetes manifests and configuration
- `test/` - Additional test files
- `hack/` - Scripts for development and CI

### Naming Conventions

- Use descriptive variable and function names
- Follow Go naming conventions (e.g., `MixedCaps` or `mixedCaps`)
- Exported names should be well-documented
- Use consistent naming across the codebase

### Documentation

- Add godoc comments for all exported types, functions, and constants
- Update README.md if your changes affect user-facing functionality
- Add inline comments for complex logic
- Include examples where appropriate

### Commit Messages

Write clear, concise commit messages:

```
<type>: <short summary>

<optional body>

<optional footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Adding or updating tests
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `chore`: Maintenance tasks
- `ci`: CI/CD changes

**Example:**
```
feat: add support for ConfigMap change triggers

Implements the ability to trigger jobs when ConfigMaps change.
Includes validation and status reporting.

Closes #42
```

## Testing

### Unit Tests

```bash
# Run unit tests
make test

# Run tests with coverage
make test-coverage
```

### Integration Tests

```bash
# Run integration tests
make test-integration
```

### End-to-End Tests

```bash
# Run E2E tests against a real cluster
make test-e2e
```

### Writing Tests

- Write tests for all new code
- Use the Ginkgo/Gomega testing framework (already used in the project)
- Aim for high test coverage, especially for critical paths
- Include both positive and negative test cases
- Test edge cases and error conditions

### Test Requirements

All pull requests must:
- Include appropriate tests
- Have all existing tests passing
- Maintain or improve code coverage

## Submitting Changes

### Pull Request Process

1. **Update your branch** with the latest upstream changes:
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

2. **Push your branch** to your fork:
   ```bash
   git push origin your-branch-name
   ```

3. **Open a pull request** on GitHub:
   - Use the pull request template
   - Provide a clear description of your changes
   - Link to related issues
   - Mark the PR as draft if it's not ready for review

4. **Address review feedback**:
   - Respond to comments
   - Make requested changes
   - Push updates to your branch

5. **Wait for approval**:
   - At least one maintainer must approve
   - CI checks must pass
   - All conversations must be resolved

### Review Process

- Maintainers will review your PR as soon as possible
- Be patient and responsive to feedback
- Reviews focus on:
  - Code quality and style
  - Test coverage
  - Documentation
  - Backwards compatibility
  - Security implications

## Release Process

Releases are managed by project maintainers. The process typically includes:

1. Version bump and changelog update
2. Tag creation
3. Container image build and push
4. GitHub release with release notes

Contributors don't need to worry about releases, but understanding the process helps context.

## Getting Help

### Communication Channels

- **GitHub Issues**: For bugs and feature requests
- **GitHub Discussions**: For questions and general discussion
- **Pull Request Comments**: For code review discussions

### Questions?

- Check existing issues and discussions first
- Create a new discussion for general questions
- Ask in pull request comments for specific code questions
- Be respectful and patient

## Recognition

All contributors will be recognized in:
- GitHub contributor list
- Release notes (for significant contributions)
- Project documentation (where appropriate)

Thank you for contributing to kube-changejob! 🎉
