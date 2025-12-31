#!/usr/bin/env bash
# Download Docker images with retry logic
# Usage: download_docker_images.sh IMAGE1 [IMAGE2 ...]

set -e

# Helper function to pull Docker images with retry logic
docker_pull_with_retry() {
  local image="$1"
  local max_attempts=3
  local attempt=1
  local wait_time=5
  
  while [ $attempt -le $max_attempts ]; do
    echo "Attempt $attempt of $max_attempts: Pulling $image..."
    if docker pull --quiet "$image"; then
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

# Pull all images passed as arguments
for image in "$@"; do
  docker_pull_with_retry "$image"
done
