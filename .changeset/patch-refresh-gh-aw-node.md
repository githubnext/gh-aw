---
"gh-aw": patch
---

Fix broken brace-expansion patch in `gh-aw-node` Dockerfile (GHSA-mh99-v99m-4gvg): the previous `npm install --prefix "$(npm root -g)/npm"` approach failed silently because it reads npm's own private `package.json` (which references `@npmcli/docs`, a private package not on the public registry). Replace it with a temp-directory overlay: install brace-expansion ≥5.0.8, tar ≥7.5.22, and undici ≥6.27.0 into a fresh prefix with no package.json, then copy the patched modules into npm's bundled `node_modules`. Also add a push trigger to `publish-safe-outputs-node.yml` on `main` for the Dockerfile path so the image rebuilds automatically on merge.
