# Release Automation Summary

This document provides an overview of the automated release system for kube-changejob.

## 📦 What's Included

The release automation system includes:

### ✅ Automated Workflows

1. **Release Workflow** (`.github/workflows/release.yml`)
   - Triggered by version tags (e.g., `v0.1.0`)
   - Builds and publishes all release artifacts
   - Creates GitHub release with changelog

2. **Pre-Release Validation** (`.github/workflows/pre-release.yml`)
   - Runs on every push to main
   - Validates build, tests, and manifests
   - Shows release readiness status

### ✅ Release Artifacts

Each release includes:

1. **Docker Images** (multi-arch: amd64, arm64)
   - Published to: `ghcr.io/nusnewob/kube-changejob`
   - Tags: version, major.minor, major, latest

2. **Helm Chart** (OCI format)
   - Published to: `oci://ghcr.io/nusnewob/charts/kube-changejob`

3. **Kubernetes Manifests**
   - `install.yaml` - Complete installation
   - `crds.yaml` - CRDs only

4. **Source Archives**
   - `.tar.gz` for Linux/macOS
   - `.zip` for Windows

5. **Checksums**
   - SHA256 hashes for all artifacts

6. **Changelog**
   - Auto-generated from git commits
   - Includes installation instructions

### ✅ Helper Tools

1. **Release Script** (`hack/create-release.sh`)
   - Interactive release creation
   - Validates prerequisites
   - Updates versions automatically

2. **Documentation** (`docs/RELEASE.md`)
   - Comprehensive release guide
   - Best practices
   - Troubleshooting tips

## 🚀 Quick Start

### Creating Your First Release

```bash
# 1. Ensure you're on main branch with latest changes
git checkout main
git pull origin main

# 2. Run the release script
./hack/create-release.sh v0.1.0

# 3. The script will:
#    - Validate the environment
#    - Update Helm chart versions
#    - Show what will be released
#    - Create and push the tag

# 4. Monitor the workflow
# Visit: https://github.com/nusnewob/kube-changejob/actions

# 5. Once complete, verify the release
# Visit: https://github.com/nusnewob/kube-changejob/releases
```

## 📋 Release Checklist

Before creating a release:

- [ ] All tests pass on main branch
- [ ] Documentation is up to date
- [ ] Breaking changes are documented
- [ ] Version follows semantic versioning
- [ ] No uncommitted changes
- [ ] Ready to announce the release

## 🔍 What Happens During Release

### 1. Tag Push Detection (Trigger)
```
Developer pushes tag → GitHub Actions triggered
```

### 2. Build Phase
```
✓ Extract version from tag
✓ Set up build environment (Go, Docker, Helm)
✓ Run tests
✓ Generate manifests
```

### 3. Docker Image Phase
```
✓ Build multi-arch images (amd64, arm64)
✓ Tag with version and latest
✓ Push to ghcr.io
```

### 4. Helm Chart Phase
```
✓ Update chart version
✓ Package chart
✓ Push to OCI registry (ghcr.io)
```

### 5. Manifest Generation Phase
```
✓ Generate CRDs (crds.yaml)
✓ Generate full installation (install.yaml)
✓ Update image references to release version
```

### 6. Source Archive Phase
```
✓ Create .tar.gz archive
✓ Create .zip archive
✓ Generate SHA256 checksums
```

### 7. Changelog Generation Phase
```
✓ Compare with previous tag
✓ Generate changelog from commits
✓ Add installation instructions
✓ Add artifact links
```

### 8. GitHub Release Phase
```
✓ Create release on GitHub
✓ Upload all artifacts
✓ Set pre-release flag if alpha/beta/rc
✓ Generate release notes
```

## 📊 Release Outputs

### Docker Images

```bash
# Pull the image
docker pull ghcr.io/nusnewob/kube-changejob:v0.1.0

# Available tags
ghcr.io/nusnewob/kube-changejob:v0.1.0
ghcr.io/nusnewob/kube-changejob:0.1
ghcr.io/nusnewob/kube-changejob:0
ghcr.io/nusnewob/kube-changejob:latest
```

### Helm Chart

```bash
# Install using Helm
helm install kube-changejob \
  oci://ghcr.io/nusnewob/charts/kube-changejob \
  --version 0.1.0

# Pull chart
helm pull oci://ghcr.io/nusnewob/charts/kube-changejob --version 0.1.0
```

### Kubernetes Manifests

```bash
# Install everything
kubectl apply -f https://github.com/nusnewob/kube-changejob/releases/download/v0.1.0/install.yaml

# Install CRDs only
kubectl apply -f https://github.com/nusnewob/kube-changejob/releases/download/v0.1.0/crds.yaml
```

### Source Code

```bash
# Download source
curl -LO https://github.com/nusnewob/kube-changejob/releases/download/v0.1.0/kube-changejob-0.1.0.tar.gz

# Verify checksum
curl -sL https://github.com/nusnewob/kube-changejob/releases/download/v0.1.0/checksums.txt | grep tar.gz
sha256sum kube-changejob-0.1.0.tar.gz
```

## 🛡️ Security Features

1. **Image Signing**: Docker images are signed with provenance
2. **Checksum Verification**: SHA256 checksums for all artifacts
3. **Security Scanning**: Trivy scans during pre-release validation
4. **Multi-arch Support**: Native builds for amd64 and arm64
5. **Minimal Images**: Based on distroless or minimal base images

## 🔄 Release Cadence

Recommended release schedule:

- **Patch releases** (v0.1.x): As needed for bug fixes
- **Minor releases** (v0.x.0): Monthly or feature-based
- **Major releases** (vx.0.0): When breaking changes are introduced

## 📚 Additional Resources

- **Full Documentation**: `docs/RELEASE.md`
- **Changelog**: `CHANGELOG.md`
- **Contributing Guide**: `CONTRIBUTING.md`
- **Release Notes Template**: `.github/RELEASE_NOTES.md`

## 💡 Tips

1. **Use Semantic Versioning**: Always follow semver (MAJOR.MINOR.PATCH)
2. **Write Good Commit Messages**: They become your changelog
3. **Test Before Releasing**: Run pre-release validation locally
4. **Announce Releases**: Update docs and notify users
5. **Monitor After Release**: Watch for issues with new version

## 🐛 Troubleshooting

### Release workflow failed?
- Check Actions tab for detailed logs
- Verify Docker builds locally
- Ensure manifests generate correctly

### Images not available?
- Wait 2-3 minutes after workflow completes
- Check ghcr.io package page
- Verify you have correct permissions

### Helm chart not found?
- Ensure OCI registry is accessible
- Verify chart version format
- Check helm package was created

## 🎯 Next Steps

1. **Create first release**: `./hack/create-release.sh v0.1.0`
2. **Monitor workflow**: Check GitHub Actions
3. **Verify artifacts**: Test installation methods
4. **Announce release**: Update documentation
5. **Gather feedback**: Monitor issues and discussions

## 📝 Notes

- Pre-releases (alpha, beta, rc) are marked as such automatically
- The `latest` tag is only updated for stable releases
- Source archives exclude `.git`, `dist`, and `vendor` directories
- Changelog generation uses conventional commit format when available

---

**Ready to create your first release?** Run `./hack/create-release.sh v0.1.0`

For detailed information, see [docs/RELEASE.md](docs/RELEASE.md)
