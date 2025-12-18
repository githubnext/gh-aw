// Tests for interpolate_prompt.cjs
import{describe,it,expect,vi,beforeEach,afterEach}from"vitest";import fs from"fs";import path from"path";import os from"os";import{fileURLToPath}from"url";const __filename=fileURLToPath(import.meta.url),__dirname=path.dirname(__filename),core={info:vi.fn(),setFailed:vi.fn()};global.core=core;
// Import the interpolation script
const interpolatePromptScript=fs.readFileSync(path.join(__dirname,"interpolate_prompt.cjs"),"utf8"),{isTruthy}=require("./is_truthy.cjs"),interpolateVariablesMatch=interpolatePromptScript.match(/function interpolateVariables\(content, variables\)\s*{[\s\S]*?return result;[\s\S]*?}/),renderMarkdownTemplateMatch=interpolatePromptScript.match(/function renderMarkdownTemplate\(markdown\)\s*{[\s\S]*?return result;[\s\S]*?}/);
// Import isTruthy from its own module for testing renderMarkdownTemplate
if(!interpolateVariablesMatch)throw new Error("Could not extract interpolateVariables function from interpolate_prompt.cjs");if(!renderMarkdownTemplateMatch)throw new Error("Could not extract renderMarkdownTemplate function from interpolate_prompt.cjs");
// eslint-disable-next-line no-eval
const interpolateVariables=eval(`(${interpolateVariablesMatch[0]})`),renderMarkdownTemplate=eval(`(${renderMarkdownTemplateMatch[0]})`);
// eslint-disable-next-line no-eval
describe("interpolate_prompt",()=>{describe("interpolateVariables",()=>{it("should interpolate single variable",()=>{const result=interpolateVariables("Repository: ${GH_AW_EXPR_TEST123}",{GH_AW_EXPR_TEST123:"github/test-repo"});expect(result).toBe("Repository: github/test-repo")}),it("should interpolate multiple variables",()=>{const result=interpolateVariables("Repo: ${GH_AW_EXPR_REPO}, Actor: ${GH_AW_EXPR_ACTOR}, Issue: ${GH_AW_EXPR_ISSUE}",{GH_AW_EXPR_REPO:"github/test-repo",GH_AW_EXPR_ACTOR:"testuser",GH_AW_EXPR_ISSUE:"123"});expect(result).toBe("Repo: github/test-repo, Actor: testuser, Issue: 123")}),it("should handle multiline content",()=>{const result=interpolateVariables("# Test Workflow\n\nRepository: ${GH_AW_EXPR_REPO}\nActor: ${GH_AW_EXPR_ACTOR}\n\nSome other content here.",{GH_AW_EXPR_REPO:"github/test-repo",GH_AW_EXPR_ACTOR:"testuser"});expect(result).toContain("Repository: github/test-repo"),expect(result).toContain("Actor: testuser")}),it("should handle empty variable values",()=>{const result=interpolateVariables("Value: ${GH_AW_EXPR_EMPTY}",{GH_AW_EXPR_EMPTY:""});expect(result).toBe("Value: ")}),it("should replace all occurrences of the same variable",()=>{const result=interpolateVariables("Repo: ${GH_AW_EXPR_REPO}, Same repo: ${GH_AW_EXPR_REPO}",{GH_AW_EXPR_REPO:"github/test-repo"});expect(result).toBe("Repo: github/test-repo, Same repo: github/test-repo")}),it("should not modify content without variables",()=>{const result=interpolateVariables("No variables here",{});expect(result).toBe("No variables here")}),it("should handle content with literal dollar signs",()=>{const result=interpolateVariables("Price: $100, Repo: ${GH_AW_EXPR_REPO}",{GH_AW_EXPR_REPO:"github/test-repo"});expect(result).toBe("Price: $100, Repo: github/test-repo")})}),describe("renderMarkdownTemplate",()=>{it("should keep content in truthy blocks",()=>{const output=renderMarkdownTemplate("{{#if true}}\nHello\n{{/if}}");expect(output).toBe("Hello\n")}),it("should remove content in falsy blocks",()=>{const output=renderMarkdownTemplate("{{#if false}}\nHello\n{{/if}}");expect(output).toBe("")}),it("should process multiple blocks",()=>{const output=renderMarkdownTemplate("{{#if true}}\nKeep this\n{{/if}}\n{{#if false}}\nRemove this\n{{/if}}");expect(output).toBe("Keep this\n")}),it("should handle nested content",()=>{const output=renderMarkdownTemplate("# Title\n\n{{#if true}}\n## Section 1\nThis should be kept.\n{{/if}}\n\n{{#if false}}\n## Section 2\nThis should be removed.\n{{/if}}\n\n## Section 3\nThis is always visible.");
// With empty line cleanup, we expect at most 2 consecutive newlines
expect(output).toBe("# Title\n\n## Section 1\nThis should be kept.\n\n## Section 3\nThis is always visible.")}),it("should leave content without conditionals unchanged",()=>{const input="# Normal Markdown\n\nNo conditionals here.",output=renderMarkdownTemplate(input);expect(output).toBe(input)}),it("should handle conditionals with various expressions",()=>{expect(renderMarkdownTemplate("{{#if 1}}\nKeep\n{{/if}}")).toBe("Keep\n"),expect(renderMarkdownTemplate("{{#if 0}}\nRemove\n{{/if}}")).toBe(""),expect(renderMarkdownTemplate("{{#if null}}\nRemove\n{{/if}}")).toBe(""),expect(renderMarkdownTemplate("{{#if undefined}}\nRemove\n{{/if}}")).toBe("")}),it("should preserve markdown formatting inside blocks",()=>{const output=renderMarkdownTemplate("{{#if true}}\n## Header\n- List item 1\n- List item 2\n\n```javascript\nconst x = 1;\n```\n{{/if}}");expect(output).toBe("## Header\n- List item 1\n- List item 2\n\n```javascript\nconst x = 1;\n```\n")}),it("should handle whitespace in conditionals",()=>{expect(renderMarkdownTemplate("{{#if   true  }}\nKeep\n{{/if}}")).toBe("Keep\n"),expect(renderMarkdownTemplate("{{#if\ttrue\t}}\nKeep\n{{/if}}")).toBe("Keep\n")}),it("should clean up multiple consecutive empty lines",()=>{
// When a false block is removed, it should not leave more than 2 consecutive newlines
const output=renderMarkdownTemplate("# Title\n\n{{#if false}}\n## Hidden Section\nThis should be removed.\n{{/if}}\n\n## Visible Section\nThis is always visible.");expect(output).toBe("# Title\n\n## Visible Section\nThis is always visible.")}),it("should collapse multiple false blocks without excessive empty lines",()=>{const output=renderMarkdownTemplate("Start\n\n{{#if false}}\nBlock 1\n{{/if}}\n\n{{#if false}}\nBlock 2\n{{/if}}\n\n{{#if false}}\nBlock 3\n{{/if}}\n\nEnd");
// Should not have more than 2 consecutive newlines anywhere
expect(output).not.toMatch(/\n{3,}/),expect(output).toContain("Start"),expect(output).toContain("End")})}),describe("combined interpolation and template rendering",()=>{it("should interpolate variables and then render templates",()=>{
// First interpolate
let result=interpolateVariables("Repo: ${GH_AW_EXPR_REPO}\n{{#if true}}\nShow this\n{{/if}}",{GH_AW_EXPR_REPO:"github/test-repo"});expect(result).toBe("Repo: github/test-repo\n{{#if true}}\nShow this\n{{/if}}"),
// Then render template
result=renderMarkdownTemplate(result),expect(result).toBe("Repo: github/test-repo\nShow this\n")}),it("should handle template conditionals that depend on interpolated values",()=>{
// First interpolate
let result=interpolateVariables("${GH_AW_EXPR_CONDITION}\n{{#if ${GH_AW_EXPR_CONDITION}}}\nShow this\n{{/if}}",{GH_AW_EXPR_CONDITION:"true"});expect(result).toBe("true\n{{#if true}}\nShow this\n{{/if}}"),
// Then render template
result=renderMarkdownTemplate(result),expect(result).toBe("true\nShow this\n")})}),describe("main function integration",()=>{let tmpDir,promptPath,originalEnv;beforeEach(()=>{
// Save original environment
// Save original environment
// Save original environment
// Save original environment
originalEnv={...process.env},
// Create a temporary directory
tmpDir=fs.mkdtempSync(path.join(os.tmpdir(),"interpolate-test-")),promptPath=path.join(tmpDir,"prompt.txt"),
// Set up environment
process.env.GH_AW_PROMPT=promptPath,
// Clear mocks
core.info.mockClear(),core.setFailed.mockClear()}),afterEach(()=>{
// Clean up
// Clean up
// Clean up
// Clean up
tmpDir&&fs.existsSync(tmpDir)&&fs.rmSync(tmpDir,{recursive:!0,force:!0}),
// Restore environment
process.env=originalEnv}),it("should fail when GH_AW_PROMPT is not set",()=>{delete process.env.GH_AW_PROMPT;
// Extract and execute main function
const mainMatch=interpolatePromptScript.match(/async function main\(\)\s*{[\s\S]*?^}/m);if(!mainMatch)throw new Error("Could not extract main function");
// eslint-disable-next-line no-eval
const main=eval(`(${mainMatch[0]})`);main(),expect(core.setFailed).toHaveBeenCalledWith("GH_AW_PROMPT environment variable is not set")})})});