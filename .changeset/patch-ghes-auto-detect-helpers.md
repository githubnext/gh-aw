---
"gh-aw": patch
---

Add GHES auto-detection helper functions for wizard configuration. This lays the foundation for auto-configuring GHES-specific settings in `gh aw add-wizard`:

- Add `isGHESInstance()` to detect GHES instances vs. public GitHub
- Add `getGHESAPIURL()` to get the GHES API URL for engine.api-target configuration
- Add `getGHESAllowedDomains()` to get GHES domains for network firewall configuration

These functions detect GHES instances by parsing the git origin remote URL and can be used by the wizard to:
1. Auto-populate `engine.api-target` for Copilot on GHES
2. Auto-add GHES domains to `network.allowed` for firewall configuration
3. Enable proper gh CLI host configuration via `GH_HOST` environment variable

Related to github/gh-aw#20875 and the GHES wizard auto-configuration requirements.
