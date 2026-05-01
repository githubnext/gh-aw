---
safe-outputs:
  threat-detection:
    steps:
      - name: Download cache-memory artifact for TruffleHog scan
        id: download-cache-memory-trufflehog
        continue-on-error: true
        uses: actions/download-artifact@v8
        with:
          name: cache-memory
          path: /tmp/gh-aw/cache-memory

      - name: Download repo-memory artifact for TruffleHog scan
        id: download-repo-memory-trufflehog
        continue-on-error: true
        uses: actions/download-artifact@v8
        with:
          name: repo-memory-default
          path: /tmp/gh-aw/repo-memory/default

      - name: Install TruffleHog
        id: install-trufflehog
        run: |
          echo "Installing TruffleHog..."
          curl -sSfL https://raw.githubusercontent.com/trufflesecurity/trufflehog/main/scripts/install.sh | sh -s -- -b /usr/local/bin
          trufflehog --version
          echo "TruffleHog installed successfully"

      - name: Run TruffleHog on safe outputs
        id: trufflehog-safeoutputs
        continue-on-error: true
        run: |
          mkdir -p /tmp/gh-aw/trufflehog
          SCAN_DIR="/tmp/gh-aw/threat-detection"
          OUTPUT_FILE="/tmp/gh-aw/trufflehog/safeoutputs-results.jsonl"
          if [ -d "$SCAN_DIR" ] && [ "$(ls -A "$SCAN_DIR" 2>/dev/null)" ]; then
            echo "Scanning safe outputs in $SCAN_DIR"
            trufflehog filesystem "$SCAN_DIR" --json --no-update --fail 2>/dev/null | tee "$OUTPUT_FILE" || SCAN_EXIT=${PIPESTATUS[0]}
            SCAN_EXIT=${SCAN_EXIT:-0}
          else
            echo "Safe outputs directory is empty or missing, skipping scan"
            SCAN_EXIT=0
          fi
          if [ "$SCAN_EXIT" -eq 183 ]; then
            echo "safeoutputs_secrets_found=true" >> "$GITHUB_OUTPUT"
          fi
          echo "exit_code=${SCAN_EXIT}" >> "$GITHUB_OUTPUT"

      - name: Run TruffleHog on cache-memory
        id: trufflehog-cache-memory
        continue-on-error: true
        run: |
          mkdir -p /tmp/gh-aw/trufflehog
          SCAN_DIR="/tmp/gh-aw/cache-memory"
          OUTPUT_FILE="/tmp/gh-aw/trufflehog/cache-memory-results.jsonl"
          if [ -d "$SCAN_DIR" ] && [ "$(ls -A "$SCAN_DIR" 2>/dev/null)" ]; then
            echo "Scanning cache-memory in $SCAN_DIR"
            trufflehog filesystem "$SCAN_DIR" --json --no-update --fail 2>/dev/null | tee "$OUTPUT_FILE" || SCAN_EXIT=${PIPESTATUS[0]}
            SCAN_EXIT=${SCAN_EXIT:-0}
          else
            echo "cache-memory directory is empty or missing, skipping scan"
            SCAN_EXIT=0
          fi
          if [ "$SCAN_EXIT" -eq 183 ]; then
            echo "cache_memory_secrets_found=true" >> "$GITHUB_OUTPUT"
          fi
          echo "exit_code=${SCAN_EXIT}" >> "$GITHUB_OUTPUT"

      - name: Run TruffleHog on repo-memory
        id: trufflehog-repo-memory
        continue-on-error: true
        run: |
          mkdir -p /tmp/gh-aw/trufflehog
          SCAN_DIR="/tmp/gh-aw/repo-memory"
          OUTPUT_FILE="/tmp/gh-aw/trufflehog/repo-memory-results.jsonl"
          if [ -d "$SCAN_DIR" ] && [ "$(ls -A "$SCAN_DIR" 2>/dev/null)" ]; then
            echo "Scanning repo-memory in $SCAN_DIR"
            trufflehog filesystem "$SCAN_DIR" --json --no-update --fail 2>/dev/null | tee "$OUTPUT_FILE" || SCAN_EXIT=${PIPESTATUS[0]}
            SCAN_EXIT=${SCAN_EXIT:-0}
          else
            echo "repo-memory directory is empty or missing, skipping scan"
            SCAN_EXIT=0
          fi
          if [ "$SCAN_EXIT" -eq 183 ]; then
            echo "repo_memory_secrets_found=true" >> "$GITHUB_OUTPUT"
          fi
          echo "exit_code=${SCAN_EXIT}" >> "$GITHUB_OUTPUT"

      - name: Evaluate TruffleHog results
        id: trufflehog-evaluate
        if: always()
        uses: actions/github-script@v9
        env:
          SAFEOUTPUTS_SECRETS_FOUND: ${{ steps.trufflehog-safeoutputs.outputs.safeoutputs_secrets_found }}
          CACHE_MEMORY_SECRETS_FOUND: ${{ steps.trufflehog-cache-memory.outputs.cache_memory_secrets_found }}
          REPO_MEMORY_SECRETS_FOUND: ${{ steps.trufflehog-repo-memory.outputs.repo_memory_secrets_found }}
        with:
          script: |
            const safeOutputs = process.env.SAFEOUTPUTS_SECRETS_FOUND === 'true';
            const cacheMemory = process.env.CACHE_MEMORY_SECRETS_FOUND === 'true';
            const repoMemory = process.env.REPO_MEMORY_SECRETS_FOUND === 'true';

            const secretsFound = safeOutputs || cacheMemory || repoMemory;

            core.info('='.repeat(60));
            core.info('🔍 TruffleHog Secret Scan Summary');
            core.info('='.repeat(60));
            core.info(`Safe outputs:  ${safeOutputs ? '❌ SECRETS FOUND' : '✅ clean'}`);
            core.info(`Cache-memory:  ${cacheMemory ? '❌ SECRETS FOUND' : '✅ clean'}`);
            core.info(`Repo-memory:   ${repoMemory ? '❌ SECRETS FOUND' : '✅ clean'}`);
            core.info('='.repeat(60));

            if (secretsFound) {
              const locations = [
                safeOutputs && 'safe outputs',
                cacheMemory && 'cache-memory',
                repoMemory && 'repo-memory',
              ].filter(Boolean).join(', ');

              core.setOutput('secrets_found', 'true');
              core.setOutput('secrets_locations', locations);
              core.setFailed(`❌ TruffleHog detected secrets in: ${locations}. Review the scan results artifacts for details.`);
            } else {
              core.setOutput('secrets_found', 'false');
              core.info('✅ No secrets detected by TruffleHog');
            }

      - name: Upload TruffleHog scan results
        if: always()
        uses: actions/upload-artifact@v7.0.1
        with:
          name: trufflehog-scan-results
          path: /tmp/gh-aw/trufflehog/
          if-no-files-found: ignore
---

# TruffleHog Secret Detection

This shared workflow adds [TruffleHog](https://github.com/trufflesecurity/trufflehog) secret scanning to the detection job. It scans the agent's safe outputs, cache-memory, and repo-memory for accidentally leaked secrets (API keys, tokens, credentials, etc.).

## How It Works

1. **Download artifacts** — fetches `cache-memory` and `repo-memory` artifacts produced by the agent job (errors are non-fatal if the workflow doesn't use those features)
2. **Install TruffleHog** — downloads and installs the latest TruffleHog binary
3. **Scan safe outputs** — runs TruffleHog on `/tmp/gh-aw/threat-detection/` (agent output and any code patches)
4. **Scan cache-memory** — runs TruffleHog on `/tmp/gh-aw/cache-memory/`
5. **Scan repo-memory** — runs TruffleHog on `/tmp/gh-aw/repo-memory/`
6. **Evaluate** — aggregates results; sets `secrets_found=true` output and fails the detection job if any secrets are detected
7. **Upload results** — saves JSONL scan result files as `trufflehog-scan-results` artifact for review

## Outputs

| Step ID | Output | Value |
|---------|--------|-------|
| `trufflehog-evaluate` | `secrets_found` | `true` or `false` |
| `trufflehog-evaluate` | `secrets_locations` | Comma-separated list of locations where secrets were found |

When `secrets_found=true` the step calls `core.setFailed()`, which fails the detection job and prevents safe outputs from being processed. The conclusion job will observe a failed `detection_conclusion` and respond accordingly.

## Usage

```yaml
---
imports:
  - shared/trufflehog.md
---
```

## Scan Results

Raw JSONL scan output is uploaded as the `trufflehog-scan-results` artifact and contains one JSON object per detected finding. Each finding includes the source type, file path, detector name, and the raw/redacted secret value.
