# GitHub Actions Workflow Documentation

## Overview

This repository uses GitHub Actions for automated data scraping, analysis, and deployment. The automation consists of two main workflows that work together:

1. **update-exams.yml**: Scrapes exam data, creates PRs, and auto-merges changes
2. **deploy-pages.yml**: Deploys analysis reports to GitHub Pages

## Workflow Architecture

### 1. Update Exam Data Workflow

**Schedule**: Every 2 hours (via cron)

**Flow**:
```
Scrape Data → Check for Changes → Create PR → Auto-merge → Trigger Deployment
```

**Jobs**:
- `update-data`: Scrapes website, compares data, creates PR if changes detected
- `analyze`: Generates R-based analysis reports (runs in parallel)
- `notify`: Sends Pushover notifications for Freising appointments (runs in parallel)

### 2. Deploy to GitHub Pages Workflow

**Triggers**:
- Push to `main` branch
- Manual workflow dispatch
- **Explicit trigger from update-exams workflow**

**Flow**:
```
Checkout → Render Quarto → Upload Artifact → Deploy to Pages
```

## Critical Issue: Auto-merge and Workflow Triggers

### The Problem

When GitHub Actions auto-merges a pull request, there's a known limitation with workflow triggers:

**Workflows triggered by actions using `GITHUB_TOKEN` do not trigger subsequent workflows.**

This is a security feature to prevent recursive workflow runs. However, it creates a problem:

```
Auto-merge PR (using GITHUB_TOKEN) → Push to main → ❌ Deploy workflow NOT triggered
Manual merge PR (by human)         → Push to main → ✅ Deploy workflow triggered
```

### The Solution

We implemented an **explicit workflow trigger** in the update-exams workflow:

**File**: `.github/workflows/update-exams.yml` (lines 143-159)

```yaml
- name: Wait for merge and trigger deployment
  if: steps.create_pr.outputs.pull-request-number
  env:
    GH_TOKEN: ${{ secrets.PAT_TOKEN || secrets.GITHUB_TOKEN }}
  run: |
    # Wait a bit for the auto-merge to complete
    sleep 10

    # Check if PR was merged
    PR_STATE=$(gh pr view ${{ steps.create_pr.outputs.pull-request-number }} --json state --jq '.state')

    if [ "$PR_STATE" = "MERGED" ]; then
      echo "PR merged successfully, triggering deploy workflow"
      gh workflow run deploy-pages.yml
    else
      echo "PR not yet merged (state: $PR_STATE), deployment will trigger on push to main"
    fi
```

**How it works**:

1. After enabling auto-merge, the workflow waits 10 seconds
2. Checks the PR state using `gh pr view`
3. If merged, explicitly triggers `deploy-pages.yml` using `gh workflow run`
4. If not yet merged, relies on the standard push trigger as fallback

**Benefits**:

- Works with both `GITHUB_TOKEN` and `PAT_TOKEN`
- Ensures deployment happens automatically
- Has fallback mechanism if auto-merge is delayed
- No manual intervention required

## Token Configuration

### GITHUB_TOKEN (default)

- Automatically provided by GitHub Actions
- Has limited permissions by default
- **Limitation**: Actions using this token won't trigger other workflows from push events

### PAT_TOKEN (optional)

If you want more reliable workflow triggers, configure a Personal Access Token:

**Required scopes**:
- `repo` (full repository access)
- `workflow` (trigger workflows)

**Setup**:
1. Generate PAT at: Settings → Developer settings → Personal access tokens
2. Add as repository secret: Settings → Secrets and variables → Actions
3. Name it `PAT_TOKEN`

**Usage in workflows**:
```yaml
env:
  GH_TOKEN: ${{ secrets.PAT_TOKEN || secrets.GITHUB_TOKEN }}
```

The `||` operator provides fallback to `GITHUB_TOKEN` if `PAT_TOKEN` is not configured.

## Permissions

### update-exams.yml permissions

```yaml
permissions:
  contents: write       # Create commits and push to branches
  pull-requests: write  # Create and manage pull requests
```

### deploy-pages.yml permissions

```yaml
permissions:
  contents: read    # Read repository contents
  pages: write      # Deploy to GitHub Pages
  id-token: write   # Required for Pages deployment
```

## Troubleshooting

### Deployment not triggered after auto-merge

**Symptoms**: PR auto-merges successfully but GitHub Pages doesn't update

**Check**:
1. View the update-exams workflow run logs
2. Look for "Wait for merge and trigger deployment" step
3. Check if it shows "PR merged successfully, triggering deploy workflow"

**Solutions**:
- Ensure the step ran without errors
- Verify deploy-pages workflow was triggered (check Actions tab)
- Manually trigger deployment: `gh workflow run deploy-pages.yml`

### Permission errors

**Error**: `refusing to allow an OAuth App to create or update workflow`

**Solution**: Use a Personal Access Token (PAT_TOKEN) with `workflow` scope instead of GITHUB_TOKEN

### Auto-merge not working

**Symptoms**: PR created but not auto-merged

**Common causes**:
- Branch protection rules require reviews
- Required status checks not passing
- Insufficient token permissions

**Check**:
- Repository Settings → Branches → Branch protection rules
- Ensure GITHUB_TOKEN or PAT_TOKEN has sufficient permissions

## Manual Operations

### Manually trigger data update
```bash
gh workflow run update-exams.yml
```

### Manually trigger deployment
```bash
gh workflow run deploy-pages.yml
```

### View workflow status
```bash
gh run list --workflow=update-exams.yml
gh run list --workflow=deploy-pages.yml
```

### View specific run details
```bash
gh run view <run-id>
```

## Best Practices

1. **Monitor workflow runs**: Regularly check the Actions tab for failures
2. **Review auto-merged PRs**: Periodically review merged PRs to ensure data quality
3. **Test changes**: Use `workflow_dispatch` to manually test workflow changes
4. **Keep dependencies updated**: Update action versions and Go dependencies regularly
5. **Document changes**: Update this file when modifying workflow behavior

## References

- [GitHub Actions: Triggering a workflow](https://docs.github.com/en/actions/using-workflows/triggering-a-workflow)
- [GitHub Actions: Permissions](https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions#permissions)
- [GitHub Actions: Using GITHUB_TOKEN](https://docs.github.com/en/actions/security-guides/automatic-token-authentication)
- [GitHub Pages deployment action](https://github.com/actions/deploy-pages)
