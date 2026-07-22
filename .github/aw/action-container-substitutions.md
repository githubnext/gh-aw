---
description: How to configure action and container image substitutions in aw.json for private-cloud and air-gapped enterprise environments.
---

# Action and Container Substitutions

Use `action_pins` and `container_pins` in `.github/workflows/aw.json` to redirect compiled action and container image references to internal mirrors. This is the recommended approach for enterprises that operate GitHub Actions runners in private clouds or air-gapped environments where public registries are unreachable.

These substitutions are configured at the repository level in `aw.json` — not in individual workflow frontmatter — so a single configuration file controls all redirects across every workflow in the repository.

## Action substitutions (`action_pins`)

`action_pins` maps `owner/repo@ref` source keys to replacement `owner/repo@ref` values. The mapping is applied before the standard pin-resolution pipeline (cache → GitHub API → embedded pins), so the full resolution chain operates on the mapped target.

```json title=".github/workflows/aw.json"
{
  "action_pins": {
    "actions/checkout@v4": "acme-corp/checkout-mirror@v4",
    "actions/setup-node@v4": "acme-corp/setup-node-mirror@v4"
  }
}
```

**Key requirements:**
- Keys and values must use the format `owner/repo@ref` (validated at schema load time).
- Each source version must be mapped individually — wildcard or prefix matching is not supported.
- The replacement target must itself be resolvable by the pin machinery (dynamic lookup, embedded pins, or local cache). If the mirror repo is not in the embedded pin table and dynamic resolution is unavailable, resolution fails.

A console message is emitted once per mapped key during compilation:

```
ℹ Action pin mapping applied: actions/checkout@v4 → acme-corp/checkout-mirror@v4
```

## Container substitutions (`container_pins`)

`container_pins` maps source container image references (e.g. `ghcr.io/owner/image:tag`) to replacement image targets. The mapping is applied before digest-pin resolution, so a privately mirrored image can be used in place of the public source.

Each value is an object with separate `image` (ref name) and `digest` (SHA-256) fields so that each component is validated independently:

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
- `image` must be a valid image reference without a digest component (e.g. `registry.acme.com/image:tag`).
- `digest` must be a full SHA-256 digest in `sha256:<64 lowercase hex characters>` form.

A console message is emitted once per mapped key during compilation:

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

- Substitutions are applied at compile time and are baked into the generated `.lock.yml` files.
- Neither `action_pins` nor `container_pins` is supported in workflow frontmatter; both are repository-level settings in `aw.json`.
- Re-run `gh aw compile` after modifying `aw.json` to regenerate all affected lock files.
- See [Self-Hosted Runners](/gh-aw/reference/self-hosted-runners/#action-and-container-substitutions-awjson) for full documentation.
