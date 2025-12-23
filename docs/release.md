---
layout: default
title: Release Process
nav_order: 7
description: "How to create releases for kube-changejob"
permalink: /release
---

# Release Process

This document describes how to create releases for kube-changejob.

## Overview

Releases are automatically built and published when a version tag is pushed to the repository. The release process includes:

- Building and publishing Docker images to GitHub Container Registry
- Packaging and publishing Helm charts to GitHub Container Registry
- Creating source code archives (`.tar.gz` and `.zip`)
- Generating Kubernetes manifests (CRDs and full installation)
- Creating a GitHub Release with changelog

## Release Types

### Stable Release

Format: `v<major>.<minor>.<patch>` (e.g., `v1.0.0`, `v0.2.5`)

Stable releases are production-ready versions that are fully tested and documented.

### Pre-Release

Formats:

- Alpha: `v<version>-alpha.<number>` (e.g., `v1.0.0-alpha.1`)
- Beta: `v<version>-beta.<number>` (e.g., `v1.0.0-beta.1`)
- Release Candidate: `v<version>-rc.<number>` (e.g., `v1.0.0-rc.1`)

Pre-releases are marked as such in GitHub and are not considered stable.

## Prerequisites

Before creating a release:

1. **All tests pass**: Ensure CI/CD pipeline is green
2. **Documentation is updated**: Update README, docs, and CHANGELOG
3. **Version is decided**: Follow [Semantic Versioning](https://semver.org/)
4. **On main branch**: Releases should be created from `main` (unless hotfix)
5. **Clean working directory**: No uncommitted changes

## Creating a Release

### Option 1: Using the Helper Script (Recommended)

```bash
# Create a release
./hack/create-release.sh v0.1.0

# The script will:
# 1. Validate the version format
# 2. Check git status
# 3. Update Helm chart versions
# 4. Show what will be released
# 5. Create and push the tag
```

### Option 2: Manual Process

1. **Update Helm Chart Version**

```bash
# Edit dist/chart/Chart.yaml
version: 0.1.0
appVersion: "0.1.0"
```

2. **Commit Version Changes**

```bash
git add dist/chart/Chart.yaml
git commit -m "chore: bump version to v0.1.0"
git push origin main
```

3. **Create and Push Tag**

```bash
git tag -a v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0
```

## What Happens During Release

When a tag is pushed, the GitHub Actions workflow automatically:

### 1. Build Docker Images

- Multi-architecture images (amd64, arm64)
- Tagged with version, major.minor, major, and latest
- Published to `ghcr.io/nusnewob/kube-changejob`

```bash
ghcr.io/nusnewob/kube-changejob:v0.1.0
ghcr.io/nusnewob/kube-changejob:0.1
ghcr.io/nusnewob/kube-changejob:0
ghcr.io/nusnewob/kube-changejob:latest
```

### 2. Package Helm Chart

- Chart version updated to match release
- Chart published to `oci://ghcr.io/nusnewob/charts`

```bash
helm pull oci://ghcr.io/nusnewob/charts/kube-changejob --version 0.1.0
```

### 3. Generate Manifests

Two manifest files are created:

**`crds.yaml`** - CRDs only

```bash
kubectl apply -f https://github.com/nusnewob/kube-changejob/releases/download/v0.1.0/crds.yaml
```

**`install.yaml`** - Full installation (CRDs + controller + RBAC + webhooks)

```bash
kubectl apply -f https://github.com/nusnewob/kube-changejob/releases/download/v0.1.0/install.yaml
```

### 4. Create Source Archives

- `.tar.gz` archive for Linux/macOS
- `.zip` archive for Windows
- Both exclude `.git`, `dist`, and `vendor` directories

### 5. Generate Changelog

Automatically generated from git commits since the last release, including:

- List of changes with commit hashes
- Installation instructions
- Container image references
- Helm chart references
- Link to full diff

### 6. Create GitHub Release

- Release page created with changelog
- All artifacts attached
- SHA256 checksums provided
- Pre-release flag set for alpha/beta/rc versions

## Verifying a Release

After creating a release, verify the following:

### 1. GitHub Release Page

Visit: `https://github.com/nusnewob/kube-changejob/releases/tag/v0.1.0`

Check:

- ✅ Changelog is generated correctly
- ✅ All files are attached (install.yaml, crds.yaml, helm chart, source archives, checksums.txt)
- ✅ Pre-release flag is correct

### 2. Docker Images

```bash
# Pull and verify image
docker pull ghcr.io/nusnewob/kube-changejob:v0.1.0

# Check image labels
docker inspect ghcr.io/nusnewob/kube-changejob:v0.1.0 | jq '.[0].Config.Labels'

# Verify multi-arch
docker manifest inspect ghcr.io/nusnewob/kube-changejob:v0.1.0
```

### 3. Helm Chart

```bash
# Search for chart
helm search repo kube-changejob --versions

# Pull chart
helm pull oci://ghcr.io/nusnewob/charts/kube-changejob --version 0.1.0

# Verify chart contents
tar -xzf kube-changejob-0.1.0.tgz
cat kube-changejob/Chart.yaml
```

### 4. Kubernetes Manifests

```bash
# Download and verify CRDs
curl -sL https://github.com/nusnewob/kube-changejob/releases/download/v0.1.0/crds.yaml | kubectl apply --dry-run=client -f -

# Download and verify full installation
curl -sL https://github.com/nusnewob/kube-changejob/releases/download/v0.1.0/install.yaml | kubectl apply --dry-run=client -f -
```

### 5. Checksums

```bash
# Download release assets
curl -sL https://github.com/nusnewob/kube-changejob/releases/download/v0.1.0/checksums.txt

# Verify checksums
curl -sL https://github.com/nusnewob/kube-changejob/releases/download/v0.1.0/install.yaml > install.yaml
sha256sum install.yaml
# Compare with checksums.txt
```

## Post-Release Tasks

After a successful release:

1. **Announce the Release**
   - Create announcement in Discussions
   - Update project website if applicable
   - Share on social media/mailing lists

2. **Update Documentation**
   - Verify docs are published to GitHub Pages
   - Update any external documentation links

3. **Monitor Issues**
   - Watch for issues related to the new release
   - Be prepared to create a hotfix if critical bugs are found

4. **Plan Next Release**
   - Create milestone for next version
   - Triage issues and PRs for next release

## Hotfix Releases

For critical bug fixes that can't wait for the next regular release:

1. Create a hotfix branch from the release tag:

```bash
git checkout -b hotfix/v0.1.1 v0.1.0
```

2. Fix the bug and commit:

```bash
git commit -m "fix: critical bug description"
```

3. Create the hotfix release:

```bash
./hack/create-release.sh v0.1.1
```

4. Merge the hotfix back to main:

```bash
git checkout main
git merge hotfix/v0.1.1
git push origin main
```

## Troubleshooting

### Release Workflow Failed

1. Check the Actions tab for error details
2. Common issues:
   - Docker build failed: Check Dockerfile and dependencies
   - Manifest generation failed: Run `make manifests` locally
   - Helm packaging failed: Run `helm lint dist/chart`

### Tag Already Exists

If you need to recreate a tag:

```bash
# Delete local tag
git tag -d v0.1.0

# Delete remote tag
git push origin :refs/tags/v0.1.0

# Recreate tag
git tag -a v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0
```

**Warning**: Only do this for tags that haven't been widely distributed.

### Docker Image Not Found

Wait a few minutes after the workflow completes. Images may take time to become available in GHCR.

### Helm Chart Not Found

Helm charts may take a few minutes to propagate after being pushed to GHCR.

## Release Checklist

Use this checklist when creating a release:

- [ ] All tests pass on main branch
- [ ] Documentation is up to date
- [ ] CHANGELOG is updated (if manual)
- [ ] Version number follows semver
- [ ] Helm chart version is updated
- [ ] No uncommitted changes
- [ ] On correct branch (usually main)
- [ ] Tag created and pushed
- [ ] Release workflow succeeded
- [ ] GitHub release page looks correct
- [ ] Docker images are available
- [ ] Helm chart is available
- [ ] Manifests work correctly
- [ ] Checksums are correct
- [ ] Release announced
- [ ] Documentation updated

## Reference

- [Semantic Versioning](https://semver.org/)
- [Keep a Changelog](https://keepachangelog.com/)
- [GitHub Releases](https://docs.github.com/en/repositories/releasing-projects-on-github)
- [OCI Registry for Helm](https://helm.sh/docs/topics/registries/)

## Questions?

If you have questions about the release process:

- Open a discussion in GitHub Discussions
- Contact the maintainers
- Review previous release workflows
