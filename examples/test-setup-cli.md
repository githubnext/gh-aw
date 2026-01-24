---
name: Test setup-cli action
engine: copilot

on:
  workflow_dispatch:

# This is a test workflow to demonstrate the setup-cli action
# It shows both tag and SHA-based installation

jobs:
  test-tag-installation:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4
      
      - name: Install gh-aw using tag
        uses: ./actions/setup-cli
        with:
          version: v0.37.18
      
      - name: Verify installation
        run: |
          gh aw version
          gh aw --help

  test-sha-installation:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4
      
      # This demonstrates SHA resolution
      # Note: The SHA must correspond to a published release
      - name: Install gh-aw using SHA
        id: install
        uses: ./actions/setup-cli
        with:
          version: "53a14809f3234d628d47864d48170c48e5bb25b9"  # Corresponds to a release
      
      - name: Verify installation and check output
        run: |
          echo "Installed version: ${{ steps.install.outputs.installed-version }}"
          gh aw version
