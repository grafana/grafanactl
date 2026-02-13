#!/usr/bin/env bash
# Release automation script using svu for semantic versioning
# Usage: scripts/release.sh [patch|minor|major|next]

set -euo pipefail

BUMP_TYPE="${1:-next}"

# Calculate versions
CURRENT=$(svu current)
case "$BUMP_TYPE" in
  patch) NEXT=$(svu patch) ;;
  minor) NEXT=$(svu minor) ;;
  major) NEXT=$(svu major) ;;
  next)  NEXT=$(svu next)  ;;
  *)     echo "Usage: $0 [patch|minor|major|next]"; exit 1 ;;
esac

echo "Current version: $CURRENT"
echo "Next version:    $NEXT"
echo ""
echo "Commits since $CURRENT:"
git log "${CURRENT}..HEAD" --oneline
echo ""

# Pre-release checks
echo "Running pre-release checks (make all)..."
make all

# Confirm
read -p "Tag and push $NEXT? [y/N] " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
  echo "Aborted."
  exit 1
fi

# Tag and push
git tag "$NEXT"
git push origin "$NEXT"

# Get repo info for URL
REPO=$(gh repo view --json nameWithOwner -q .nameWithOwner)
echo ""
echo "Release triggered! Monitor at:"
echo "  https://github.com/$REPO/actions/workflows/release.yaml"
echo ""
echo "Release will be published at:"
echo "  https://github.com/$REPO/releases/tag/$NEXT"
