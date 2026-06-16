import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    environment: "node",
    globals: true,
    include: ["frontmatter_hash_github_api.test.cjs", "set_issue_field_api_query.integration.test.cjs"],
    testTimeout: 30000,
    hookTimeout: 10000,
  },
});
