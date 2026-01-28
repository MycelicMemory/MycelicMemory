#!/bin/bash
set -e

# Semantic versioning helper script
# Automatically determines the next version based on conventional commits

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Get current version from package.json
CURRENT_VERSION=$(node -p "require('./package.json').version")
echo -e "${GREEN}Current version: ${CURRENT_VERSION}${NC}"

# Get latest tag
LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
echo -e "${GREEN}Latest tag: ${LATEST_TAG}${NC}"

# Get commits since last tag
echo ""
echo -e "${YELLOW}Commits since ${LATEST_TAG}:${NC}"
git log ${LATEST_TAG}..HEAD --pretty=format:"%h %s" 2>/dev/null || git log --pretty=format:"%h %s" | head -10

# Analyze commits
COMMITS=$(git log ${LATEST_TAG}..HEAD --pretty=format:"%s" 2>/dev/null || git log --pretty=format:"%s")

# Determine bump type
if echo "$COMMITS" | grep -qE "(BREAKING CHANGE|^[a-z]+!:|^[a-z]+\([^)]+\)!:)"; then
  BUMP_TYPE="major"
  echo ""
  echo -e "${RED}⚠️  Breaking changes detected${NC}"
elif echo "$COMMITS" | grep -qE "^feat(\([^)]+\))?:"; then
  BUMP_TYPE="minor"
  echo ""
  echo -e "${GREEN}✨ New features detected${NC}"
else
  BUMP_TYPE="patch"
  echo ""
  echo -e "${GREEN}🐛 Patch changes detected${NC}"
fi

# Parse current version
IFS='.' read -r MAJOR MINOR PATCH <<< "${CURRENT_VERSION}"

# Calculate new version
case "$BUMP_TYPE" in
  major)
    MAJOR=$((MAJOR + 1))
    MINOR=0
    PATCH=0
    ;;
  minor)
    MINOR=$((MINOR + 1))
    PATCH=0
    ;;
  patch)
    PATCH=$((PATCH + 1))
    ;;
esac

NEW_VERSION="${MAJOR}.${MINOR}.${PATCH}"

echo ""
echo -e "${GREEN}Recommended version bump: ${BUMP_TYPE}${NC}"
echo -e "${GREEN}New version: ${NEW_VERSION}${NC}"
echo ""

# Ask for confirmation
read -p "Do you want to update version to ${NEW_VERSION}? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
  echo "Aborted."
  exit 0
fi

# Update package.json
node -e "
  const fs = require('fs');
  const pkg = JSON.parse(fs.readFileSync('package.json', 'utf8'));
  pkg.version = '${NEW_VERSION}';
  fs.writeFileSync('package.json', JSON.stringify(pkg, null, 2) + '\n');
"

echo -e "${GREEN}✅ Updated package.json to version ${NEW_VERSION}${NC}"
echo ""
echo "Next steps:"
echo "  1. Review the changes: git diff package.json"
echo "  2. Commit the version bump: git add package.json && git commit -m 'chore(release): bump version to ${NEW_VERSION}'"
echo "  3. Create a tag: git tag -a v${NEW_VERSION} -m 'Release v${NEW_VERSION}'"
echo "  4. Push changes: git push origin main --tags"
