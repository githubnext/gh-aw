#!/usr/bin/env node
"use strict";

const fs = require("node:fs");

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

function buildSessionConfig(model, onPermissionRequest) {
  const config = {
    onPermissionRequest,
    model,
  };

  const providerBaseUrl = process.env.GH_AW_COPILOT_SDK_PROVIDER_BASE_URL;
  if (providerBaseUrl) {
    config.provider = { type: "openai", baseUrl: providerBaseUrl };
  }

  return config;
}

async function main() {
  const { CopilotClient, RuntimeConnection, approveAll } = require("@github/copilot-sdk");
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
    session = await client.createSession(buildSessionConfig(model, approveAll));
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

if (require.main === module) {
  main().catch(error => {
    process.stderr.write(`[copilot-sdk-driver-sample-node] ${error instanceof Error ? error.message : String(error)}\n`);
    process.exit(1);
  });
}

module.exports = {
  buildSessionConfig,
};
