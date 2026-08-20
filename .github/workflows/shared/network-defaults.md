---
network:
  allowed:
    - defaults
---

<!--
## Network Defaults

This shared workflow grants access to the `defaults` network ecosystem identifier
(certificate authorities, Ubuntu package verification, JSON schema, and other basic
infrastructure domains required by most workflows).

### Usage

```yaml
imports:
  - shared/network-defaults.md
```

Consumers that need additional domains can still declare their own `network.allowed`
list alongside this import; the resulting domain lists are merged (union), so there is
no need to repeat `defaults` locally.
-->
