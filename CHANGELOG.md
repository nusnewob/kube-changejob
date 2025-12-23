# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Initial release of kube-changejob operator
- ChangeTriggeredJob CRD for resource change monitoring
- Support for watching any Kubernetes resource
- Field-specific resource monitoring with JSONPath
- Configurable trigger conditions (Any/All)
- Cooldown period between job triggers
- Automatic job history management
- Webhook validation and defaulting
- Multi-architecture Docker images (amd64, arm64)
- Helm chart for easy deployment
- Comprehensive documentation

### Changed

### Deprecated

### Removed

### Fixed

### Security

---

## Release Notes

Releases are automatically created when tags are pushed. The changelog is generated from commit messages.

To maintain a good changelog:

- Use conventional commits (feat:, fix:, docs:, etc.)
- Write clear, descriptive commit messages
- Reference issues in commits (#123)

### Version Format

- **Major** (v1.0.0): Breaking changes
- **Minor** (v0.1.0): New features, backwards compatible
- **Patch** (v0.0.1): Bug fixes, backwards compatible
- **Pre-release**: v0.1.0-alpha.1, v0.1.0-beta.1, v0.1.0-rc.1

---

## Template for Manual Release Notes

When creating a release, you can use this template:

```markdown
## [0.1.0] - YYYY-MM-DD

### Added

- Feature description

### Changed

- Change description

### Fixed

- Bug fix description

### Security

- Security fix description
```

---

[Unreleased]: https://github.com/nusnewob/kube-changejob/compare/v0.1.0...HEAD
