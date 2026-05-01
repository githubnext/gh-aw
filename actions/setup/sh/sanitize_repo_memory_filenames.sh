#!/usr/bin/env bash
set -euo pipefail

# sanitize_repo_memory_filenames.sh
# Renames files in the repo-memory git working tree whose names contain
# characters forbidden by GitHub Actions artifact upload (NTFS: ? : * | < > ").
# Tracked files are renamed with `git mv` so the rename is reflected in git
# history. New (untracked) files written by the agent are renamed with `mv`.
#
# Required environment variables:
#   MEMORY_DIR: Path to the repo-memory git working tree

MEMORY_DIR="${MEMORY_DIR:?MEMORY_DIR is required}"

if [ ! -d "$MEMORY_DIR" ]; then
  echo "Memory directory not found: $MEMORY_DIR — skipping sanitization"
  exit 0
fi

cd "$MEMORY_DIR"

if [ ! -d ".git" ]; then
  echo "Not a git repository: $MEMORY_DIR — skipping sanitization"
  exit 0
fi

# sanitize_name: replace NTFS-forbidden characters in a filename with hyphen
sanitize_name() {
  printf '%s' "$1" | sed 's/[?:*|<>"]/-/g'
}

# Rename tracked files (in git index) that contain forbidden characters.
# git mv handles both the working-tree rename and the index update atomically.
while IFS= read -r -d '' filepath; do
  base=$(basename "$filepath")
  safe=$(sanitize_name "$base")
  if [ "$base" != "$safe" ]; then
    dir=$(dirname "$filepath")
    if [ "$dir" = "." ]; then
      newpath="$safe"
    else
      newpath="$dir/$safe"
    fi
    git mv -- "$filepath" "$newpath"
    echo "Renamed tracked: $filepath -> $newpath"
  fi
done < <(git ls-files --cached -z 2>/dev/null)

# Rename untracked (new) files written by the agent that contain forbidden characters.
# These are not yet in the git index so plain mv is used.
while IFS= read -r -d '' filepath; do
  base=$(basename "$filepath")
  safe=$(sanitize_name "$base")
  if [ "$base" != "$safe" ]; then
    dir=$(dirname "$filepath")
    if [ "$dir" = "." ]; then
      newpath="$safe"
    else
      newpath="$dir/$safe"
    fi
    mv -- "$filepath" "$newpath"
    echo "Renamed untracked: $filepath -> $newpath"
  fi
done < <(git ls-files --others -z 2>/dev/null)

echo "Sanitization complete"
