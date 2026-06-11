#!/bin/bash
# release.sh — Create a new release
# Usage: ./scripts/release.sh 0.1.12
set -euo pipefail

if [ $# -ne 1 ]; then
	echo "Usage: ./scripts/release.sh <version>"
	echo "Example: ./scripts/release.sh 0.1.12"
	exit 1
fi

VERSION="$1"
TAG="v${VERSION}"

# Validate version format
if ! echo "$VERSION" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$'; then
	echo "❌ Invalid version format: $VERSION (expected: x.y.z)"
	exit 1
fi

cd "$(dirname "$0")/.."

# Bump version in main.go and package.json
sed -i "s/const version = \".*\"/const version = \"$VERSION\"/" main.go
sed -i "s/\"version\": \".*\"/\"version\": \"$VERSION\"/" package.json

echo "✅ Version bumped to $VERSION"
echo "   main.go:     $(grep 'const version' main.go)"
echo "   package.json: $(grep '"version"' package.json | head -1)"

# Check if already committed
if git diff --quiet; then
	echo "⚠ No changes to commit — version already at $VERSION"
else
	git add main.go package.json
	git commit -m "chore: bump to $VERSION"
	echo "✅ Committed"
fi

# Push and tag
echo ""
echo "Ready to push. Run:"
echo "  git push origin master"
echo "  git tag $TAG"
echo "  git push origin $TAG"
