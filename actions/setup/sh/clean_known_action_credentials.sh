#!/usr/bin/env bash
set +o histexpand

#
# clean_known_action_credentials.sh - Remove credentials left by known GitHub Actions
#
# This script removes credential files and directories created by well-known GitHub
# Actions that authenticate to cloud providers or container registries. These
# credentials must be removed before the agentic engine executes to prevent
# unauthorized access or exfiltration.
#
# Environment variables (set to "true" to enable the corresponding cleanup):
#   GH_AW_CLEAN_GCP    - Remove GCP credentials from google-github-actions/auth
#   GH_AW_CLEAN_AWS    - Remove AWS credentials from aws-actions/configure-aws-credentials
#   GH_AW_CLEAN_AZURE  - Remove Azure credentials from azure/login
#   GH_AW_CLEAN_DOCKER - Remove Docker credentials from docker/login-action
#   GH_AW_CLEAN_SSH    - Remove SSH keys from actions/checkout (deploy key)
#
# Exit codes:
#   0 - Success (credentials cleaned or nothing to clean)
#   1 - Error (unexpected failure)

set -euo pipefail

WORKSPACE="${GITHUB_WORKSPACE:-.}"
HOME_DIR="${HOME:-/home/runner}"

CLEANED=0

# GCP credentials from google-github-actions/auth
# Creates ./gha-creds-*.json files in the workspace directory containing GCP service account keys
if [[ "${GH_AW_CLEAN_GCP:-}" == "true" ]]; then
  echo "Cleaning GCP credentials (google-github-actions/auth)..."
  while IFS= read -r cred_file; do
    rm -f "${cred_file}"
    echo "Removed GCP credential file: ${cred_file@Q}"
    CLEANED=$((CLEANED + 1))
  done < <(find "${WORKSPACE}" -maxdepth 1 -name "gha-creds-*.json" 2>/dev/null) || true
fi

# AWS credentials from aws-actions/configure-aws-credentials
# Creates ~/.aws/credentials containing AWS access keys
if [[ "${GH_AW_CLEAN_AWS:-}" == "true" ]]; then
  echo "Cleaning AWS credentials (aws-actions/configure-aws-credentials)..."
  if [[ -f "${HOME_DIR}/.aws/credentials" ]]; then
    rm -f "${HOME_DIR}/.aws/credentials"
    echo "Removed AWS credentials: ${HOME_DIR}/.aws/credentials"
    CLEANED=$((CLEANED + 1))
  fi
fi

# Azure credentials from azure/login
# Creates ~/.azure/ directory containing Azure service principal credentials
if [[ "${GH_AW_CLEAN_AZURE:-}" == "true" ]]; then
  echo "Cleaning Azure credentials (azure/login)..."
  if [[ -d "${HOME_DIR}/.azure" ]]; then
    rm -rf "${HOME_DIR}/.azure"
    echo "Removed Azure credentials directory: ${HOME_DIR}/.azure"
    CLEANED=$((CLEANED + 1))
  fi
fi

# Docker credentials from docker/login-action
# Creates ~/.docker/config.json containing registry auth tokens
if [[ "${GH_AW_CLEAN_DOCKER:-}" == "true" ]]; then
  echo "Cleaning Docker credentials (docker/login-action)..."
  if [[ -f "${HOME_DIR}/.docker/config.json" ]]; then
    rm -f "${HOME_DIR}/.docker/config.json"
    echo "Removed Docker credentials: ${HOME_DIR}/.docker/config.json"
    CLEANED=$((CLEANED + 1))
  fi
fi

# SSH keys from actions/checkout (deploy key)
# Adds SSH private keys to the SSH agent and to ~/.ssh/ when a deploy key is configured
if [[ "${GH_AW_CLEAN_SSH:-}" == "true" ]]; then
  echo "Cleaning SSH keys (actions/checkout with deploy key)..."
  # Clear all keys from the SSH agent
  if command -v ssh-add >/dev/null 2>&1; then
    if ssh-add -D 2>/dev/null; then
      echo "Cleared SSH agent keys"
      CLEANED=$((CLEANED + 1))
    fi
  fi
  # Remove deploy key files added by checkout (typically named id_rsa, id_ed25519, etc.)
  # Only remove private key files (no .pub extension) that are not system defaults
  while IFS= read -r key_file; do
    rm -f "${key_file}"
    echo "Removed SSH key file: ${key_file@Q}"
    CLEANED=$((CLEANED + 1))
  done < <(find "${HOME_DIR}/.ssh" -maxdepth 1 -type f ! -name "*.pub" ! -name "known_hosts" ! -name "config" ! -name "authorized_keys" 2>/dev/null) || true
fi

if [[ "${CLEANED}" -eq 0 ]]; then
  echo "No known action credentials found to clean"
fi

exit 0
