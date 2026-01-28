# MycelicMemory: Optimal GitHub Deployment Strategy

> A hyper-tailored CI/CD guide for MycelicMemory's Go + CGO + npm hybrid architecture.

---

## Executive Summary

MycelicMemory is a **Go CLI tool with CGO dependencies** (SQLite FTS5) distributed via **npm with binary downloads**. This creates unique deployment requirements:

| Aspect | MycelicMemory Reality | Strategy Impact |
|--------|----------------------|-----------------|
| Build system | Go 1.23 + CGO + SQLite FTS5 | Requires native builds per platform |
| Distribution | npm package + GitHub Releases | Dual release pipeline |
| Platforms | Linux (x64/ARM64), macOS (x64/ARM64), Windows (x64) | 5-target matrix build |
| Dependencies | Minimal Go modules | Fast CI cycles possible |

---

## Part 1: Current State Assessment

### What's Already Working Well

```
ci.yml              ✅ PR validation (build + lint)
release.yml         ✅ Multi-platform release builds
npm-publish.yml     ✅ npm publication on release
e2e-test.yml        ✅ Cross-platform E2E testing
```

### Gaps to Address

| Gap | Priority | Impact |
|-----|----------|--------|
| No security scanning | HIGH | Vulnerabilities could ship |
| No dependabot | MEDIUM | Stale dependencies |
| No coverage tracking | LOW | Quality visibility |
| No branch protection docs | MEDIUM | Process unclear |
| CI only on PRs, not push to main | MEDIUM | Main can break |

---

## Part 2: Branch Strategy

### Recommended: Trunk-Based Development

MycelicMemory should use **trunk-based development** (single `main` branch):

```
main ─────●─────●─────●─────●───> (always deployable)
           \   / \   /
            feature  hotfix
```

**Rationale:**

- Small team/solo project
- Continuous deployment model
- npm version bumps are atomic

### Branch Naming Convention

| Branch Type | Pattern | Example |
|-------------|---------|---------|
| Feature | `feature/<description>` | `feature/add-vector-search` |
| Bugfix | `fix/<description>` | `fix/memory-leak-on-search` |
| Hotfix | `hotfix/<description>` | `hotfix/npm-postinstall-fail` |
| Release prep | `release/v<version>` | `release/v1.3.0` |

### Branch Protection Rules for `main`

Apply these in **Settings > Branches > Add rule**:

```yaml
# Required settings for main branch
Branch name pattern: main

✅ Require a pull request before merging
   ✅ Required approving reviews: 1 (or 0 for solo)
   ✅ Dismiss stale reviews when new commits are pushed

✅ Require status checks to pass before merging
   Required checks:
   - build-test
   - lint
   ✅ Require branches to be up to date before merging

✅ Require linear history (enables squash merge enforcement)

❌ Allow force pushes (NEVER)
❌ Allow deletions (NEVER)
```

---

## Part 3: CI Pipeline (Enhanced)

### 3.1 Recommended CI Workflow

Replace `.github/workflows/ci.yml` with this enhanced version:

```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

env:
  GO_VERSION: '1.23'
  CGO_ENABLED: 1
  BUILD_TAGS: fts5

jobs:
  # ═══════════════════════════════════════════════════════════════
  # Stage 1: Fast Feedback (< 2 min)
  # ═══════════════════════════════════════════════════════════════

  lint:
    name: Lint
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true

      - name: golangci-lint
        uses: golangci/golangci-lint-action@v6
        with:
          version: v1.62
          args: --timeout=5m --build-tags=${{ env.BUILD_TAGS }}

  build-test:
    name: Build & Test
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true

      - name: Build
        run: go build -tags "$BUILD_TAGS" -o mycelicmemory ./cmd/mycelicmemory

      - name: Test with Race Detector
        run: go test -tags "$BUILD_TAGS" -race -coverprofile=coverage.out ./...

      - name: Upload Coverage
        uses: codecov/codecov-action@v4
        with:
          files: ./coverage.out
          fail_ci_if_error: false
        env:
          CODECOV_TOKEN: ${{ secrets.CODECOV_TOKEN }}

  # ═══════════════════════════════════════════════════════════════
  # Stage 2: Security (runs in parallel with build-test)
  # ═══════════════════════════════════════════════════════════════

  security:
    name: Security Scan
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true

      - name: Run govulncheck
        run: |
          go install golang.org/x/vuln/cmd/govulncheck@latest
          govulncheck ./...

      - name: Run gitleaks
        uses: gitleaks/gitleaks-action@v2
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}

  # ═══════════════════════════════════════════════════════════════
  # Stage 3: Platform Verification (only on main push)
  # ═══════════════════════════════════════════════════════════════

  build-matrix:
    name: Build (${{ matrix.name }})
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    needs: [lint, build-test, security]
    strategy:
      fail-fast: false
      matrix:
        include:
          - os: ubuntu-latest
            goos: linux
            goarch: amd64
            name: linux-x64
          - os: ubuntu-latest
            goos: linux
            goarch: arm64
            name: linux-arm64
            cc: aarch64-linux-gnu-gcc
          - os: macos-latest
            goos: darwin
            goarch: arm64
            name: macos-arm64
          - os: windows-latest
            goos: windows
            goarch: amd64
            name: windows-x64
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true

      - name: Install cross-compiler (Linux ARM64)
        if: matrix.name == 'linux-arm64'
        run: sudo apt-get update && sudo apt-get install -y gcc-aarch64-linux-gnu

      - name: Build
        env:
          GOOS: ${{ matrix.goos }}
          GOARCH: ${{ matrix.goarch }}
          CC: ${{ matrix.cc || '' }}
        run: go build -tags "$BUILD_TAGS" -ldflags "-s -w" -o mycelicmemory-${{ matrix.name }} ./cmd/mycelicmemory
```

### 3.2 Required Status Checks

Configure these as **required** in branch protection:

| Check Name | Purpose | Blocking? |
|------------|---------|-----------|
| `lint` | Code quality | YES |
| `build-test` | Core functionality | YES |
| `security` | Vulnerability detection | YES (recommended) |

---

## Part 4: Release Pipeline

### 4.1 Release Trigger Strategy

```
Developer tags v1.3.0
        │
        ▼
┌───────────────────┐
│   release.yml     │ ◄── Triggered by tag push v*
│   (builds all     │
│    platforms)     │
└────────┬──────────┘
         │
         ▼
┌───────────────────┐
│ GitHub Release    │ ◄── Creates release with binaries
│ created           │
└────────┬──────────┘
         │
         ▼
┌───────────────────┐
│ npm-publish.yml   │ ◄── Triggered by release.published
│ (publishes to npm)│
└───────────────────┘
```

### 4.2 Version Synchronization Checklist

Before tagging a release:

```bash
# 1. Update package.json version
npm version 1.3.0 --no-git-tag-version

# 2. Update Makefile VERSION (optional, used for local builds)
# VERSION?=1.3.0

# 3. Commit version bump
git add package.json
git commit -m "chore: bump version to 1.3.0"

# 4. Tag and push
git tag v1.3.0
git push origin main --tags
```

### 4.3 Current release.yml Assessment

Your current `release.yml` is well-structured. Recommended additions:

```yaml
# Add to release.yml after artifact upload
- name: Generate checksums
  run: |
    cd release
    sha256sum * > checksums.txt
    cat checksums.txt

- name: Create Release
  uses: softprops/action-gh-release@v2
  with:
    # ... existing config ...
    files: |
      release/*
      release/checksums.txt
    generate_release_notes: true  # Auto-generate from PRs
```

---

## Part 5: Security Configuration

### 5.1 Add `.github/dependabot.yml`

```yaml
version: 2
updates:
  # Go modules
  - package-ecosystem: "gomod"
    directory: "/"
    schedule:
      interval: "weekly"
      day: "monday"
    open-pull-requests-limit: 5
    labels:
      - "dependencies"
      - "go"
    commit-message:
      prefix: "chore(deps):"

  # GitHub Actions
  - package-ecosystem: "github-actions"
    directory: "/"
    schedule:
      interval: "weekly"
    labels:
      - "dependencies"
      - "ci"
    commit-message:
      prefix: "chore(ci):"

  # npm (for postinstall script)
  - package-ecosystem: "npm"
    directory: "/"
    schedule:
      interval: "weekly"
    labels:
      - "dependencies"
      - "npm"
```

### 5.2 Add `.golangci.yml`

```yaml
run:
  timeout: 5m
  build-tags:
    - fts5

linters:
  enable:
    - errcheck      # Check errors are handled
    - govet         # Report suspicious constructs
    - staticcheck   # Suite of static analysis
    - unused        # Find unused code
    - gosec         # Security-focused linting
    - gofmt         # Check formatting
    - goimports     # Check import ordering
    - misspell      # Catch common typos
    - ineffassign   # Detect ineffective assignments
    - typecheck     # Type checking

linters-settings:
  gosec:
    excludes:
      - G104  # Audit errors not checked (too noisy for CLI)

  govet:
    enable-all: true

issues:
  exclude-rules:
    # Exclude test files from some checks
    - path: _test\.go
      linters:
        - gosec
        - errcheck
```

### 5.3 Secret Scanning

GitHub's built-in secret scanning is automatically enabled. Additionally:

```yaml
# Add to CI workflow
- name: Gitleaks
  uses: gitleaks/gitleaks-action@v2
  env:
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

---

## Part 6: Environment & Secrets Management

### 6.1 Required Secrets

| Secret | Scope | Purpose |
|--------|-------|---------|
| `NPM_TOKEN` | Environment: `npm-publish` | npm publication |
| `CODECOV_TOKEN` | Repository | Coverage uploads |
| `GITHUB_TOKEN` | Auto-provided | Release creation, gitleaks |

### 6.2 Environment Configuration

Create environment `npm-publish` in **Settings > Environments**:

```yaml
Environment: npm-publish
Protection rules:
  ✅ Required reviewers: 0 (auto-publish) or 1 (manual approval)
  ✅ Wait timer: 0 minutes

Deployment branches:
  ✅ Selected branches: main

Environment secrets:
  NPM_TOKEN: <your npm automation token>
```

---

## Part 7: Workflow Decision Trees

### 7.1 "Which workflow runs when?"

```
Event                          Workflow(s) Triggered
─────────────────────────────────────────────────────
Push to feature/*              None (open PR first)
PR opened/updated              ci.yml, e2e-test.yml
PR merged to main              ci.yml (push trigger)
Tag pushed (v*)                release.yml
Release published              npm-publish.yml
Manual trigger                 Any workflow with workflow_dispatch
```

### 7.2 "Should this block the PR?"

```
                    ┌─────────────────────┐
                    │ Does it affect      │
                    │ production safety?  │
                    └──────────┬──────────┘
                               │
              ┌────────────────┴────────────────┐
              │                                 │
              ▼                                 ▼
        ┌─────────┐                       ┌─────────┐
        │   YES   │                       │   NO    │
        └────┬────┘                       └────┬────┘
             │                                 │
             ▼                                 ▼
    ┌─────────────────┐               ┌─────────────────┐
    │ BLOCKING CHECK  │               │ Non-blocking    │
    │ (build, lint,   │               │ annotation only │
    │  security)      │               │ (coverage, etc) │
    └─────────────────┘               └─────────────────┘
```

### 7.3 "Hotfix or Rollback?"

```
                    ┌─────────────────────┐
                    │ Production issue    │
                    │ detected            │
                    └──────────┬──────────┘
                               │
                               ▼
                    ┌─────────────────────┐
                    │ Can git revert      │
                    │ fix it cleanly?     │
                    └──────────┬──────────┘
                               │
              ┌────────────────┴────────────────┐
              │                                 │
              ▼                                 ▼
        ┌─────────┐                       ┌─────────┐
        │   YES   │                       │   NO    │
        └────┬────┘                       └────┬────┘
             │                                 │
             ▼                                 ▼
    ┌─────────────────┐               ┌─────────────────┐
    │ 1. git revert   │               │ 1. hotfix/branch│
    │ 2. Push PR      │               │ 2. Minimal fix  │
    │ 3. Fast merge   │               │ 3. Expedited PR │
    │ 4. Tag patch    │               │ 4. Tag patch    │
    │    v1.2.3       │               │    v1.2.3       │
    └─────────────────┘               └─────────────────┘
```

---

## Part 8: Monitoring & Notifications

### 8.1 Workflow Failure Notifications

Add to any critical workflow:

```yaml
- name: Notify on Failure
  if: failure() && github.ref == 'refs/heads/main'
  uses: slackapi/slack-github-action@v1
  with:
    channel-id: 'YOUR_CHANNEL_ID'
    slack-message: |
      :x: *${{ github.workflow }}* failed on `main`
      <${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}|View Run>
  env:
    SLACK_BOT_TOKEN: ${{ secrets.SLACK_BOT_TOKEN }}
```

### 8.2 Release Announcement (Optional)

Add to `npm-publish.yml`:

```yaml
- name: Announce Release
  if: success()
  uses: slackapi/slack-github-action@v1
  with:
    channel-id: 'YOUR_CHANNEL_ID'
    slack-message: |
      :rocket: *MycelicMemory ${{ needs.verify.outputs.version }}* published!
      ```
      npm install -g mycelicmemory
      ```
  env:
    SLACK_BOT_TOKEN: ${{ secrets.SLACK_BOT_TOKEN }}
```

---

## Part 9: Quick Reference

### 9.1 Common Operations

| Task | Command |
|------|---------|
| Create feature branch | `git checkout -b feature/my-feature` |
| Push for CI | `git push -u origin feature/my-feature` |
| Prepare release | `npm version 1.3.0 --no-git-tag-version` |
| Tag release | `git tag v1.3.0 && git push origin v1.3.0` |
| Manual npm publish | Go to Actions > npm-publish > Run workflow |
| View CI status | `gh run list` or GitHub Actions tab |

### 9.2 Troubleshooting

| Symptom | Likely Cause | Fix |
|---------|--------------|-----|
| CI fails on ARM64 build | Missing cross-compiler | Check `gcc-aarch64-linux-gnu` install step |
| npm publish fails | Version exists | Bump version in package.json |
| E2E tests flaky | Server startup race | Increase wait time or add health check retry |
| Release missing binaries | Matrix job failed | Check individual platform build logs |

### 9.3 Files to Create/Update

| File | Action | Priority |
|------|--------|----------|
| `.github/dependabot.yml` | CREATE | HIGH |
| `.golangci.yml` | CREATE | MEDIUM |
| `.github/workflows/ci.yml` | UPDATE | HIGH |
| `.github/CODEOWNERS` | CREATE | LOW |

---

## Appendix A: Complete File Templates

### A.1 `.github/CODEOWNERS`

```
# Default owner
* @your-github-username

# CI/CD files
.github/ @your-github-username

# Core code
/cmd/ @your-github-username
/internal/ @your-github-username

# Documentation
/docs/ @your-github-username
README.md @your-github-username
```

### A.2 Recommended Repository Settings

```
Settings > General:
  ✅ Automatically delete head branches (after PR merge)

Settings > Actions > General:
  ✅ Allow GitHub Actions to create and approve pull requests

Settings > Code security:
  ✅ Dependabot alerts: Enabled
  ✅ Dependabot security updates: Enabled
  ✅ Secret scanning: Enabled
```

---

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0.0 | 2025-01-27 | Initial tailored strategy for MycelicMemory |
