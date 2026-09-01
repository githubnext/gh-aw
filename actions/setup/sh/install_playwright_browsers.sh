#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -eq 0 ]; then
  set -- chromium
fi

max_attempts=3
for browser in "$@"; do
  case "${browser,,}" in
    chrome|chromium) browser=chromium ;;
    firefox|webkit) ;;
    *)
      echo "::error::Unsupported Playwright browser: ${browser}"
      exit 1
      ;;
  esac

  for ((attempt = 1; attempt <= max_attempts; attempt++)); do
    echo "Installing Playwright ${browser} browser (attempt ${attempt}/${max_attempts})"
    if playwright-cli install-browser "$browser"; then
      break
    fi
    if [ "$attempt" -eq "$max_attempts" ]; then
      echo "::error::Failed to install Playwright ${browser} browser after ${max_attempts} attempts"
      exit 1
    fi
    sleep $((attempt * 5))
  done
done
