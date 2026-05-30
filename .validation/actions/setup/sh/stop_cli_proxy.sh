#!/usr/bin/env bash
set +o histexpand

# Stop CLI proxy container after AWF execution
# This script removes the awmg-cli-proxy container started by start_cli_proxy.sh.

set -e

docker rm -f awmg-cli-proxy 2>/dev/null || true
