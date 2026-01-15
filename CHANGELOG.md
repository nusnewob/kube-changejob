# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

### Changed

### Deprecated

### Removed

### Fixed

### Security

---

## [v0.1.0-alpha.3] - 2026-01-15

### Added

- Additional tests to improve test coverage (454fc68)
- Webhook validation for spec.cooldown value (c218484)

### Changed

- Cleanup and improve github pipelines (931d920)
- Remove duplicate docs (a7fad88)
- Reduce unnecessary rbac permission for watched resources (a598dab)
- Bump github.com/onsi/gomega from 1.38.3 to 1.39.0 (ce7d678)
- Bump github.com/onsi/ginkgo/v2 from 2.27.3 to 2.27.4 (29887b5)
- Bump go.uber.org/zap from 1.27.0 to 1.27.1 (6527c87)

### Fixed

- Bug 2 unsafe json.Marshal ordering (dc08b79)
- Always update status during controller reconcile (9167811)

## [v0.1.0-alpha.2] - 2025-12-30

### Added

- Configurable RBAC for watched resources in helm (ec701e5)
- Tests to cover more test cases (209be8f)

### Changed

- Refactor e2e tests to use k8s go client instead of kubectl, improve speed and efficiency (9b43f01)
- Update github action triggers allow manual run (aa16588)
- Update codecov.yml (b7dda05, 29f9e2d)
- Improve controller logic, add additional validation (16cdce6)
- Consolidate update status logic and function (107e066)
- Remove unnecessary code (7a23c83)
- Cleanup controller utility functions (2a590ab)
- Bump github.com/onsi/ginkgo/v2 from 2.25.3 to 2.27.3 (c38862f)

### Fixed

- Fix controller logic and premature returns (b11fa65)

### Documentation

- Update readme.md (84501aa)

## [v0.1.0-alpha.1] - 2025-12-23

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
- Logging options for controller cmd (81d7ba8)
- Shortname for ChangeTriggeredJob CRD (8c9171e)
- GitHub page docs and readme (db80441)
- GitHub actions for release, docs, and wiki pages (487cdfc, 40d7ec8, 55dbcc2)
- Codecov github action (1112bb1)
- Tests for cmd logging options (a4b433d)
- Additional tests (6f0671e)

### Fixed

- Release pipeline (05bd8c7)
- GitHub action to sync docs to wiki (9e0a35f)
- Docs links (24bc4c6)
- GitHub actions for docs (55dbcc2)
- Helm chart test (38e40cc)
- Linting errors (020ab6c)

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

[Unreleased]: https://github.com/nusnewob/kube-changejob/compare/v0.1.0-alpha.3...HEAD
[v0.1.0-alpha.3]: https://github.com/nusnewob/kube-changejob/compare/v0.1.0-alpha.2...v0.1.0-alpha.3
[v0.1.0-alpha.2]: https://github.com/nusnewob/kube-changejob/compare/v0.1.0-alpha.1...v0.1.0-alpha.2
[v0.1.0-alpha.1]: https://github.com/nusnewob/kube-changejob/releases/tag/v0.1.0-alpha.1
