#!/usr/bin/env bash
# Bump version: compute next version from latest git tag, update flake.nix,
# create a commit and tag. Does not push.
#
# Usage: scripts/bump-version.sh <major|minor|patch>
# Outputs the new version (without v prefix) to stdout on success.

set -euo pipefail

BUMP="${1:-}"
case "$BUMP" in
  major|minor|patch) ;;
  *)
    echo "Usage: $0 <major|minor|patch>" >&2
    exit 1
    ;;
esac

LATEST=$(git tag --list 'v*' --sort=-v:refname | head -n1)
if [[ -z "$LATEST" ]]; then
  CURRENT="0.0.0"
else
  CURRENT="${LATEST#v}"
fi

if [[ ! "$CURRENT" =~ ^([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then
  echo "Cannot parse latest version: $CURRENT" >&2
  exit 1
fi
MAJOR="${BASH_REMATCH[1]}"
MINOR="${BASH_REMATCH[2]}"
PATCH="${BASH_REMATCH[3]}"

case "$BUMP" in
  major) MAJOR=$((MAJOR + 1)); MINOR=0; PATCH=0 ;;
  minor) MINOR=$((MINOR + 1)); PATCH=0 ;;
  patch) PATCH=$((PATCH + 1)) ;;
esac

NEW="${MAJOR}.${MINOR}.${PATCH}"

if git rev-parse "v$NEW" >/dev/null 2>&1; then
  echo "Tag v$NEW already exists" >&2
  exit 1
fi

sed -i.bak -E 's/^(\s*version\s*=\s*")[^"]+(";)/\1'"$NEW"'\2/' flake.nix
rm -f flake.nix.bak
if ! grep -q "version = \"$NEW\";" flake.nix; then
  echo "Failed to update flake.nix" >&2
  exit 1
fi

git add flake.nix >&2
git commit -m "Bump version to $NEW" >&2
git tag "v$NEW" >&2

echo "$NEW"
