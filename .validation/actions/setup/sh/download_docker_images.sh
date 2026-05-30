#!/usr/bin/env bash
set +o histexpand

# Download Docker images with retry logic and controlled parallelism
# Usage: download_docker_images.sh IMAGE1 [IMAGE2 ...]
#
# This script downloads multiple Docker images in parallel with controlled
# parallelism (max 4 concurrent downloads) to improve performance without
# overwhelming the system. Docker daemon supports concurrent pulls, which can
# provide significant speedup when downloading multiple images.
#
# Each image is pulled with retry logic (3 attempts with exponential backoff).
# The script fails if any image fails to download after all retry attempts.
#
# When images include a digest pin (e.g. image:tag@sha256:abc), the script
# ensures the tag alias (image:tag) is created after pulling so that tools
# referencing images by tag (such as AWF with --skip-pull) can find them.

set -euo pipefail

# Helper function to pull Docker images with retry logic
docker_pull_with_retry() {
  local image="$1"
  local max_attempts=3
  local wait_time=5
  
  for attempt in $(seq 1 $max_attempts); do
    echo "Attempt $attempt of $max_attempts: Pulling $image..."
    
    if timeout 5m docker pull --quiet "$image" 2>&1; then
      echo "Successfully pulled $image"

      # When pulling with a digest pin (image:tag@sha256:...), Docker may not
      # create the tag alias automatically. Ensure the tag exists so that
      # downstream tools (e.g. AWF --skip-pull) can find the image by tag.
      if [[ "$image" == *"@sha256:"* ]]; then
        local tag_ref="${image%%@sha256:*}"
        if [[ "$tag_ref" == *":"* ]]; then
          echo "Tagging digest-pinned image as $tag_ref"
          docker tag "$image" "$tag_ref"
        fi
      fi

      return 0
    fi
    
    local exit_code=$?
    
    # Timeout produces exit code 124
    if [ $exit_code -eq 124 ]; then
      echo "docker pull timed out for $image after 5 minutes"
      return 1
    fi
    
    # Retry with exponential backoff unless this was the last attempt
    if [ "$attempt" -lt "$max_attempts" ]; then
      echo "Failed to pull $image. Retrying in ${wait_time}s..."
      sleep $wait_time
      wait_time=$((wait_time * 2))
    else
      echo "Failed to pull $image after $max_attempts attempts"
      return 1
    fi
  done
}

# Export function so xargs can use it
export -f docker_pull_with_retry

# Pull images with controlled parallelism using xargs
echo "Starting download of ${#@} image(s) with max 4 concurrent downloads..."
printf '%s\n' "$@" | xargs -P 4 -I {} bash -c 'docker_pull_with_retry "$@"' _ {}

echo "All images downloaded successfully"
