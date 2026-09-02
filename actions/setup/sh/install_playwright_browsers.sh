#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -eq 0 ]; then
  set -- chromium
fi

normalized_browsers=()
for browser in "$@"; do
  case "${browser,,}" in
    chrome|chromium) browser=chromium ;;
    firefox|webkit) ;;
    *)
      echo "::error::Unsupported Playwright browser: ${browser}"
      exit 1
      ;;
  esac
  normalized_browsers+=("$browser")
done

# Install the OS-level shared library dependencies (e.g. libnspr4, libnss3,
# libatk-bridge2.0-0) required for the browser binaries to actually launch.
# Without this step, browser processes can be downloaded successfully but
# fail to start at runtime with missing shared library errors.
playwright_cli_bin="$(command -v playwright-cli || true)"
if [ -n "$playwright_cli_bin" ]; then
  playwright_cli_real="$(readlink -f "$playwright_cli_bin")"
  playwright_js="$(dirname "$playwright_cli_real")/node_modules/playwright/cli.js"
  if [ -f "$playwright_js" ]; then
    echo "Installing Playwright system dependencies for: ${normalized_browsers[*]}"
    node "$playwright_js" install-deps "${normalized_browsers[@]}"
  else
    echo "::warning::Could not locate playwright/cli.js next to playwright-cli; skipping system dependency install"
  fi
else
  echo "::warning::playwright-cli not found on PATH; skipping system dependency install"
fi

max_attempts=3
for browser in "${normalized_browsers[@]}"; do
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
