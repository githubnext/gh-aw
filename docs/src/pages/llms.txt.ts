import type { APIRoute } from "astro";
import { getAwPrompts } from "./_aw-prompts.js";

export const prerender = true;

export const GET: APIRoute = () => {
  const prompts = getAwPrompts();

  const lines = [
    "# GitHub Agentic Workflows",
    "",
    "GitHub Agentic Workflows (gh-aw) is a GitHub CLI extension for creating, compiling, and running AI-powered workflows in GitHub Actions.",
    "",
    "> Use this index to find the project documentation, guides, reference material, and agent-optimised prompt files.",
    "",
    "## Documentation",
    "",
    "- [Documentation home](https://github.github.com/gh-aw/): Learn about GitHub Agentic Workflows.",
    "- [Quickstart](https://github.github.com/gh-aw/setup/quick-start/): Install gh-aw and run an agentic workflow.",
    "- [Create a workflow](https://github.github.com/gh-aw/setup/creating-workflows/): Author and compile a workflow.",
    "",
    "## Guides",
    "",
    "- [Agentic authoring](https://github.github.com/gh-aw/guides/agentic-authoring/): Use agents to create workflows.",
    "- [Reusing workflows](https://github.github.com/gh-aw/guides/reusing-workflows/): Share and update reusable workflows.",
    "- [Using gh-aw at scale](https://github.github.com/gh-aw/guides/using-at-scale/): Manage agentic workflows across repositories.",
    "",
    "## Reference",
    "",
    "- [Authentication](https://github.github.com/gh-aw/reference/auth/): Configure tokens, secrets, and GitHub Apps.",
    "- [Editors](https://github.github.com/gh-aw/reference/editors/): Configure editor support for workflow authoring.",
    "- [Triggering CI](https://github.github.com/gh-aw/reference/triggering-ci/): Trigger CI safely from agentic workflows.",
    "",
    "## Agent Prompts",
    "",
    ...prompts.map(({ file, description, rawUrl }) => {
      const label = file.replace(/\.md$/, "");
      return description ? `- [${label}](${rawUrl}): ${description}` : `- [${label}](${rawUrl})`;
    }),
  ];

  return new Response(`${lines.join("\n")}\n`, {
    headers: { "Content-Type": "text/plain; charset=utf-8" },
  });
};
