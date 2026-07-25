---
description: How to configure action and container image substitutions in aw.json for private-cloud and air-gapped enterprise environments.
---

# Action and Container Substitutions

Use `action_pins` and `container_pins` in `.github/workflows/aw.json` to redirect compiled action and container image references to internal mirrors — for private-cloud or air-gapped runners where public registries are unreachable.

These are repository-level settings in `aw.json`, not workflow frontmatter, so one file controls all redirects across every workflow.

## Action substitutions (`action_pins`)

`action_pins` maps `owner/repo@ref` source keys to replacement `owner/repo@ref` values. Applied before the pin-resolution pipeline (cache → GitHub API → embedded pins), so the full chain operates on the mapped target.

```json title=".github/workflows/aw.json"
{
  "action_pins": {
    "actions/checkout@v4": "acme-corp/checkout-mirror@v4",
    "actions/setup-node@v4": "acme-corp/setup-node-mirror@v4"
  }
}
```

**Key requirements:**
- Keys and values must use format `owner/repo@ref` (validated at schema load time).
- Map each source version individually — no wildcard or prefix matching.
- The replacement target must itself be resolvable by the pin machinery (dynamic lookup, embedded pins, or local cache); otherwise resolution fails.

One console message per mapped key at compile time:

```
ℹ Action pin mapping applied: actions/checkout@v4 → acme-corp/checkout-mirror@v4
```

## Container substitutions (`container_pins`)

`container_pins` maps source container image references (e.g. `ghcr.io/owner/image:tag`) to replacement targets. Applied before digest-pin resolution, so a mirrored image can replace the public source.

Each value is an object with separate `image` (ref name) and `digest` (SHA-256) fields, validated independently:

```json title=".github/workflows/aw.json"
{
  "container_pins": {
    "ghcr.io/actions/runner:latest": {
      "image": "registry.acme.com/runner:latest",
      "digest": "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
    },
    "node:lts-alpine": {
      "image": "registry.acme.com/node:lts-alpine",
      "digest": "sha256:abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
    }
  }
}
```

**Key requirements:**
- Keys are source image references as they appear in compiled workflows (e.g. `image:tag`, `registry/image:tag`). Digest-pinned source keys are not supported.
- `image` must be a valid reference without a digest component (e.g. `registry.acme.com/image:tag`).
- `digest` must be a full SHA-256 digest in `sha256:<64 lowercase hex chars>` form.

One console message per mapped key at compile time:

```
ℹ Container pin mapping applied: ghcr.io/actions/runner:latest → registry.acme.com/runner:latest@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
```

## Combined example

```json title=".github/workflows/aw.json"
{
  "action_pins": {
    "actions/checkout@v4": "acme-corp/checkout-mirror@v4",
    "actions/setup-node@v4": "acme-corp/setup-node-mirror@v4"
  },
  "container_pins": {
    "ghcr.io/github/gh-aw-firewall:0.27.22": {
      "image": "registry.acme.com/gh-aw-firewall:0.27.22",
      "digest": "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
    },
    "node:lts-alpine": {
      "image": "registry.acme.com/node:lts-alpine",
      "digest": "sha256:abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
    }
  }
}
```

## Notes

- Substitutions apply at compile time and are baked into the generated `.lock.yml` files.
- Neither `action_pins` nor `container_pins` works in workflow frontmatter; both are `aw.json` repository-level settings.
- Re-run `gh aw compile` after modifying `aw.json` to regenerate affected lock files.
- See [Self-Hosted Runners](/gh-aw/reference/self-hosted-runners/#action-and-container-substitutions-awjson) for full docs.
