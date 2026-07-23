---
name: Daily Container Image Security Scan
description: Scan container images used by compiled workflows for vulnerabilities, updates, and rejected licenses
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
    - "grep * /tmp/gh-aw/agent/image-scan/grype-compile.txt"
safe-outputs:
  create-issue:
    title-prefix: "[container-image-scan] "
    labels: [cookie, security]
    max: 25
    deduplicate-by-title: true
  noop:
    report-as-issue: false
steps:
  - name: Download image scan results
    id: download_scan
    continue-on-error: true
    uses: actions/download-artifact@v8.0.1
    with:
      name: container-image-scan
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
          images: [],
          grype_has_errors: false,
          license_rejected: false,
          operational_errors: ["The scan job did not publish a valid result artifact."],
          tools: {
            syft: "unknown",
            grant: "unknown"
          }
        }' > "$results/summary.json"
      fi
post-steps:
  - name: Enforce critical vulnerability and license gates
    if: always()
    env:
      SCAN_JOB_RESULT: ${{ needs.scan_image.result }}
    run: |
      if [ "$SCAN_JOB_RESULT" != "success" ]; then
        echo "::error::Image scan job concluded with ${SCAN_JOB_RESULT}."
        exit 1
      fi
      summary="/tmp/gh-aw/agent/image-scan/summary.json"
      jq -e '
        (.grype_has_errors == false) and
        (.license_rejected == false) and
        ((.operational_errors | length) == 0)
      ' "$summary"
jobs:
  scan_image:
    name: Scan immutable workflow images
    runs-on: ubuntu-latest
    timeout-minutes: 120
    permissions:
      contents: read
      packages: read
    env:
      SYFT_VERSION: "1.49.0"
      SYFT_SHA256_AMD64: "7aa2f03ee92739cf643279ba3990548b9925d4e22cae13f46831ee62821147fe"
      SYFT_SHA256_ARM64: "c7c32de183c32368de197edba75e8dba7632915f7761bacd55149a9ca7fe0fa4"
      GRANT_VERSION: "0.6.8"
      GRANT_SHA256_AMD64: "6500f8bbf0f20fb993de8084686e199f0ba1eb494769ff75454286d5ef63f919"
      GRANT_SHA256_ARM64: "15ec0b4346a64b5580958dc62c4e7c25ca9e59b7582bab9706679f6b9d2288b8"
    steps:
      - name: Checkout repository
        uses: actions/checkout@v7.0.0
        with:
          persist-credentials: false
      - name: Build gh-aw from source
        run: |
          set -euo pipefail
          make build
      - name: Install pinned security tools
        id: install_tools
        continue-on-error: true
        run: |
          set -euo pipefail
          arch="$(uname -m)"
          case "$arch" in
            x86_64)
              suffix="amd64"
              ;;
            aarch64|arm64)
              suffix="arm64"
              ;;
            *)
              echo "Unsupported runner architecture: $arch" >&2
              exit 1
              ;;
          esac
          tools_dir="$RUNNER_TEMP/image-scan-tools"
          mkdir -p "$tools_dir"

          install_tool() {
            tool="$1"
            version="$2"
            checksum="$3"
            archive="$RUNNER_TEMP/${tool}.tar.gz"
            curl -fsSL \
              "https://github.com/anchore/${tool}/releases/download/v${version}/${tool}_${version}_linux_${suffix}.tar.gz" \
              -o "$archive"
            echo "${checksum}  ${archive}" | sha256sum -c -
            tar -xzf "$archive" --no-same-owner -C "$tools_dir" "$tool"
          }

          syft_sha256_var="SYFT_SHA256_${suffix^^}"
          grant_sha256_var="GRANT_SHA256_${suffix^^}"

          install_tool syft "$SYFT_VERSION" "${!syft_sha256_var}"
          install_tool grant "$GRANT_VERSION" "${!grant_sha256_var}"
          echo "$tools_dir" >> "$GITHUB_PATH"
      - name: Scan compiled workflow images
        if: always()
        env:
          INSTALL_OUTCOME: ${{ steps.install_tools.outcome }}
        run: |
          set -uo pipefail
          output="$RUNNER_TEMP/container-image-scan"
          mkdir -p "$output"
          : > "$output/errors.txt"
          : > "$output/image-results.jsonl"

          record_error() {
            printf '%s\n' "$1" | tee -a "$output/errors.txt"
          }

          if ! grep -h '^# gh-aw-manifest: ' .github/workflows/*.lock.yml |
            sed 's/^# gh-aw-manifest: //' |
            jq -s '
              [
                .[].containers[]?
                | select(.image | test("(^|/)gh-[^/:@]+"))
              ]
              | unique_by(.pinned_image)
              | sort_by(.image)
            ' > "$output/images.json"; then
            record_error "Unable to read container images from compiled workflow manifests."
            echo '[]' > "$output/images.json"
          fi

          if ! jq -e '
            length > 0 and
            all(.[];
              (.image | type == "string" and length > 0) and
              (.digest | test("^sha256:[0-9a-f]{64}$")) and
              (.pinned_image | test("@sha256:[0-9a-f]{64}$"))
            )
          ' "$output/images.json" > /dev/null; then
            record_error "Compiled workflow manifests contain no images or invalid image pins."
          fi

          tools_ready=true
          if [ "$INSTALL_OUTCOME" != "success" ]; then
            tools_ready=false
            record_error "Pinned security tool installation failed."
          fi

          license_rejected=false

          image_number=0
          while IFS= read -r image_entry; do
            image_number=$((image_number + 1))
            artifact_prefix=$(printf 'image-%02d' "$image_number")
            image_tag=$(jq -r '.image' <<<"$image_entry")
            pinned_image=$(jq -r '.pinned_image' <<<"$image_entry")
            index_digest=$(jq -r '.digest' <<<"$image_entry")
            echo "Scanning ${pinned_image}"
            image_errors="$output/${artifact_prefix}-errors.txt"
            image_platforms="$output/${artifact_prefix}-platforms.tsv"
            : > "$image_errors"
            : > "$image_platforms"

            record_image_error() {
              printf '%s\n' "$1" | tee -a "$output/errors.txt" "$image_errors"
            }

            current_digest="unknown"
            image_updated=false
            if current_digest=$(
              docker buildx imagetools inspect "$image_tag" --format '{{json .Manifest}}' |
                jq -er '.digest'
            ); then
              if [ "$current_digest" != "$index_digest" ]; then
                image_updated=true
              fi
            else
              current_digest="unknown"
              record_image_error "Unable to resolve the current digest for ${image_tag}."
            fi

            image_manifest="$output/${artifact_prefix}-manifest.json"
            if docker buildx imagetools inspect "$pinned_image" --raw > "$image_manifest"; then
              if ! echo "${index_digest#sha256:}  $image_manifest" | sha256sum -c -; then
                record_image_error "The manifest for ${pinned_image} does not match its pinned digest."
              fi
            else
              record_image_error "Unable to download the manifest for ${pinned_image}."
              echo '{}' > "$image_manifest"
            fi

            jq -r '
              .manifests[]?
              | select(.platform.os == "linux" and (.platform.architecture == "amd64" or .platform.architecture == "arm64"))
              | [
                  .platform.os + "/" + .platform.architecture +
                    (if .platform.variant then "/" + .platform.variant else "" end),
                  .digest
                ]
              | @tsv
            ' "$image_manifest" > "$image_platforms"
            if [ ! -s "$image_platforms" ]; then
              if jq -e 'has("manifests")' "$image_manifest" > /dev/null; then
                record_image_error "The image index for ${pinned_image} has no linux/amd64 or linux/arm64 manifests."
              elif platform=$(
                docker buildx imagetools inspect "$pinned_image" --format '{{json .Image}}' |
                  jq -er '
                    select(.os == "linux" and (.architecture == "amd64" or .architecture == "arm64"))
                    | .os + "/" + .architecture +
                      (if .variant then "/" + .variant else "" end)
                  '
              ); then
                printf '%s\t%s\n' "$platform" "$index_digest" > "$image_platforms"
              else
                record_image_error "Unable to resolve a Linux platform for ${pinned_image}."
              fi
            fi

            image_license_rejected=false

            while IFS=$'\t' read -r platform child_digest; do
              suffix="${artifact_prefix}-$(tr '/_' '--' <<<"$platform" | tr -cd '[:alnum:].-')"
              immutable_image="${pinned_image%@*}@${child_digest}"

              if [ "$tools_ready" != true ]; then
                continue
              fi

              syft_json="$output/sbom-${suffix}.syft.json"
              spdx_json="$output/sbom-${suffix}.spdx.json"
              syft_args=()
              if [ "$platform" != "default" ]; then
                syft_args=(--platform "$platform")
              fi

              if ! syft "${syft_args[@]}" "$immutable_image" \
                -o "syft-json=${syft_json}" \
                -o "spdx-json=${spdx_json}" > "$output/syft-${suffix}.txt" 2>&1; then
                cat "$output/syft-${suffix}.txt" >&2
                record_image_error "Syft failed to scan ${image_tag} for ${platform} at ${child_digest}."
                continue
              fi

              if ! grant list "$spdx_json" > "$output/grant-list-${suffix}.txt" 2>&1; then
                cat "$output/grant-list-${suffix}.txt" >&2
                record_image_error "Grant could not list licenses for ${image_tag} on ${platform}."
              fi
              grant_json="$output/grant-check-${suffix}.json"
              grant_stderr="$output/grant-check-${suffix}.stderr"
              grant_exit=0
              grant check --config .grant.yaml --output json "$spdx_json" \
                > "$grant_json" 2> "$grant_stderr" ||
                grant_exit=$?
              if ! jq -e '.run.targets | length > 0' "$grant_json" > /dev/null 2>&1; then
                cat "$grant_stderr" >&2
                record_image_error "Grant did not produce valid results for ${image_tag} on ${platform}."
              elif jq -e '
                any(.run.targets[]; .evaluation.status == "error")
              ' "$grant_json" > /dev/null; then
                cat "$grant_stderr" >&2
                record_image_error "Grant encountered an evaluation error for ${image_tag} on ${platform}."
              elif jq -e '
                any(.run.targets[]; .evaluation.status == "noncompliant")
              ' "$grant_json" > /dev/null; then
                image_license_rejected=true
              elif [ "$grant_exit" -ne 0 ]; then
                cat "$grant_stderr" >&2
                record_image_error "Grant failed unexpectedly for ${image_tag} on ${platform}."
              fi
              echo "${image_tag} ${platform}: scanned"
            done < "$image_platforms"

            if [ "$image_license_rejected" = true ]; then
              license_rejected=true
            fi

            platforms=$(jq -Rn '
              [inputs | split("\t") | {platform: .[0], digest: .[1]}]
            ' < "$image_platforms")
            image_operational_errors=$(jq -Rs '
              split("\n") | map(select(length > 0))
            ' "$image_errors")
            jq -n \
              --arg artifact_prefix "$artifact_prefix" \
              --arg image_tag "$image_tag" \
              --arg pinned_image "$pinned_image" \
              --arg index_digest "$index_digest" \
              --arg current_digest "$current_digest" \
              --argjson image_updated "$image_updated" \
              --argjson platforms "$platforms" \
              --argjson license_rejected "$image_license_rejected" \
              --argjson operational_errors "$image_operational_errors" \
              '{
                artifact_prefix: $artifact_prefix,
                image_tag: $image_tag,
                pinned_image: $pinned_image,
                index_digest: $index_digest,
                current_digest: $current_digest,
                image_updated: $image_updated,
                platforms: $platforms,
                license_rejected: $license_rejected,
                operational_errors: $operational_errors
              }' >> "$output/image-results.jsonl"
          done < <(jq -c '.[]' "$output/images.json")

          echo "$license_rejected" > "$output/license-rejected.txt"
      - name: Scan container images for vulnerabilities
        if: always()
        run: |
          set -uo pipefail
          output="$RUNNER_TEMP/container-image-scan"
          mkdir -p "$output"
          : > "$output/grype-compile.txt"
          "$GITHUB_WORKSPACE/gh-aw" compile --grype 2>&1 | tee "$output/grype-compile.txt" || true
      - name: Generate scan summary
        if: always()
        run: |
          set -uo pipefail
          output="$RUNNER_TEMP/container-image-scan"

          license_rejected=$(cat "$output/license-rejected.txt" 2>/dev/null || echo "false")
          grype_has_errors=false
          if grep -q '^::error' "$output/grype-compile.txt" 2>/dev/null; then
            grype_has_errors=true
          fi

          errors=$(jq -Rs 'split("\n") | map(select(length > 0))' "$output/errors.txt")
          images=$(if [ -s "$output/image-results.jsonl" ]; then jq -s '.' "$output/image-results.jsonl"; else echo '[]'; fi)
          jq -n \
            --argjson images "$images" \
            --argjson grype_has_errors "$grype_has_errors" \
            --arg license_rejected "$license_rejected" \
            --argjson operational_errors "$errors" \
            '{
              images: $images,
              grype_has_errors: $grype_has_errors,
              license_rejected: ($license_rejected == "true"),
              operational_errors: $operational_errors,
              tools: {
                syft: $ENV.SYFT_VERSION,
                grant: $ENV.GRANT_VERSION
              }
            }' > "$output/summary.json"
      - name: Upload image scan results
        if: always()
        uses: actions/upload-artifact@v7.0.1
        with:
          name: container-image-scan
          path: ${{ runner.temp }}/container-image-scan
          if-no-files-found: error
          retention-days: 14
sandbox:
  agent:
    sudo: false
timeout-minutes: 20
---

# Daily Container Image Security Scan

Review the Syft, Grype (via `gh aw compile --grype`), and Grant results in
`/tmp/gh-aw/agent/image-scan/`.

1. Read `summary.json` first.
2. Read `grype-compile.txt` for vulnerability findings reported by the builtin
   grype validator. Each line starting with `::error` indicates a critical or
   high severity vulnerability; lines starting with `::warning` indicate medium
   severity.
3. For each image with digest drift, vulnerabilities (`grype_has_errors: true` or
   entries in `grype-compile.txt`), rejected licenses, or operational errors,
   create one issue. Use the title
   `Container findings for <first 12 characters of index_digest>` so repeated
   findings for the same immutable image are deduplicated. If there are only
   global operational errors, create one `Container scan operational failure`
   issue.
4. If no image has findings and there are no global operational errors, call
   `noop`.
5. In each image issue, include:
   - the pinned image, current tag digest, and platform child digests;
   - scanner versions;
   - every vulnerability from `grype-compile.txt`, with severity, vulnerability
     ID, package, installed version, and fixed versions;
   - every rejected or unknown license shown in the
     `grant-check-<artifact_prefix>-*.json` files;
   - image digest drift, operational errors, and actionable remediation.
6. Keep the report factual and compact. Never omit lower-severity
   vulnerabilities.

The configured `create-issue` safe output is the only allowed write operation.
