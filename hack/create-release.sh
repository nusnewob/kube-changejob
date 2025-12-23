#!/usr/bin/env bash

# Script to help create a new release
# Usage: ./hack/create-release.sh <version>
# Example: ./hack/create-release.sh v0.1.0

set -e

VERSION=$1

if [ -z "$VERSION" ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 v0.1.0"
    exit 1
fi

# Ensure version starts with 'v'
if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+.*$ ]]; then
    echo "Error: Version must be in format v0.0.0 (e.g., v0.1.0, v1.0.0-alpha.1)"
    exit 1
fi

VERSION_NO_V=${VERSION#v}

echo "🚀 Preparing release $VERSION"
echo ""

# Check if git is clean
if [ -n "$(git status --porcelain)" ]; then
    echo "❌ Error: Working directory is not clean. Please commit or stash changes."
    git status --short
    exit 1
fi

# Check if on main branch
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [ "$CURRENT_BRANCH" != "main" ]; then
    echo "⚠️  Warning: Not on main branch (currently on $CURRENT_BRANCH)"
    read -p "Do you want to continue? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Update to latest
echo "📥 Fetching latest changes..."
git fetch --all --tags

# Check if tag already exists
if git rev-parse "$VERSION" >/dev/null 2>&1; then
    echo "❌ Error: Tag $VERSION already exists"
    exit 1
fi

# Update helm chart version
echo "📝 Updating Helm chart version..."
sed -i.bak "s/^version:.*/version: $VERSION_NO_V/" dist/chart/Chart.yaml
sed -i.bak "s/^appVersion:.*/appVersion: \"$VERSION_NO_V\"/" dist/chart/Chart.yaml
rm -f dist/chart/Chart.yaml.bak

# Show what will be released
echo ""
echo "📋 Release Summary:"
echo "  Version: $VERSION"
echo "  Branch: $CURRENT_BRANCH"
echo "  Commit: $(git rev-parse --short HEAD)"
echo ""

# Show commits since last tag
LAST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
if [ -n "$LAST_TAG" ]; then
    echo "📝 Changes since $LAST_TAG:"
    git log --pretty=format:"  - %s (%h)" --no-merges "$LAST_TAG..HEAD"
    echo ""
else
    echo "📝 This will be the first release"
fi
echo ""

# Confirm
read -p "Create release $VERSION? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "❌ Release cancelled"
    git checkout dist/chart/Chart.yaml 2>/dev/null || true
    exit 1
fi

# Commit version updates if changed
if [ -n "$(git status --porcelain dist/chart/Chart.yaml)" ]; then
    echo "💾 Committing version updates..."
    git add dist/chart/Chart.yaml
    git commit -m "chore: bump version to $VERSION"
    git push origin "$CURRENT_BRANCH"
fi

# Create and push tag
echo "🏷️  Creating and pushing tag..."
git tag -a "$VERSION" -m "Release $VERSION"
git push origin "$VERSION"

echo ""
echo "✅ Release tag $VERSION created and pushed!"
echo ""
echo "🎯 Next steps:"
echo "  1. GitHub Actions will automatically build and publish the release"
echo "  2. Monitor the workflow at: https://github.com/$(git remote get-url origin | sed 's/.*github.com[:/]\(.*\)\.git/\1/')/actions"
echo "  3. Once complete, the release will be available at: https://github.com/$(git remote get-url origin | sed 's/.*github.com[:/]\(.*\)\.git/\1/')/releases/tag/$VERSION"
echo ""
