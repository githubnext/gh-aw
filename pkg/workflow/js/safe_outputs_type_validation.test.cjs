import{describe,it,expect}from"vitest";import fs from"fs";import path from"path";
/**
 * Test suite to validate that all safe output JavaScript files
 * use underscores in their type filtering, not dashes.
 *
 * Example:
 *   ✓ item.type === "create_issue"
 *   ✗ item.type === "create_issue"
 */describe("Safe Output Type Validation",()=>{
// Test each file
// Test each file
Object.entries({"create_issue.cjs":"create_issue","add_comment.cjs":"add_comment","update_issue.cjs":"update_issue","create_pr_review_comment.cjs":"create_pull_request_review_comment","add_labels.cjs":"add_labels","create_code_scanning_alert.cjs":"create_code_scanning_alert","upload_assets.cjs":"upload_asset","create_discussion.cjs":"create_discussion","push_to_pull_request_branch.cjs":"push_to_pull_request_branch","create_pull_request.cjs":"create_pull_request"}).forEach(([fileName,expectedType])=>{it(`should use underscores in type filter for ${fileName}`,()=>{const filePath=path.join(process.cwd(),fileName),content=fs.readFileSync(filePath,"utf8"),hasUnderscoreType=content.includes(`"${expectedType}"`);expect(hasUnderscoreType).toBe(!0);
// Create the dash version of the type
const dashType=expectedType.replace(/_/g,"-"),hasDashType=new RegExp(`item\\.type\\s*===\\s*["']${dashType.replace(/[.*+?^${}()|[\]\\]/g,"\\$&")}["']`).test(content);
// Check that the dash version is NOT present (except in comments or strings)
// We need to be careful not to match URLs or other non-type strings
expect(hasDashType).toBe(!1)})}),it("should validate schema uses underscores",()=>{const schemaPath=path.join(process.cwd(),"..","..","..","schemas","agent-output.json"),schemaContent=fs.readFileSync(schemaPath,"utf8");
// Check that each expected type is defined in the schema
["create_issue","add_comment","create_pull_request","add_labels","update_issue","push_to_pull_request_branch","create_pull_request_review_comment","create_discussion","missing_tool","create_code_scanning_alert"].forEach(type=>{const hasType=schemaContent.includes(`"const": "${type}"`);expect(hasType).toBe(!0);
// Also verify that the dash version is NOT present
const dashType=type.replace(/_/g,"-"),hasDashType=schemaContent.includes(`"const": "${dashType}"`);expect(hasDashType).toBe(!1)})}),it("should validate MCP server normalizes types to underscores",()=>{
// Check that the append function normalizes type fields to underscores
const appendPath=path.join(process.cwd(),"safe_outputs_append.cjs"),hasNormalization=fs.readFileSync(appendPath,"utf8").includes('entry.type = entry.type.replace(/-/g, "_")');expect(hasNormalization).toBe(!0);
// Check that all tool names use underscores in the tools.json file
const toolsJsonPath=path.join(process.cwd(),"safe_outputs_tools.json"),toolsContent=fs.readFileSync(toolsJsonPath,"utf8"),actualToolNames=JSON.parse(toolsContent).map(t=>t.name);["create_issue","create_discussion","add_comment","create_pull_request","create_pull_request_review_comment","create_code_scanning_alert","add_labels","update_issue","push_to_pull_request_branch","upload_asset"].forEach(toolName=>{
// Check that the tool exists in the tools.json file
expect(actualToolNames).toContain(toolName)})})});