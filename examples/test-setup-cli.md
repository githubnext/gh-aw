---
name: Test setup-cli action
engine: copilot

on:
  workflow_dispatch:

# This is a test workflow to demonstrate the setup-cli action

jobs:
  test-installation:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4
      
      - name: Install gh-aw using release tag
        id: install
        uses: ./actions/setup-cli
        with:
          version: v0.37.18
      
      - name: Verify installation
        run: |
          echo "Installed version: ${{ steps.install.outputs.installed-version }}"
          gh aw version
          gh aw --help

  test-matrix:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        version: [v0.37.18, v0.37.17]
    steps:
      - name: Checkout
        uses: actions/checkout@v4
      
      - name: Install gh-aw version ${{ matrix.version }}
        uses: ./actions/setup-cli
        with:
          version: ${{ matrix.version }}
      
      - name: Verify installation
        run: |
          gh aw version
