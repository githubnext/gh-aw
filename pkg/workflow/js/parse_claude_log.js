function parseClaudeLogMain() {
    const fs = require("fs");
    try {
        const logFile = process.env.GITHUB_AW_AGENT_OUTPUT;
        if (!logFile) {
            core.info("No agent log file specified");
            return;
        }
        if (!fs.existsSync(logFile)) {
            core.info(`Log file not found: ${logFile}`);
            return;
        }
        const logContent = fs.readFileSync(logFile, "utf8");
        const result = parseClaudeLog(logContent);
        core.summary.addRaw(result.markdown).write();
        if (result.mcpFailures && result.mcpFailures.length > 0) {
            const failedServers = result.mcpFailures.join(", ");
            core.setFailed(`MCP server(s) failed to launch: ${failedServers}`);
        }
    }
    catch (error) {
        const errorMessage = error instanceof Error ? error.message : String(error);
        core.setFailed(errorMessage);
    }
}
function parseClaudeLog(logContent) {
    try {
        let logEntries;
        try {
            logEntries = JSON.parse(logContent);
            if (!Array.isArray(logEntries)) {
                throw new Error("Not a JSON array");
            }
        }
        catch (jsonArrayError) {
            logEntries = [];
            const lines = logContent.split("\n");
            for (const line of lines) {
                const trimmedLine = line.trim();
                if (trimmedLine === "") {
                    continue;
                }
                try {
                    const entry = JSON.parse(trimmedLine);
                    if (typeof entry === "object" && entry !== null) {
                        logEntries.push(entry);
                    }
                }
                catch (jsonError) {
                    if (trimmedLine.includes("level=") ||
                        trimmedLine.includes("msg=") ||
                        trimmedLine.includes("time=")) {
                        const entry = {};
                        const parts = trimmedLine.split(/\s+/);
                        for (const part of parts) {
                            const [key, ...valueParts] = part.split("=");
                            if (valueParts.length > 0) {
                                const value = valueParts.join("=").replace(/^"(.*)"$/, "$1");
                                entry[key] = value;
                            }
                        }
                        logEntries.push(entry);
                    }
                    else {
                        logEntries.push({
                            level: "debug",
                            msg: trimmedLine,
                            time: new Date().toISOString(),
                        });
                    }
                }
            }
        }
        let markdown = "## 🤖 Claude Agent Execution Log\n\n";
        const mcpFailures = [];
        const toolCalls = {};
        const errorMessages = [];
        const mcpServers = [];
        for (const entry of logEntries) {
            const msg = entry.msg || "";
            const level = entry.level || "info";
            if (msg.includes("launching MCP server") || msg.includes("MCP server started")) {
                const serverMatch = msg.match(/server[:\s]+([^\s,]+)/i);
                if (serverMatch && !mcpServers.includes(serverMatch[1])) {
                    mcpServers.push(serverMatch[1]);
                }
            }
            if (level === "error" &&
                (msg.includes("MCP server") || msg.includes("MCP") || msg.includes("failed to launch"))) {
                const serverMatch = msg.match(/server[:\s]+([^\s,]+)/i);
                if (serverMatch) {
                    mcpFailures.push(serverMatch[1]);
                }
                else {
                    mcpFailures.push("unknown server");
                }
            }
            if (msg.includes("calling tool") || msg.includes("tool call")) {
                const toolMatch = msg.match(/tool[:\s]+([^\s,()]+)/i);
                if (toolMatch) {
                    const tool = toolMatch[1];
                    toolCalls[tool] = (toolCalls[tool] || 0) + 1;
                }
            }
            if (level === "error" && !errorMessages.includes(msg)) {
                errorMessages.push(msg);
            }
        }
        if (mcpServers.length > 0) {
            markdown += "### 🔧 MCP Servers\n\n";
            for (const server of mcpServers) {
                const status = mcpFailures.includes(server) ? "❌ Failed" : "✅ Running";
                markdown += `- **${server}**: ${status}\n`;
            }
            markdown += "\n";
        }
        if (Object.keys(toolCalls).length > 0) {
            markdown += "### 🛠️ Tool Usage Summary\n\n";
            const sortedTools = Object.entries(toolCalls).sort((a, b) => b[1] - a[1]);
            for (const [tool, count] of sortedTools) {
                markdown += `- **${tool}**: ${count} call${count !== 1 ? 's' : ''}\n`;
            }
            markdown += "\n";
        }
        if (errorMessages.length > 0) {
            markdown += "### ❌ Errors\n\n";
            for (const error of errorMessages) {
                markdown += `- ${error}\n`;
            }
            markdown += "\n";
        }
        markdown += "### 📋 Execution Details\n\n";
        let currentSection = "";
        let sectionCount = 0;
        for (let i = 0; i < logEntries.length; i++) {
            const entry = logEntries[i];
            const msg = entry.msg || "";
            const level = entry.level || "info";
            const time = entry.time || "";
            if (msg.includes("calling tool") || msg.includes("tool call")) {
                const toolMatch = msg.match(/tool[:\s]+([^\s,()]+)/i);
                if (toolMatch) {
                    const tool = toolMatch[1];
                    if (currentSection !== tool) {
                        currentSection = tool;
                        sectionCount++;
                        markdown += `#### ${sectionCount}. 🔧 Tool: ${tool}\n\n`;
                    }
                }
            }
            else if (msg.includes("executing") || msg.includes("running")) {
                if (currentSection !== "execution") {
                    currentSection = "execution";
                    sectionCount++;
                    markdown += `#### ${sectionCount}. ⚡ Execution\n\n`;
                }
            }
            const levelEmoji = getLevelEmoji(level);
            const timeStr = time ? ` \`${time}\`` : "";
            if (level === "error") {
                markdown += `${levelEmoji}${timeStr} **ERROR**: ${msg}\n\n`;
            }
            else if (level === "warn") {
                markdown += `${levelEmoji}${timeStr} **WARNING**: ${msg}\n\n`;
            }
            else if (msg.length > 200) {
                markdown += `${levelEmoji}${timeStr} ${msg.substring(0, 200)}...\n\n`;
            }
            else {
                markdown += `${levelEmoji}${timeStr} ${msg}\n\n`;
            }
        }
        markdown += "---\n";
        markdown += `*Log parsed at ${new Date().toISOString()}*\n`;
        markdown += `*Total entries: ${logEntries.length}*\n`;
        return {
            markdown,
            mcpFailures,
        };
    }
    catch (error) {
        core.error(`Error parsing log: ${error instanceof Error ? error.message : String(error)}`);
        return {
            markdown: `## ❌ Log Parsing Error\n\nFailed to parse Claude log: ${error instanceof Error ? error.message : String(error)}\n`,
            mcpFailures: [],
        };
    }
}
function getLevelEmoji(level) {
    switch (level.toLowerCase()) {
        case "error":
            return "❌ ";
        case "warn":
        case "warning":
            return "⚠️ ";
        case "info":
            return "ℹ️ ";
        case "debug":
            return "🔍 ";
        default:
            return "📝 ";
    }
}
if (typeof module !== "undefined" && module.exports) {
    module.exports = {
        parseClaudeLog,
        getLevelEmoji,
    };
}
(async () => {
    parseClaudeLogMain();
})();
