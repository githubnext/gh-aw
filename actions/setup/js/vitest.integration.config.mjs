import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    environment: "node",
    globals: true,
    include: ["frontmatter_hash_github_api.test.cjs"],
    testTimeout: 30000,
    hookTimeout: 10000,
  },
});
