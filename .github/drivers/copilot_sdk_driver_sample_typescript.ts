import { readFileSync } from "node:fs";
import { CopilotClient, RuntimeConnection, approveAll } from "@github/copilot-sdk";

// Default timeout for a single sendAndWait call: 10 minutes.
// Override via the COPILOT_SDK_SEND_TIMEOUT_MS environment variable.
const DEFAULT_SEND_TIMEOUT_MS = 10 * 60 * 1000;

function readRequiredEnv(name: string): string {
  const value = process.env[name];
  if (!value) {
    throw new Error(`${name} is not set`);
  }
  return value;
}

function parseSendTimeoutMs(): number {
  const raw = process.env.COPILOT_SDK_SEND_TIMEOUT_MS;
  if (typeof raw === "string") {
    const trimmed = raw.trim();
    if (/^\d+$/.test(trimmed)) {
      const parsed = Number.parseInt(trimmed, 10);
      if (Number.isSafeInteger(parsed) && parsed > 0) {
        return parsed;
      }
    }
  }
  return DEFAULT_SEND_TIMEOUT_MS;
}

function extractAssistantContent(message: unknown): string {
  if (!message || typeof message !== "object") {
    return "";
  }

  const withData = message as { data?: { content?: unknown }; content?: unknown };
  if (typeof withData.data?.content === "string") {
    return withData.data.content;
  }
  if (typeof withData.content === "string") {
    return withData.content;
  }
  return "";
}

async function main(): Promise<void> {
  const promptPath = readRequiredEnv("GH_AW_PROMPT");
  const sdkUri = readRequiredEnv("COPILOT_SDK_URI");
  const connectionToken = readRequiredEnv("COPILOT_CONNECTION_TOKEN");
  const model = readRequiredEnv("COPILOT_MODEL");
  const prompt = readFileSync(promptPath, "utf8");

  const client = new CopilotClient({
    connection: RuntimeConnection.forUri(sdkUri, { connectionToken }),
    workingDirectory: process.env.GITHUB_WORKSPACE || process.cwd(),
  });

  let session;
  await client.start();
  try {
    session = await client.createSession({
      onPermissionRequest: approveAll,
      model,
    });

    const response = await session.sendAndWait({ prompt }, parseSendTimeoutMs());
    const content = extractAssistantContent(response);
    if (content) {
      process.stdout.write(content.endsWith("\n") ? content : `${content}\n`);
    }
  } finally {
    if (session) {
      await session.disconnect();
    }
    await client.stop();
  }
}

main().catch(error => {
  const message = error instanceof Error ? error.message : String(error);
  process.stderr.write(`[copilot-sdk-driver-sample-typescript] ${message}\n`);
  process.exit(1);
});
