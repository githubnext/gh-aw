#!/usr/bin/env node
"use strict";

const fs = require("node:fs");
const { CopilotClient, RuntimeConnection, approveAll } = require("@github/copilot-sdk");

function readRequiredEnv(name) {
  const value = process.env[name];
  if (!value) {
    throw new Error(`${name} is not set`);
  }
  return value;
}

function extractAssistantContent(message) {
  if (!message || typeof message !== "object") {
    return "";
  }
  const data = message.data;
  if (data && typeof data.content === "string") {
    return data.content;
  }
  if (typeof message.content === "string") {
    return message.content;
  }
  return "";
}

async function main() {
  const promptPath = readRequiredEnv("GH_AW_PROMPT");
  const sdkUri = readRequiredEnv("COPILOT_SDK_URI");
  const connectionToken = readRequiredEnv("COPILOT_CONNECTION_TOKEN");
  const model = readRequiredEnv("COPILOT_MODEL");
  const prompt = fs.readFileSync(promptPath, "utf8");

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
    const response = await session.sendAndWait({ prompt });
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
  process.stderr.write(`[copilot-sdk-driver-sample-node] ${error instanceof Error ? error.message : String(error)}\n`);
  process.exit(1);
});
