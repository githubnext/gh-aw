#!/usr/bin/env bash
# Download Docker images with retry logic and concurrent downloads
# Usage: download_docker_images.sh IMAGE1 [IMAGE2 ...]
#
# This script downloads multiple Docker images in parallel to improve performance.
# Docker daemon supports concurrent pulls from multiple clients, which can provide
# significant speedup when downloading multiple images (e.g., 4x faster for 3 images).
#
# Each image is pulled in a background process with retry logic (3 attempts with
# exponential backoff). The script waits for all downloads to complete and fails
# if any image fails to download after all retry attempts.

set -e

# Helper function to pull Docker images with retry logic
docker_pull_with_retry() {
  local image="$1"
  local max_attempts=3
  local attempt=1
  local wait_time=5
  
  while [ $attempt -le $max_attempts ]; do
    echo "Attempt $attempt of $max_attempts: Pulling $image..."
    if docker pull --quiet "$image" 2>&1; then
      echo "Successfully pulled $image"
      return 0
    fi
    
    if [ $attempt -lt $max_attempts ]; then
      echo "Failed to pull $image. Retrying in ${wait_time}s..."
      sleep $wait_time
      wait_time=$((wait_time * 2))  # Exponential backoff
    else
      echo "Failed to pull $image after $max_attempts attempts"
      return 1
    fi
    attempt=$((attempt + 1))
  done
}

# Track background processes and their associated images
declare -a pids=()
declare -a images_list=()

# Start all downloads in parallel
echo "Starting concurrent download of ${#@} image(s)..."
for image in "$@"; do
  docker_pull_with_retry "$image" &
  pids+=($!)
  images_list+=("$image")
done

# Wait for all downloads and track failures
failed=0
for i in "${!pids[@]}"; do
  pid="${pids[$i]}"
  image="${images_list[$i]}"
  if ! wait "$pid"; then
    echo "ERROR: Failed to download $image"
    failed=1
  fi
done

# Exit with error if any download failed
if [ $failed -eq 1 ]; then
  echo "One or more images failed to download"
  exit 1
fi

echo "All images downloaded successfully"
