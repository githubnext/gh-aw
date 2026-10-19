---
runtimes:
  go:
    version: "1.26"
tools:
  bash:
    - go version
    - go env
    - go list
    - go doc
    - go build
    - go test
    - go vet
    - gofmt
network:
  allowed:
    - go
sandbox:
  agent:
    mounts:
      - ${{ env.GH_AW_TOOL_PROFILE_GO_DIR }}/bin:${{ env.GH_AW_TOOL_PROFILE_GO_DIR }}/bin:ro
      - ${{ env.GH_AW_TOOL_PROFILE_GO_DIR }}/cache:${{ env.GH_AW_TOOL_PROFILE_GO_DIR }}/cache:rw
pre-agent-steps:
  - name: Setup Go tool profile
    run: |
      set -euo pipefail
      go_root="$(go env GOROOT)"
      case "$go_root" in
        "${RUNNER_TOOL_CACHE:?RUNNER_TOOL_CACHE must be set}"/go/*) ;;
        *)
          echo "Go tool profile requires a Go runtime under RUNNER_TOOL_CACHE; check runtimes.go or actions/setup-go." >&2
          exit 1
          ;;
      esac
      if [ ! -x "$go_root/bin/go" ]; then
        echo "Go tool profile has no executable bin/go in its mounted toolchain." >&2
        exit 1
      fi
      profile="${RUNNER_TEMP:?RUNNER_TEMP must be set}/gh-aw/tool-profiles/go"
      mkdir -p "$profile/bin" "$profile/cache/bin" "$profile/cache/build" "$profile/cache/mod" "$profile/cache/home"
      export GOBIN="$profile/cache/bin"
      export GOPATH="$profile/cache/go"
      export GOCACHE="$profile/cache/build"
      export GOMODCACHE="$profile/cache/mod"
      if [ -f "$GITHUB_WORKSPACE/go.mod" ] || [ -f "$GITHUB_WORKSPACE/go.work" ]; then
        (cd "$GITHUB_WORKSPACE" && GOTOOLCHAIN=local go mod download)
      fi
      GOBIN="$profile/bin" CGO_ENABLED=0 GOTOOLCHAIN=local go install golang.org/x/tools/gopls@v0.23.0
      chmod 555 "$profile/bin/gopls" "$profile/bin"
      "$profile/bin/gopls" version
      probe="$(mktemp -d)"
      trap 'rm -f "$probe/go.mod" "$probe/profile.go"; rmdir "$probe"' EXIT
      printf 'module example.com/gh-aw-profile-readiness\n\ngo 1.26\n' > "$probe/go.mod"
      printf 'package readiness\n\nfunc GhAwGoProfileReady() {}\n' > "$probe/profile.go"
      symbols="$(cd "$probe" && GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local timeout 60 "$profile/bin/gopls" -remote= symbols profile.go)"
      case "$symbols" in
        *"GhAwGoProfileReady Function"*) ;;
        *)
          echo "Go tool profile failed semantic readiness: gopls did not return the probe symbol." >&2
          exit 1
          ;;
      esac
      {
        printf 'GH_AW_TOOL_PROFILE_GO_DIR=%s\n' "$profile"
        printf 'GH_AW_TOOL_PROFILE_GO_ROOT=%s\n' "$go_root"
        printf 'GH_AW_TOOL_PROFILE_GO_PATH=%s/bin:%s/bin\n' "$profile" "$go_root"
        printf 'GOBIN=%s\n' "$GOBIN"
        printf 'GOPATH=%s\nGOCACHE=%s\nGOMODCACHE=%s\n' "$GOPATH" "$GOCACHE" "$GOMODCACHE"
        printf 'GOTELEMETRY=off\n'
      } >> "$GITHUB_ENV"
      printf '%s\n' "$profile/bin" "$GOBIN" >> "$GITHUB_PATH"
mcp-servers:
  gopls:
    container: node:lts-alpine
    args:
      - --workdir
      - \${GITHUB_WORKSPACE}
    entrypoint: \${GH_AW_TOOL_PROFILE_GO_DIR}/bin/gopls
    entrypointArgs:
      - mcp
    allowed:
      - go_workspace
      - go_search
      - go_file_context
      - go_package_api
      - go_diagnostics
      - go_symbol_references
    mounts:
      - \${GITHUB_WORKSPACE}:\${GITHUB_WORKSPACE}:ro
      - \${RUNNER_TOOL_CACHE}:\${RUNNER_TOOL_CACHE}:ro
      - \${GH_AW_TOOL_PROFILE_GO_DIR}/bin:\${GH_AW_TOOL_PROFILE_GO_DIR}/bin:ro
      - \${GH_AW_TOOL_PROFILE_GO_DIR}/cache:\${GH_AW_TOOL_PROFILE_GO_DIR}/cache:rw
    env:
      GITHUB_WORKSPACE: ${{ github.workspace }}
      GH_AW_TOOL_PROFILE_GO_DIR: ${{ env.GH_AW_TOOL_PROFILE_GO_DIR }}
      GOROOT: ${{ env.GH_AW_TOOL_PROFILE_GO_ROOT }}
      PATH: ${{ env.GH_AW_TOOL_PROFILE_GO_PATH }}:/usr/local/bin:/usr/bin:/bin
      GOPATH: ${{ env.GH_AW_TOOL_PROFILE_GO_DIR }}/cache/go
      GOCACHE: ${{ env.GH_AW_TOOL_PROFILE_GO_DIR }}/cache/build
      GOMODCACHE: ${{ env.GH_AW_TOOL_PROFILE_GO_DIR }}/cache/mod
      GOPROXY: off
      GOSUMDB: off
      HOME: ${{ env.GH_AW_TOOL_PROFILE_GO_DIR }}/cache/home
      CGO_ENABLED: "0"
      GOTOOLCHAIN: local
      GOTELEMETRY: "off"
---

## Go tool profile

Go, gofmt, and gopls are installed before the agent starts. Use the gopls MCP
tools for workspace discovery, symbol search, references, package APIs, file
context, and diagnostics. Run Go builds, tests, vet, and formatting through the
agent's sandboxed shell, not through a host-side MCP command runner.

The Go shell commands are matched by command prefix, not by argument. In
particular, `go build`, `go test`, and `go vet` can execute arbitrary programs
through supported flags or test code. Treat this profile as trusted-code
execution inside the agent sandbox; its shell list is not an argument-level
security boundary.

The gopls sidecar has read-only access to the workspace and Go toolchain, and
writes only to its profile cache. Its analysis disables cgo and automatic
toolchain downloads. The profile preloads Go module dependencies when a root
module or workspace is present, then disables Go proxy and checksum-database
lookups in the sidecar so analysis cannot fetch missing modules dynamically.
