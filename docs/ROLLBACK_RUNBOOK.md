# Release Rollback Runbook

This runbook provides step-by-step procedures for rolling back a problematic release.

## When to Rollback

Rollback should be considered when:

- ✅ **Critical bug** affecting core functionality
- ✅ **Security vulnerability** discovered post-release
- ✅ **Data corruption** or loss risk
- ✅ **Breaking changes** not caught in testing
- ✅ **Performance regression** beyond acceptable thresholds

**Do NOT rollback for:**
- ❌ Minor UI issues
- ❌ Non-critical bugs with workarounds
- ❌ Feature requests
- ❌ Cosmetic issues

## Rollback Procedure

### 1. Assess the Situation

```bash
# Check current versions across channels
npm view mycelicmemory@latest version
npm view mycelicmemory@beta version
npm view mycelicmemory@canary version

# Check download counts
npm view mycelicmemory
```

### 2. Identify Restore Point

Determine the last known good version:

```bash
# List recent versions
npm view mycelicmemory versions --json | tail -10

# Check release notes for stable versions
gh release list --limit 5
```

### 3. Execute Automated Rollback

**Option A: Using GitHub Actions (Recommended)**

1. Go to [Actions](../../actions/workflows/rollback-release.yml)
2. Click "Run workflow"
3. Fill in the form:
   - **Rollback Version:** The problematic version (e.g., `1.2.3`)
   - **Restore Version:** The last known good version (e.g., `1.2.2`)
   - **Channel:** The affected channel (`latest`, `beta`, or `canary`)
   - **Reason:** Brief description of the issue

4. Click "Run workflow"
5. Monitor the workflow progress
6. Verify rollback completion

**Option B: Manual Rollback**

If the automated workflow fails:

```bash
# 1. Authenticate with npm
npm login

# 2. Rollback the channel tag
npm dist-tag add mycelicmemory@<RESTORE_VERSION> <CHANNEL>

# Example:
npm dist-tag add mycelicmemory@1.2.2 latest

# 3. Verify the rollback
npm view mycelicmemory@latest version
# Should show: 1.2.2

# 4. Deprecate the problematic version
npm deprecate mycelicmemory@<ROLLBACK_VERSION> "⚠️ This version has issues. Please upgrade to the latest."

# Example:
npm deprecate mycelicmemory@1.2.3 "⚠️ Critical bug in memory persistence. Please upgrade to latest."
```

### 4. Verify Rollback

```bash
# Check that the channel points to the correct version
npm view mycelicmemory@latest version

# Test installation
npm install -g mycelicmemory@latest
mycelicmemory --version

# Run basic smoke tests
mycelicmemory doctor
mycelicmemory remember "test message"
mycelicmemory search test
```

### 5. Communicate

**Immediate Communication:**

1. Update the GitHub issue created by the rollback workflow
2. Post in Discord/Slack channels (if applicable)
3. Update status page (if applicable)

**Template for User Communication:**

```
🔄 Release Rollback Notice

We've identified an issue with MycelicMemory v<ROLLBACK_VERSION> and have
rolled back to v<RESTORE_VERSION>.

**Issue:** <Brief description>
**Action Required:** Run `npm install -g mycelicmemory@latest` to get the stable version

**Impact:** <Describe user impact>
**Timeline:** <When was it detected, when was it rolled back>

We apologize for any inconvenience. A fix is being developed.
```

### 6. Post-Rollback Actions

- [ ] Create incident report (auto-generated in `incidents/` directory)
- [ ] Schedule post-mortem meeting
- [ ] Identify root cause
- [ ] Update tests to catch the issue
- [ ] Implement fix
- [ ] Test fix thoroughly
- [ ] Document lessons learned

## Rollback Decision Matrix

| Severity | User Impact | Action | Channel | Timeline |
|----------|-------------|--------|---------|----------|
| Critical | >50% users | Immediate rollback | `latest` | <1 hour |
| High | 10-50% users | Scheduled rollback | `latest` | <4 hours |
| Medium | <10% users | Promote hotfix to `beta` | `beta` | <24 hours |
| Low | Minimal | Fix in next release | - | Next sprint |

## Testing Before Rollback

Before executing a rollback, verify the restore version works:

```bash
# Install the restore version
npm install -g mycelicmemory@<RESTORE_VERSION>

# Run comprehensive tests
mycelicmemory doctor
mycelicmemory remember "test1"
mycelicmemory search test1
mycelicmemory --mcp < test-mcp-commands.json

# Check for known issues in that version
gh issue list --milestone "v<RESTORE_VERSION>"
```

## Rollback Metrics to Track

After rollback, monitor:

- npm download rates for both versions
- Error rates in telemetry (if available)
- User reports and issues
- CI/CD pipeline health

## Emergency Contacts

- **Release Manager:** @username
- **Technical Lead:** @username
- **On-Call Engineer:** Check PagerDuty/OpsGenie

## Rollback History

All rollbacks are documented in the `incidents/` directory:

```bash
ls -la incidents/
cat incidents/rollback-<version>.md
```

## Preventing Future Rollbacks

After each rollback, update:

1. **Pre-release checklist** - Add checks to catch similar issues
2. **Validation pipeline** - Enhance automated tests
3. **Staging process** - Improve canary/beta testing
4. **Documentation** - Document edge cases and gotchas

## Automated Safeguards

The following safeguards are in place:

- ✅ Pre-release validation pipeline (runs tests, benchmarks, smoke tests)
- ✅ Progressive rollout (canary → beta → latest)
- ✅ Automated metrics monitoring
- ✅ Deprecation of rolled-back versions
- ✅ Incident tracking and reporting

## References

- [Release Process](./RELEASE_PROCESS.md)
- [Deployment Pipeline](./DEPLOYMENT.md)
- [Incident Response](./INCIDENT_RESPONSE.md)
