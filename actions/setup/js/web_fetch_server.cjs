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
 *     - url        {string}  URL to fetch (http or https only)
 *     - max_length {number}  Maximum characters to return (default: 50000)
 *     - raw        {boolean} Return raw HTML instead of simplified text (default: false)
 *
 * Transport: MCP stdio (Content-Length framed JSON-RPC 2.0)
 */

const https = require("https");
const http = require("http");
const { createServer, registerTool, start } = require("./mcp_server_core.cjs");

const DEFAULT_MAX_LENGTH = 50000;
/** Maximum raw response bytes accepted from a single HTTP response (10 MiB). */
const MAX_RESPONSE_BYTES = 10 * 1024 * 1024;
/** Maximum number of HTTP redirects to follow. */
const MAX_REDIRECTS = 5;

/**
 * Return true if the URL uses http or https.
 * @param {string} url
 * @returns {boolean}
 */
function isHttpUrl(url) {
  return url.startsWith("http://") || url.startsWith("https://");
}

/**
 * Fetch the content of a URL and return it as text.
 * Only http and https URLs are accepted (SSRF guard).
 * @param {string} url
 * @param {number} [redirectsLeft]
 * @returns {Promise<string>}
 */
function fetchUrl(url, redirectsLeft) {
  if (redirectsLeft === undefined) {
    redirectsLeft = MAX_REDIRECTS;
  }
  if (!isHttpUrl(url)) {
    return Promise.reject(new Error("Unsupported protocol in URL: " + url));
  }
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
        // Follow redirects (301/302/307/308) with a bounded counter.
        if (res.statusCode && res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
          res.resume();
          if (redirectsLeft <= 0) {
            reject(new Error("Too many redirects"));
            return;
          }
          const redirectUrl = res.headers.location;
          // Only follow http/https redirects to prevent SSRF via file:// etc.
          if (!isHttpUrl(redirectUrl)) {
            reject(new Error("Redirect to non-HTTP URL blocked: " + redirectUrl));
            return;
          }
          resolve(fetchUrl(redirectUrl, redirectsLeft - 1));
          return;
        }
        let totalBytes = 0;
        const chunks = /** @type {Buffer[]} */ [];
        res.on("data", chunk => {
          totalBytes += chunk.length;
          if (totalBytes > MAX_RESPONSE_BYTES) {
            req.destroy();
            reject(new Error("Response too large (> " + MAX_RESPONSE_BYTES + " bytes)"));
            return;
          }
          chunks.push(chunk);
        });
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
        description: "URL to fetch (http or https only)",
      },
      max_length: {
        type: "number",
        description: "Maximum characters to return (default: " + DEFAULT_MAX_LENGTH + ")",
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

    if (!isHttpUrl(url)) {
      return {
        content: [{ type: "text", text: "Error: only http and https URLs are supported" }],
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
