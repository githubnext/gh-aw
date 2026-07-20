import { cpSync, mkdirSync } from "fs";
import path from "path";

export function syncRuntimePromptTemplates(metaUrl) {
  const promptsDir = new URL("../md", metaUrl).pathname;
  const runtimePromptsDir = path.join(process.env.RUNNER_TEMP || "/tmp", "gh-aw", "prompts");
  mkdirSync(runtimePromptsDir, { recursive: true });
  cpSync(promptsDir, runtimePromptsDir, { recursive: true });
  return { promptsDir, runtimePromptsDir };
}
