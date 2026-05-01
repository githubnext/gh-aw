#!/usr/bin/env bash
set -euo pipefail

# sanitize_repo_memory_filenames.sh
# Renames files in the repo-memory directory whose names contain characters
# forbidden by GitHub Actions artifact upload (NTFS: ? : * | < > ").
# When the directory is a git working tree, tracked files are renamed with
# `git mv` so the rename is reflected in git history. Untracked files and
# files in plain (non-git) directories are renamed with `mv`.
#
# Collision handling: if the sanitized destination already exists, a numeric
# suffix is appended (e.g. "a-.md" -> "a--1.md") to avoid overwriting.
#
# Required environment variables:
#   MEMORY_DIR: Path to the repo-memory directory (git working tree or plain dir)

MEMORY_DIR="${MEMORY_DIR:?MEMORY_DIR is required}"

if [ ! -d "$MEMORY_DIR" ]; then
  echo "Memory directory not found: $MEMORY_DIR — skipping sanitization"
  exit 0
fi

cd "$MEMORY_DIR"

IS_GIT=false
if [ -d ".git" ]; then
  IS_GIT=true
fi

# sanitize_name: replace NTFS-forbidden characters in a filename with hyphen
sanitize_name() {
  printf '%s' "$1" | sed 's/[?:*|<>"]/-/g'
}

# unique_path: given a path, append a numeric suffix until it does not exist.
# Note: extension splitting uses the last dot only (e.g. "file.tar.gz" splits
# into base="file.tar" and ext=".gz"), which is intentional — we only need a
# unique name, not perfect extension preservation.
unique_path() {
  local candidate="$1"
  # Use both -e and -L to also detect broken symlinks
  if [ ! -e "$candidate" ] && [ ! -L "$candidate" ]; then
    printf '%s' "$candidate"
    return
  fi
  local base ext n
  # Split candidate into base and extension (last dot only)
  base="${candidate%.*}"
  if [ "$base" = "$candidate" ]; then
    ext=""
  else
    ext=".${candidate##*.}"
  fi
  n=1
  while [ -e "${base}-${n}${ext}" ] || [ -L "${base}-${n}${ext}" ]; do
    n=$((n + 1))
  done
  printf '%s' "${base}-${n}${ext}"
}

# do_rename: rename $1 to $2 using git mv (tracked) or mv (untracked/plain)
do_rename() {
  local src="$1" dst="$2" is_tracked="$3"
  dst=$(unique_path "$dst")
  if [ "$is_tracked" = "true" ]; then
    git mv -- "$src" "$dst"
    echo "Renamed tracked: $src -> $dst"
  else
    mv -- "$src" "$dst"
    echo "Renamed untracked: $src -> $dst"
  fi
}

if [ "$IS_GIT" = "true" ]; then
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
      do_rename "$filepath" "$newpath" true
    fi
  done < <(git ls-files --cached -z 2>/dev/null)

  # Rename untracked (new) files written by the agent that contain forbidden characters.
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
      do_rename "$filepath" "$newpath" false
    fi
  done < <(git ls-files --others -z 2>/dev/null)
else
  # Plain directory (no git): best-effort rename all files with forbidden characters.
  echo "Not a git repository — using plain mv for filename sanitization"
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
      do_rename "$filepath" "$newpath" false
    fi
  done < <(find . -depth -type f -print0 2>/dev/null)
fi

echo "Sanitization complete"
