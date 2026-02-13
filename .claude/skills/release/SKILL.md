---
name: release
description: Guide releasing a new grafanactl version. Use when user wants to create a release, tag a version, or asks about the release process.
allowed-tools: [Bash, Read]
---

# Release Skill

## Overview

Automates the grafanactl release process using `scripts/release.sh` and `svu` for semantic versioning.

## When to Use

- User asks to "release a new version" or "create a release"
- User says "tag v0.x.y" or "bump version"
- User mentions "publish a release"
- Questions about the release process

## Release Workflow

### Step 1: Show Current State

Check the current version and what versions are available:

```bash
# Run in parallel
svu current
svu next
svu patch
svu minor
svu major
```

Show commits since the last tag:
```bash
git log $(svu current)..HEAD --oneline
```

### Step 2: Ask User for Bump Type

Present the options to the user:
- **`next`** - Auto-detect based on conventional commits (recommended)
- **`patch`** - Bug fixes only (0.1.8 → 0.1.9)
- **`minor`** - New features (0.1.8 → 0.2.0)
- **`major`** - Breaking changes (0.1.8 → 1.0.0)

### Step 3: Run Release Script

The `scripts/release.sh` script handles everything:

```bash
scripts/release.sh <bump-type>
```

The script will:
1. Calculate the next version using `svu`
2. Show commits since last tag
3. Run `make all` (lint, tests, build, docs)
4. Ask for confirmation
5. Create and push the git tag
6. Print GitHub Actions workflow URL

**Note**: The script is interactive and requires user confirmation before pushing the tag.

### Step 4: Monitor Release

After the tag is pushed, monitor the GitHub Actions workflow:

```bash
# Check latest release workflow run
gh run list --workflow=release.yaml --limit=1

# Watch the run in progress (optional)
gh run watch
```

The workflow performs:
- GoReleaser builds (linux/darwin/windows, multiple architectures)
- Documentation build
- GitHub Pages deployment
- Changelog generation (auto-excludes `docs:`, `test:`, `chore:` commits)

### Step 5: Verify Release

Once the workflow completes:

```bash
# View the release
gh release view <version>

# Get release URL
echo "https://github.com/$(gh repo view --json nameWithOwner -q .nameWithOwner)/releases/tag/<version>"
```

## Pre-Release Checklist

Before running the release script, ensure:
- ✅ All changes committed and pushed to main
- ✅ CI passing on main branch
- ✅ No uncommitted changes: `git status`
- ✅ Up to date with remote: `git pull`
- ✅ All PRs for this release are merged

The release script will also run `make all` which includes:
- Linting (`make lint`)
- Tests (`make tests`)
- Build (`make build`)
- Documentation generation (`make docs`)
- Documentation drift check (`make reference-drift`)

## Examples

### Example 1: Auto-Detect Version Bump

```bash
# Show what version will be tagged
svu next

# Run release with auto-detection
scripts/release.sh next
```

### Example 2: Force Patch Release

```bash
# Force a patch bump
scripts/release.sh patch
```

### Example 3: Check Without Releasing

```bash
# Show current and next versions
svu current
svu next

# Review commits
git log $(svu current)..HEAD --oneline
```

## Version Calculation (svu)

The `svu` tool uses conventional commits to determine version bumps:

| Commit Prefix | Version Bump | Example |
|---------------|--------------|---------|
| `fix:` | Patch | 0.1.8 → 0.1.9 |
| `feat:` | Minor | 0.1.8 → 0.2.0 |
| `fix!:` or `feat!:` | Major | 0.1.8 → 1.0.0 |
| `BREAKING CHANGE:` footer | Major | 0.1.8 → 1.0.0 |
| `chore:` | None | No bump |

## Troubleshooting

### Issue: "svu: command not found"

**Solution**: Install svu from https://github.com/caarlos0/svu or via:
```bash
go install github.com/caarlos0/svu@latest
```

### Issue: Release Script Fails on `make all`

**Solution**: Fix the reported issues first:
- Linter errors: Run `make lint` and fix
- Test failures: Run `make tests` and fix
- Docs drift: Run `make docs` to regenerate

### Issue: Tag Already Exists

**Solution**:
1. Check existing tags: `git tag`
2. Delete local tag if needed: `git tag -d <version>`
3. Force-delete remote tag (CAUTION): `git push origin :refs/tags/<version>`

### Issue: GitHub Actions Workflow Fails

**Solution**:
1. View workflow logs: `gh run view <run-id>`
2. Check for common issues:
   - GoReleaser configuration errors
   - Build failures
   - Documentation build errors
3. Fix and re-run if possible, or create a patch release

## Related Files

- `scripts/release.sh` - Release automation script
- `.goreleaser.yaml` - GoReleaser configuration
- `.github/workflows/release.yaml` - Release workflow
- `AGENTS.md` - Release process documentation

## Best Practices

### Do
- ✅ Use `next` for automatic version detection
- ✅ Review commits before releasing
- ✅ Wait for CI to pass before releasing
- ✅ Monitor the release workflow after pushing tag
- ✅ Verify the GitHub release page after completion

### Don't
- ❌ Release with uncommitted changes
- ❌ Skip the pre-release checks (`make all`)
- ❌ Force-push release tags
- ❌ Delete released tags (breaks users)
- ❌ Release from non-main branches
