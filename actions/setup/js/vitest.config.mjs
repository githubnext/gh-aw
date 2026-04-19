import { configDefaults, defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    environment: "node",
    globals: true,
    include: ["**/*.test.{js,cjs}"],
    exclude: ["**/frontmatter_hash_github_api.test.cjs", ...configDefaults.exclude],
    testTimeout: 10000,
    hookTimeout: 10000,
    coverage: {
      provider: "v8",
      reporter: ["text", "html"],
      include: ["**/*.cjs"],
      exclude: ["**/*.test.{js,cjs}"],
    },
  },
});
