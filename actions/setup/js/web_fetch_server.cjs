// @ts-check
"use strict";

/**
 * web_fetch_server.cjs
 *
 * Minimal MCP stdio server that provides a `fetch` tool for HTTP content retrieval.
 *
 * Intended for use with the Codex engine where web-fetch is not a built-in MCP tool.
 * Runs inside a node:lts-alpine Docker container managed by the MCP gateway.
 *
 * MCP tool provided:
 *   fetch(url, [max_length], [raw])
 *     - url        {string}  URL to fetch (http or https)
 *     - max_length {number}  Maximum characters to return (default: 50000)
 *     - raw        {boolean} Return raw HTML instead of simplified markdown (default: false)
 *
 * Transport: MCP stdio (Content-Length framed JSON-RPC 2.0)
 */

const https = require("https");
const http = require("http");
const { createServer, registerTool, start } = require("./mcp_server_core.cjs");

const DEFAULT_MAX_LENGTH = 50000;

/**
 * Fetch the content of a URL and return it as text.
 * @param {string} url
 * @returns {Promise<string>}
 */
function fetchUrl(url) {
  return new Promise((resolve, reject) => {
    const protocol = url.startsWith("https://") ? https : http;
    const req = protocol.get(
      url,
      {
        headers: {
          "User-Agent": "gh-aw/web-fetch-mcp",
          Accept: "text/html,text/plain,application/json,*/*",
        },
        timeout: 30000,
      },
      res => {
        // Follow up to 5 redirects
        if (res.statusCode && res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
          const redirectUrl = res.headers.location;
          res.resume();
          resolve(fetchUrl(redirectUrl));
          return;
        }
        const chunks = /** @type {Buffer[]} */ [];
        res.on("data", chunk => chunks.push(chunk));
        res.on("end", () => resolve(Buffer.concat(chunks).toString("utf8")));
        res.on("error", reject);
      }
    );
    req.on("error", reject);
    req.on("timeout", () => {
      req.destroy();
      reject(new Error("Request timed out"));
    });
  });
}

/** @param {string} html */
function htmlToText(html) {
  return (
    html
      // Remove <script> and <style> blocks
      .replace(/<script\b[^>]*>[\s\S]*?<\/script>/gi, "")
      .replace(/<style\b[^>]*>[\s\S]*?<\/style>/gi, "")
      // Strip all remaining HTML tags
      .replace(/<[^>]+>/g, " ")
      // Decode common HTML entities
      .replace(/&amp;/g, "&")
      .replace(/&lt;/g, "<")
      .replace(/&gt;/g, ">")
      .replace(/&quot;/g, '"')
      .replace(/&#39;/g, "'")
      .replace(/&nbsp;/g, " ")
      // Collapse whitespace
      .replace(/\s+/g, " ")
      .trim()
  );
}

const server = createServer({ name: "web-fetch", version: "1.0.0" });

registerTool(server, {
  name: "fetch",
  description: "Fetch content from a URL and return the page text",
  inputSchema: {
    type: "object",
    properties: {
      url: {
        type: "string",
        description: "URL to fetch (http or https)",
      },
      max_length: {
        type: "number",
        description: `Maximum characters to return (default: ${DEFAULT_MAX_LENGTH})`,
      },
      raw: {
        type: "boolean",
        description: "Return raw HTML instead of simplified text (default: false)",
      },
    },
    required: ["url"],
  },
  /** @param {{ url: string, max_length?: number, raw?: boolean }} args */
  handler: async args => {
    const { url, max_length = DEFAULT_MAX_LENGTH, raw = false } = args;

    if (!url || typeof url !== "string") {
      return {
        content: [{ type: "text", text: "Error: url parameter is required and must be a string" }],
        isError: true,
      };
    }

    const rawContent = await fetchUrl(url);
    const content = raw ? rawContent : htmlToText(rawContent);
    const truncated = content.length > max_length ? content.slice(0, max_length) + "\n[content truncated]" : content;
    return { content: [{ type: "text", text: truncated }] };
  },
});

start(server);
