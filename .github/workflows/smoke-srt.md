---
description: Smoke test workflow for Sandbox Runtime (SRT) - validates SRT functionality with Copilot
on:
  workflow_dispatch:
  pull_request:
    types: [labeled]
    names: ["test-srt"]
permissions:
  contents: read
  issues: read
name: Smoke SRT
engine: copilot
network:
  allowed:
    - defaults
    - github
    - "*.githubcopilot.com"
    - "example.com"
sandbox:
  type: sandbox-runtime
  config:
    network:
      allowedDomains:
        - "*.githubusercontent.com"
        - "*.github.com"
        - "*.githubcopilot.com"
        - "api.enterprise.githubcopilot.com"
        - "api.github.com"
        - "api.snapcraft.io"
        - "archive.ubuntu.com"
        - "azure.archive.ubuntu.com"
        - "codeload.github.com"
        - "crl.geotrust.com"
        - "crl.globalsign.com"
        - "crl.identrust.com"
        - "crl.sectigo.com"
        - "crl.thawte.com"
        - "crl.usertrust.com"
        - "crl.verisign.com"
        - "crl3.digicert.com"
        - "crl4.digicert.com"
        - "crls.ssl.com"
        - "github-cloud.githubusercontent.com"
        - "github-cloud.s3.amazonaws.com"
        - "github.com"
        - "json-schema.org"
        - "json.schemastore.org"
        - "keyserver.ubuntu.com"
        - "lfs.github.com"
        - "objects.githubusercontent.com"
        - "ocsp.digicert.com"
        - "ocsp.geotrust.com"
        - "ocsp.globalsign.com"
        - "ocsp.identrust.com"
        - "ocsp.sectigo.com"
        - "ocsp.ssl.com"
        - "ocsp.thawte.com"
        - "ocsp.usertrust.com"
        - "ocsp.verisign.com"
        - "packagecloud.io"
        - "packages.cloud.google.com"
        - "packages.microsoft.com"
        - "ppa.launchpad.net"
        - "raw.githubusercontent.com"
        - "registry.npmjs.org"
        - "registry.npmjs.com"
        - "registry.bower.io"
        - "registry.yarnpkg.com"
        - "repo.yarnpkg.com"
        - "api.npms.io"
        - "bun.sh"
        - "deb.nodesource.com"
        - "deno.land"
        - "get.pnpm.io"
        - "nodejs.org"
        - "npm.pkg.github.com"
        - "npmjs.com"
        - "npmjs.org"
        - "www.npmjs.com"
        - "www.npmjs.org"
        - "yarnpkg.com"
        - "skimdb.npmjs.com"
        - "s.symcb.com"
        - "s.symcd.com"
        - "security.ubuntu.com"
        - "ts-crl.ws.symantec.com"
        - "ts-ocsp.ws.symantec.com"
        - "example.com"
      deniedDomains: []
      allowUnixSockets:
        - "/var/run/docker.sock"
      allowLocalBinding: true
      allowAllUnixSockets: true
    filesystem:
      denyRead: []
      allowWrite:
        - "."
        - "/tmp"
        - "/home/runner/.copilot"
        - "/home/runner"
      denyWrite: []
    enableWeakerNestedSandbox: true
tools:
  bash:
  github:
timeout-minutes: 5
strict: true
---

**IMPORTANT: Keep all outputs extremely short and concise. Use single-line responses where possible. No verbose explanations.**

Test the Sandbox Runtime (SRT) integration:

1. Run `echo "Hello from SRT!"` using bash
2. Check the current directory with `pwd`
3. List files in the current directory with `ls -la`

## Output

**PIRATE STYLE**: Write your output like a swashbuckling sea captain reporting to the crew. Use nautical terms and pirate speak.

Output a **very brief** summary (max 3-5 lines) in pirate style:
- Use pirate exclamations: "Ahoy!", "Arrr!", "Shiver me timbers!"
- Nautical metaphors: "The ship be sailin' smooth!", "All hands on deck!"
- Use ✅ or ❌ as "treasure marks"
- End with a captain's verdict: "Yo ho ho! The voyage be a SUCCESS!" or "Blimey! We've hit rough waters!"

Example tone: "Arrr! ✅ The echo command bellowed across the seven seas! The SRT treasure be secured, me hearties!"
