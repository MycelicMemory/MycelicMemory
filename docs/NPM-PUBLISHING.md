# Publishing ultrathink to npm

## Current Status

ultrathink is currently installable from GitHub:
```bash
npm install -g github:MycelicMemory/ultrathink
```

This document covers how to publish to the npm registry for easier installation:
```bash
npm install -g ultrathink  # Goal
```

## Prerequisites

1. **npm account**: Create at https://www.npmjs.com/signup
2. **npm CLI**: Install with Node.js
3. **Package ownership**: Verify `ultrathink` name is available or you own it

## Publishing Steps

### 1. Check Package Name Availability
```bash
npm view ultrathink
# If 404, name is available
# If shows package info, name is taken
```

### 2. Login to npm
```bash
npm login
# Enter username, password, email
# May require 2FA
```

### 3. Verify package.json
Ensure these fields are correct:
```json
{
  "name": "ultrathink",
  "version": "1.2.2",
  "description": "AI-powered persistent memory system",
  "bin": {
    "ultrathink": "bin/ultrathink"
  },
  "files": [
    "bin/ultrathink",
    "README.md",
    "LICENSE"
  ],
  "repository": {
    "type": "git",
    "url": "https://github.com/MycelicMemory/ultrathink.git"
  },
  "keywords": ["ai", "memory", "mcp", "claude", "llm"],
  "author": "Mycelic Memory",
  "license": "MIT"
}
```

### 4. Test Locally
```bash
# Pack without publishing
npm pack

# Install from tarball
npm install -g ultrathink-1.2.2.tgz

# Test
ultrathink --version
```

### 5. Publish
```bash
# Dry run first
npm publish --dry-run

# Publish
npm publish

# Or with tag
npm publish --tag latest
```

### 6. Verify Publication
```bash
npm view ultrathink
npm install -g ultrathink
ultrathink --version
```

## Automated Publishing (CI/CD)

### GitHub Actions Workflow

Create `.github/workflows/npm-publish.yml`:

```yaml
name: Publish to npm

on:
  release:
    types: [published]
  workflow_dispatch:

jobs:
  publish:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-node@v4
        with:
          node-version: '20'
          registry-url: 'https://registry.npmjs.org'

      - name: Verify package
        run: npm pack --dry-run

      - name: Publish
        run: npm publish
        env:
          NODE_AUTH_TOKEN: ${{ secrets.NPM_TOKEN }}
```

### Setup NPM Token

1. Go to npmjs.com → Account Settings → Access Tokens
2. Create new "Automation" token
3. Add to GitHub: Settings → Secrets → Actions → New secret
   - Name: `NPM_TOKEN`
   - Value: `npm_...` (your token)

## Version Management

### Bump Version
```bash
# Patch: 1.2.2 → 1.2.3
npm version patch

# Minor: 1.2.2 → 1.3.0
npm version minor

# Major: 1.2.2 → 2.0.0
npm version major
```

### Sync with Go Version
Keep `package.json` version in sync with Go binary version:
1. Update `cmd/ultrathink/root.go`: `Version = "1.2.3"`
2. Update `package.json`: `"version": "1.2.3"`
3. Commit and tag: `git tag v1.2.3`

## Deprecation

If you need to deprecate a version:
```bash
npm deprecate ultrathink@1.0.0 "Critical bug, please upgrade"
```

## Unpublishing

**Warning**: npm has strict unpublish policies

```bash
# Within 72 hours of publish
npm unpublish ultrathink@1.2.2

# After 72 hours, contact npm support
```

## Scoped Packages (Alternative)

If `ultrathink` is taken, use scoped name:
```bash
# In package.json
"name": "@mycelicmemory/ultrathink"

# Publish public scoped package
npm publish --access public

# Install
npm install -g @mycelicmemory/ultrathink
```

## Troubleshooting

### "Package name too similar to existing package"
- Use scoped name: `@org/ultrathink`
- Or contact npm support

### "403 Forbidden"
- Check npm login: `npm whoami`
- Verify token permissions
- Check 2FA settings

### "Version already exists"
- Bump version: `npm version patch`
- Or unpublish if within 72 hours

### Binary not found after install
- Verify `bin` field in package.json
- Check file permissions
- Ensure file is in `files` array

## Best Practices

1. **Always test before publish**: `npm pack` and install locally
2. **Use semantic versioning**: MAJOR.MINOR.PATCH
3. **Document changes**: Update CHANGELOG.md
4. **Tag releases**: `git tag v1.2.3`
5. **Keep README updated**: npm displays README on package page
6. **Include LICENSE**: Required for trust
