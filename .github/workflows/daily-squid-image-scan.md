---
name: Daily Squid Image Security Scan
description: Scan the pinned agentic workflow firewall image for vulnerabilities, updates, and rejected licenses
emoji: "🛡️"
on:
  schedule: daily
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  copilot-requests: write
strict: true
if: always() && needs.scan_image.result != 'skipped'
network:
  allowed:
    - defaults
tools:
  cli-proxy: true
  bash:
    - "cat /tmp/gh-aw/agent/image-scan/*"
    - "jq * /tmp/gh-aw/agent/image-scan/*"
safe-outputs:
  create-issue:
    title-prefix: "[squid-image-scan] "
    labels: [cookie, security]
    max: 1
    deduplicate-by-title: true
steps:
  - name: Download image scan results
    id: download_scan
    continue-on-error: true
    uses: actions/download-artifact@v8.0.1
    with:
      name: squid-image-scan
      path: /tmp/gh-aw/agent/image-scan
  - name: Ensure image scan summary exists
    if: always()
    env:
      DOWNLOAD_OUTCOME: ${{ steps.download_scan.outcome }}
    run: |
      results="/tmp/gh-aw/agent/image-scan"
      mkdir -p "$results"
      if [ "$DOWNLOAD_OUTCOME" != "success" ] ||
         ! jq -e 'type == "object"' "$results/summary.json" > /dev/null 2>&1; then
        jq -n '{
          image_tag: "unknown",
          pinned_image: "unknown",
          index_digest: "unknown",
          current_digest: "unknown",
          image_updated: false,
          platforms: [],
          total_vulnerabilities: 0,
          critical_vulnerabilities: 0,
          fixable_vulnerabilities: 0,
          license_rejected: false,
          operational_errors: ["The scan job did not publish a valid result artifact."],
          tools: {
            syft: "1.49.0",
            grype: "0.116.0",
            grant: "0.6.8"
          }
        }' > "$results/summary.json"
      fi
post-steps:
  - name: Enforce critical vulnerability and license gates
    if: always()
    run: |
      summary="/tmp/gh-aw/agent/image-scan/summary.json"
      jq -e '
        (.critical_vulnerabilities == 0) and
        (.license_rejected == false) and
        ((.operational_errors | length) == 0)
      ' "$summary"
jobs:
  scan_image:
    name: Scan immutable firewall image
    runs-on: ubuntu-latest
    timeout-minutes: 30
    permissions:
      contents: read
      packages: read
    steps:
      - name: Checkout repository
        uses: actions/checkout@v7.0.0
        with:
          persist-credentials: false
      - name: Install pinned security tools
        id: install_tools
        continue-on-error: true
        env:
          SYFT_VERSION: "1.49.0"
          SYFT_SHA256: "7aa2f03ee92739cf643279ba3990548b9925d4e22cae13f46831ee62821147fe"
          GRYPE_VERSION: "0.116.0"
          GRYPE_SHA256: "40aff724297312f91ea390d003bed8d8651c74cc7f5b26732db80b3a408d2fc5"
          GRANT_VERSION: "0.6.8"
          GRANT_SHA256: "6500f8bbf0f20fb993de8084686e199f0ba1eb494769ff75454286d5ef63f919"
        run: |
          set -euo pipefail
          test "$(uname -m)" = "x86_64"
          tools_dir="$RUNNER_TEMP/image-scan-tools"
          mkdir -p "$tools_dir"

          install_tool() {
            tool="$1"
            version="$2"
            checksum="$3"
            archive="$RUNNER_TEMP/${tool}.tar.gz"
            curl -fsSL \
              "https://github.com/anchore/${tool}/releases/download/v${version}/${tool}_${version}_linux_amd64.tar.gz" \
              -o "$archive"
            echo "${checksum}  ${archive}" | sha256sum -c -
            tar -xzf "$archive" --no-same-owner -C "$tools_dir" "$tool"
          }

          install_tool syft "$SYFT_VERSION" "$SYFT_SHA256"
          install_tool grype "$GRYPE_VERSION" "$GRYPE_SHA256"
          install_tool grant "$GRANT_VERSION" "$GRANT_SHA256"
          echo "$tools_dir" >> "$GITHUB_PATH"
      - name: Scan each platform image
        if: always()
        env:
          INSTALL_OUTCOME: ${{ steps.install_tools.outcome }}
        run: |
          set -uo pipefail
          output="$RUNNER_TEMP/squid-image-scan"
          mkdir -p "$output"
          : > "$output/errors.txt"
          : > "$output/platforms.tsv"

          record_error() {
            printf '%s\n' "$1" | tee -a "$output/errors.txt"
          }

          image_prefix="ghcr.io/github/gh-aw-firewall/squid:"
          pin_file="pkg/actionpins/data/action_pins.json"
          mirror_file="pkg/workflow/data/action_pins.json"

          pin_entry=$(jq -cer --arg prefix "$image_prefix" '
            [.entries | to_entries[] | select(.key | startswith($prefix))]
            | if length == 1 then .[0] else error("expected exactly one squid image pin") end
          ' "$pin_file") || record_error "Unable to read the canonical Squid image pin."
          mirror_entry=$(jq -cer --arg prefix "$image_prefix" '
            [.entries | to_entries[] | select(.key | startswith($prefix))]
            | if length == 1 then .[0] else error("expected exactly one squid image pin") end
          ' "$mirror_file") || record_error "Unable to read the mirrored Squid image pin."

          if [ -z "${pin_entry:-}" ] || [ -z "${mirror_entry:-}" ]; then
            image_tag="unknown"
            pinned_image="unknown"
            index_digest="unknown"
          else
            if [ "$pin_entry" != "$mirror_entry" ]; then
              record_error "The canonical and mirrored Squid image pins disagree."
            fi
            image_tag=$(jq -r '.key' <<<"$pin_entry")
            pinned_image=$(jq -r '.value.pinned_image' <<<"$pin_entry")
            index_digest=$(jq -r '.value.digest' <<<"$pin_entry")
            if [[ ! "$pinned_image" =~ @sha256:[0-9a-f]{64}$ ]] ||
               [[ ! "$index_digest" =~ ^sha256:[0-9a-f]{64}$ ]]; then
              record_error "The Squid image is not pinned to a valid SHA-256 digest."
            fi
          fi

          current_digest="unknown"
          image_updated=false
          if [ "$image_tag" != "unknown" ]; then
            if current_digest=$(
              docker buildx imagetools inspect "$image_tag" --format '{{json .Manifest}}' |
                jq -er '.digest'
            ); then
              if [ "$current_digest" != "$index_digest" ]; then
                image_updated=true
              fi
            else
              current_digest="unknown"
              record_error "Unable to resolve the current Squid image tag digest."
            fi
          fi

          tools_ready=true
          if [ "$INSTALL_OUTCOME" != "success" ]; then
            tools_ready=false
            record_error "Pinned security tool installation failed."
          fi

          total_vulnerabilities=0
          critical_vulnerabilities=0
          fixable_vulnerabilities=0
          license_rejected=false

          if [ "$pinned_image" != "unknown" ]; then
            if docker buildx imagetools inspect "$pinned_image" --raw > "$output/index.json"; then
              if ! echo "${index_digest#sha256:}  $output/index.json" | sha256sum -c -; then
                record_error "The downloaded image index does not match its pinned digest."
              fi
            else
              record_error "Unable to download the pinned image index."
              echo '{"manifests":[]}' > "$output/index.json"
            fi
          else
            echo '{"manifests":[]}' > "$output/index.json"
          fi

          if [ "$tools_ready" = true ]; then
            if ! grype db update > "$output/grype-db-update.txt" 2>&1; then
              record_error "Grype vulnerability database update failed."
            fi
            grype db status > "$output/grype-db-status.txt" 2>&1 || true
          fi

          for platform in linux/amd64 linux/arm64; do
            arch="${platform#linux/}"
            suffix="linux-${arch}"
            child_digest=$(jq -er --arg arch "$arch" '
              [.manifests[] | select(.platform.os == "linux" and .platform.architecture == $arch)]
              | if length == 1 then .[0].digest else error("expected exactly one platform manifest") end
            ' "$output/index.json") || {
              record_error "Unable to resolve exactly one ${platform} image manifest."
              continue
            }
            printf '%s\t%s\n' "$platform" "$child_digest" >> "$output/platforms.tsv"

            if [ "$tools_ready" != true ]; then
              continue
            fi

            image_repository="${image_tag%:*}"
            immutable_image="${image_repository}@${child_digest}"
            syft_json="$output/sbom-${suffix}.syft.json"
            spdx_json="$output/sbom-${suffix}.spdx.json"

            if ! syft --platform "$platform" "$immutable_image" \
              -o "syft-json=${syft_json}" \
              -o "spdx-json=${spdx_json}" > "$output/syft-${suffix}.txt" 2>&1; then
              record_error "Syft failed to scan ${platform} at ${child_digest}."
              continue
            fi

            if ! GRYPE_DB_AUTO_UPDATE=false grype "sbom:${syft_json}" \
              -o json > "$output/grype-${suffix}.json" 2> "$output/grype-${suffix}.stderr"; then
              record_error "Grype failed to produce JSON results for ${platform}."
              continue
            fi

            platform_total=$(jq '.matches | length' "$output/grype-${suffix}.json")
            platform_critical=$(jq '
              [.matches[] | select((.vulnerability.severity // "") | ascii_downcase == "critical")]
              | length
            ' "$output/grype-${suffix}.json")
            platform_fixable=$(jq '
              [.matches[] | select((.vulnerability.fix.versions // []) | length > 0)]
              | length
            ' "$output/grype-${suffix}.json")
            total_vulnerabilities=$((total_vulnerabilities + platform_total))
            critical_vulnerabilities=$((critical_vulnerabilities + platform_critical))
            fixable_vulnerabilities=$((fixable_vulnerabilities + platform_fixable))

            GRYPE_DB_AUTO_UPDATE=false grype "sbom:${syft_json}" \
              --fail-on critical -o table > "$output/grype-${suffix}.txt" 2>&1
            grype_exit=$?
            if [ "$grype_exit" -ne 0 ] && [ "$platform_critical" -eq 0 ]; then
              record_error "Grype table scan failed unexpectedly for ${platform}."
            fi

            if ! grant list "$spdx_json" > "$output/grant-list-${suffix}.txt" 2>&1; then
              record_error "Grant could not list licenses for ${platform}."
            fi
            grant_json="$output/grant-check-${suffix}.json"
            grant_stderr="$output/grant-check-${suffix}.stderr"
            grant check --config .grant.yaml --output json "$spdx_json" \
              > "$grant_json" 2> "$grant_stderr"
            grant_exit=$?
            if ! jq -e '.run.targets | length > 0' "$grant_json" > /dev/null 2>&1; then
              record_error "Grant did not produce valid results for ${platform}."
            elif jq -e '
              any(.run.targets[]; .evaluation.status == "error")
            ' "$grant_json" > /dev/null; then
              record_error "Grant encountered an evaluation error for ${platform}."
            elif jq -e '
              any(.run.targets[]; .evaluation.status == "noncompliant")
            ' "$grant_json" > /dev/null; then
              license_rejected=true
            elif [ "$grant_exit" -ne 0 ]; then
              record_error "Grant failed unexpectedly for ${platform}."
            fi
          done

          errors=$(jq -Rs 'split("\n") | map(select(length > 0))' "$output/errors.txt")
          platforms=$(jq -Rn '
            [inputs | split("\t") | {platform: .[0], digest: .[1]}]
          ' < "$output/platforms.tsv")
          jq -n \
            --arg image_tag "$image_tag" \
            --arg pinned_image "$pinned_image" \
            --arg index_digest "$index_digest" \
            --arg current_digest "$current_digest" \
            --argjson image_updated "$image_updated" \
            --argjson platforms "$platforms" \
            --argjson total_vulnerabilities "$total_vulnerabilities" \
            --argjson critical_vulnerabilities "$critical_vulnerabilities" \
            --argjson fixable_vulnerabilities "$fixable_vulnerabilities" \
            --argjson license_rejected "$license_rejected" \
            --argjson operational_errors "$errors" \
            '{
              image_tag: $image_tag,
              pinned_image: $pinned_image,
              index_digest: $index_digest,
              current_digest: $current_digest,
              image_updated: $image_updated,
              platforms: $platforms,
              total_vulnerabilities: $total_vulnerabilities,
              critical_vulnerabilities: $critical_vulnerabilities,
              fixable_vulnerabilities: $fixable_vulnerabilities,
              license_rejected: $license_rejected,
              operational_errors: $operational_errors,
              tools: {
                syft: "1.49.0",
                grype: "0.116.0",
                grant: "0.6.8"
              }
            }' > "$output/summary.json"
      - name: Upload image scan results
        if: always()
        uses: actions/upload-artifact@v7.0.1
        with:
          name: squid-image-scan
          path: ${{ runner.temp }}/squid-image-scan
          if-no-files-found: error
          retention-days: 14
sandbox:
  agent:
    sudo: false
timeout-minutes: 20
---

# Daily Squid Image Security Scan

Review the deterministic Syft, Grype, and Grant results in
`/tmp/gh-aw/agent/image-scan/`.

1. Read `summary.json` first.
2. If the image tag has not changed, no vulnerabilities were found, all licenses
   are accepted, and there are no operational errors, call `noop`.
3. Otherwise, create one issue. Use the title
   `Container findings for <first 12 characters of index_digest>` so repeated
   findings for the same immutable image are deduplicated.
4. Include:
   - the pinned image, current tag digest, and both platform child digests;
   - scanner versions and Grype database status;
   - every vulnerability from both `grype-linux-*.json` files, with platform,
     severity, vulnerability ID, package, installed version, and fixed versions;
   - every rejected or unknown license shown in the
     `grant-check-linux-*.json` files;
   - image digest drift, operational errors, and actionable remediation.
5. Keep the report factual and compact. Never omit lower-severity
   vulnerabilities.

The configured `create-issue` safe output is the only allowed write operation.
