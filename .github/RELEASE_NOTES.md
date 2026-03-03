# Release Automation

This repository uses GitHub Actions to automate the release process.

## Release Workflow

The release workflow (`.github/workflows/release.yml`) is triggered when a version tag is pushed to the repository.

### What Gets Built and Published

1. **Docker Images** (Multi-architecture: amd64, arm64)
   - Registry: `ghcr.io/nusnewob/kube-changejob`
   - Tags: `vX.Y.Z`, `X.Y`, `X`, `latest`

2. **Helm Chart** (OCI format)
   - Registry: `oci://ghcr.io/nusnewob/charts/kube-changejob`
   - Version: `X.Y.Z`

3. **Kubernetes Manifests**
   - `crds.yaml` - CRDs only
   - `install.yaml` - Full installation (CRDs + controller + RBAC + webhooks)

4. **Source Archives**
   - `kube-changejob-X.Y.Z.tar.gz` (Linux/macOS)
   - `kube-changejob-X.Y.Z.zip` (Windows)

5. **SHA256 Checksums**
   - `checksums.txt` - SHA256 checksums for all release artifacts

6. **Changelog**
   - Auto-generated from git commits since last release
   - Includes installation instructions
   - Links to container images and helm charts

## Creating a Release

### Quick Method

```bash
./hack/create-release.sh v0.1.0
```

### Manual Method

1. Update versions:
   ```bash
   # Edit dist/chart/Chart.yaml
   version: 0.1.0
   appVersion: "0.1.0"
   ```

2. Commit and push:
   ```bash
   git add dist/chart/Chart.yaml
   git commit -m "chore: bump version to v0.1.0"
   git push origin main
   ```

3. Create and push tag:
   ```bash
   git tag -a v0.1.0 -m "Release v0.1.0"
   git push origin v0.1.0
   ```

## Pre-Release Validation

The pre-release workflow (`.github/workflows/pre-release.yml`) runs on every push to main and validates:

- ✅ All tests pass
- ✅ Binary builds successfully
- ✅ Docker image builds
- ✅ Manifests generate correctly
- ✅ Helm chart is valid
- ✅ Security scans pass
- ✅ Shows release readiness status

## Release Types

- **Stable**: `v1.0.0` - Production ready
- **Alpha**: `v1.0.0-alpha.1` - Early testing
- **Beta**: `v1.0.0-beta.1` - Feature complete, testing
- **RC**: `v1.0.0-rc.1` - Release candidate

Pre-releases are automatically marked in GitHub.

## Post-Release

After a successful release:

1. GitHub Release is created with changelog
2. Docker images are available at `ghcr.io/nusnewob/kube-changejob`
3. Helm chart is available at `oci://ghcr.io/nusnewob/charts/kube-changejob`
4. Documentation is automatically updated
5. Release announcement PR is created

## Verification

```bash
# Verify Docker image
docker pull ghcr.io/nusnewob/kube-changejob:v0.1.0

# Verify Helm chart
helm pull oci://ghcr.io/nusnewob/charts/kube-changejob --version 0.1.0

# Verify manifests
kubectl apply --dry-run=client -f https://github.com/nusnewob/kube-changejob/releases/download/v0.1.0/install.yaml

# Verify checksums
curl -sL https://github.com/nusnewob/kube-changejob/releases/download/v0.1.0/checksums.txt
```

## Troubleshooting

### Workflow Failed

1. Check Actions tab for detailed logs
2. Common issues:
   - Docker build errors
   - Manifest generation errors
   - Helm packaging errors
   - Network issues

### Need to Recreate Tag

⚠️ Only for unreleased or early tags:

```bash
# Delete tag locally and remotely
git tag -d v0.1.0
git push origin :refs/tags/v0.1.0

# Recreate and push
git tag -a v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0
```

## Documentation

- [Full Release Process](../docs/RELEASE.md)
- [Contributing Guidelines](../CONTRIBUTING.md)
- [Changelog](../CHANGELOG.md)
