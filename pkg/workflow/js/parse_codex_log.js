function parseCodexLogMain() {
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
        const content = fs.readFileSync(logFile, "utf8");
        const parsedLog = parseCodexLog(content);
        if (parsedLog) {
            core.summary.addRaw(parsedLog).write();
            core.info("Codex log parsed successfully");
        }
        else {
            core.error("Failed to parse Codex log");
        }
    }
    catch (error) {
        core.setFailed(error instanceof Error ? error : String(error));
    }
}
function parseCodexLog(logContent) {
    try {
        const lines = logContent.split("\n");
        let markdown = "## 🤖 Commands and Tools\n\n";
        const commandSummary = [];
        for (let i = 0; i < lines.length; i++) {
            const line = lines[i];
            if (line.includes("] tool ") && line.includes("(")) {
                const toolMatch = line.match(/\] tool ([^(]+)\(/);
                if (toolMatch) {
                    const toolName = toolMatch[1].trim();
                    let existing = commandSummary.find(cmd => cmd.command === toolName);
                    if (!existing) {
                        existing = { command: toolName, count: 0, details: [] };
                        commandSummary.push(existing);
                    }
                    existing.count++;
                    existing.details.push(line.trim());
                }
            }
            if (line.includes("exec:") || line.includes("$ ")) {
                const execMatch = line.match(/(?:exec:|[$])\s*(.+)/);
                if (execMatch) {
                    const command = execMatch[1].trim();
                    let existing = commandSummary.find(cmd => cmd.command === command);
                    if (!existing) {
                        existing = { command: command, count: 0, details: [] };
                        commandSummary.push(existing);
                    }
                    existing.count++;
                    existing.details.push(line.trim());
                }
            }
        }
        if (commandSummary.length > 0) {
            markdown += "### Command Summary\n\n";
            commandSummary.sort((a, b) => b.count - a.count);
            for (const cmd of commandSummary) {
                markdown += `- **${cmd.command}** (${cmd.count} time${cmd.count !== 1 ? 's' : ''})\n`;
            }
            markdown += "\n";
        }
        markdown += "### Execution Log\n\n";
        let currentSection = "";
        let inCodeBlock = false;
        for (let i = 0; i < lines.length; i++) {
            const line = lines[i];
            if (line.trim() === "") {
                if (inCodeBlock) {
                    markdown += "\n";
                }
                continue;
            }
            if (line.includes("] tool ") && line.includes("(")) {
                if (inCodeBlock) {
                    markdown += "```\n\n";
                    inCodeBlock = false;
                }
                const toolMatch = line.match(/\] tool ([^(]+)\(/);
                if (toolMatch) {
                    const toolName = toolMatch[1].trim();
                    if (currentSection !== toolName) {
                        currentSection = toolName;
                        markdown += `#### 🔧 Tool: ${toolName}\n\n`;
                    }
                    markdown += "```\n";
                    markdown += line + "\n";
                    inCodeBlock = true;
                }
            }
            else if (line.includes("exec:") || line.includes("$ ")) {
                if (inCodeBlock) {
                    markdown += "```\n\n";
                    inCodeBlock = false;
                }
                if (currentSection !== "shell") {
                    currentSection = "shell";
                    markdown += "#### 💻 Shell Commands\n\n";
                }
                markdown += "```bash\n";
                markdown += line + "\n";
                inCodeBlock = true;
            }
            else if (inCodeBlock) {
                markdown += line + "\n";
            }
            else {
                if (inCodeBlock) {
                    markdown += "```\n\n";
                    inCodeBlock = false;
                }
                markdown += `${line}\n\n`;
            }
        }
        if (inCodeBlock) {
            markdown += "```\n\n";
        }
        markdown += "---\n";
        markdown += "*Generated from Codex agent execution log*\n";
        return markdown;
    }
    catch (error) {
        core.error(`Error parsing log: ${error instanceof Error ? error.message : String(error)}`);
        return "";
    }
}
if (typeof module !== "undefined" && module.exports) {
    module.exports = {
        parseCodexLog,
    };
}
(async () => {
    parseCodexLogMain();
})();
