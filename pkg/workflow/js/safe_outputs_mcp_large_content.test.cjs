import{describe,it,expect,beforeEach,afterEach}from"vitest";import fs from"fs";import path from"path";import{spawn}from"child_process";import crypto from"crypto";describe.sequential("safe_outputs_mcp_server.cjs large content handling",()=>{let originalEnv,tempOutputDir,tempConfigFile,tempOutputFile;beforeEach(()=>{originalEnv={...process.env},
// Create temporary directories for testing
tempOutputDir=path.join("/tmp",`test_large_content_${Date.now()}`),fs.mkdirSync(tempOutputDir,{recursive:!0}),tempConfigFile=path.join(tempOutputDir,"config.json"),tempOutputFile=path.join(tempOutputDir,"outputs.jsonl"),fs.writeFileSync(tempConfigFile,JSON.stringify({"create-issue":{}}));
// Create tools.json file with all tools for tests
const toolsJsonPath=path.join(tempOutputDir,"tools.json"),toolsJsonContent=fs.readFileSync(path.join(__dirname,"safe_outputs_tools.json"),"utf8");fs.writeFileSync(toolsJsonPath,toolsJsonContent)}),afterEach(()=>{process.env=originalEnv,
// Clean up temporary files
fs.existsSync(tempOutputDir)&&fs.rmSync(tempOutputDir,{recursive:!0,force:!0})}),it("should write large content to file when exceeding 16000 tokens",async()=>{process.env.GH_AW_SAFE_OUTPUTS=tempOutputFile,process.env.GH_AW_SAFE_OUTPUTS_CONFIG_PATH=tempConfigFile,process.env.GH_AW_SAFE_OUTPUTS_TOOLS_PATH=path.join(tempOutputDir,"tools.json");const serverPath=path.join(__dirname,"safe_outputs_mcp_server.cjs");return new Promise((resolve,reject)=>{const timeout=setTimeout(()=>{child.kill(),reject(new Error("Test timeout"))},1e4),child=spawn("node",[serverPath],{stdio:["pipe","pipe","pipe"],env:{...process.env}});let stderr="",stdout="";child.stderr.on("data",data=>{stderr+=data.toString()}),child.stdout.on("data",data=>{stdout+=data.toString()}),child.on("error",error=>{clearTimeout(timeout),reject(error)});
// Send initialization
const initMessage=JSON.stringify({jsonrpc:"2.0",id:1,method:"initialize",params:{protocolVersion:"2024-11-05",capabilities:{},clientInfo:{name:"test-client",version:"1.0.0"}}})+"\n";child.stdin.write(initMessage),
// Wait for initialization, then send large content
setTimeout(()=>{
// Create content that's larger than 16000 tokens (> 64000 characters)
const largeBody="A".repeat(7e4),toolCallMessage=JSON.stringify({jsonrpc:"2.0",id:2,method:"tools/call",params:{name:"create_issue",arguments:{title:"Test Issue",body:largeBody}}})+"\n";// ~17500 tokens
child.stdin.write(toolCallMessage),
// Wait for response
setTimeout(()=>{child.kill(),clearTimeout(timeout);
// Parse the response
const toolCallResponse=stdout.trim().split("\n").find(line=>{try{return 2===JSON.parse(line).id}catch{return!1}});expect(toolCallResponse).toBeDefined();const parsed=JSON.parse(toolCallResponse);expect(parsed.result).toBeDefined(),expect(parsed.result.content).toBeDefined(),expect(parsed.result.content.length).toBeGreaterThan(0);const responseText=parsed.result.content[0].text,responseObj=JSON.parse(responseText);
// Check response format
expect(responseObj.filename).toBeDefined(),expect(responseObj.description).toBeDefined(),
// Description should be a schema description, not the old static text
expect(responseObj.description).not.toBe("generated content large!");
// Verify file was created
const expectedFilePath=path.join("/tmp/gh-aw/safeoutputs",responseObj.filename);expect(fs.existsSync(expectedFilePath)).toBe(!0);
// Verify file content
const fileContent=fs.readFileSync(expectedFilePath,"utf8");expect(fileContent).toBe(largeBody);
// Verify filename is SHA256 hash + .json extension (always JSON now)
const hash=crypto.createHash("sha256").update(largeBody).digest("hex");expect(responseObj.filename).toBe(`${hash}.json`);
// Verify safe output was written with file reference
const outputLines=fs.readFileSync(tempOutputFile,"utf8").trim().split("\n"),lastOutput=JSON.parse(outputLines[outputLines.length-1]);expect(lastOutput.type).toBe("create_issue"),expect(lastOutput.body).toContain("Content too large, saved to file:"),expect(lastOutput.body).toContain(responseObj.filename),
// Clean up created file
fs.existsSync(expectedFilePath)&&fs.unlinkSync(expectedFilePath),resolve()},1e3)},1e3)})}),it("should handle normal content without writing to file",async()=>{process.env.GH_AW_SAFE_OUTPUTS=tempOutputFile,process.env.GH_AW_SAFE_OUTPUTS_CONFIG_PATH=tempConfigFile,process.env.GH_AW_SAFE_OUTPUTS_TOOLS_PATH=path.join(tempOutputDir,"tools.json");const serverPath=path.join(__dirname,"safe_outputs_mcp_server.cjs");return new Promise((resolve,reject)=>{const timeout=setTimeout(()=>{child.kill(),reject(new Error("Test timeout"))},1e4),child=spawn("node",[serverPath],{stdio:["pipe","pipe","pipe"],env:{...process.env}});let stderr="",stdout="";child.stderr.on("data",data=>{stderr+=data.toString()}),child.stdout.on("data",data=>{stdout+=data.toString()}),child.on("error",error=>{clearTimeout(timeout),reject(error)});
// Send initialization
const initMessage=JSON.stringify({jsonrpc:"2.0",id:1,method:"initialize",params:{protocolVersion:"2024-11-05",capabilities:{},clientInfo:{name:"test-client",version:"1.0.0"}}})+"\n";child.stdin.write(initMessage),
// Wait for initialization, then send normal content
setTimeout(()=>{const toolCallMessage=JSON.stringify({jsonrpc:"2.0",id:2,method:"tools/call",params:{name:"create_issue",arguments:{title:"Test Issue",body:"This is a normal issue body."}}})+"\n";child.stdin.write(toolCallMessage),
// Wait for response
setTimeout(()=>{child.kill(),clearTimeout(timeout);
// Parse the response
const toolCallResponse=stdout.trim().split("\n").find(line=>{try{return 2===JSON.parse(line).id}catch{return!1}});expect(toolCallResponse).toBeDefined();const parsed=JSON.parse(toolCallResponse);expect(parsed.result).toBeDefined(),expect(parsed.result.content).toBeDefined(),expect(parsed.result.content.length).toBeGreaterThan(0);const responseText=parsed.result.content[0].text,responseObj=JSON.parse(responseText);
// Normal response should just be success
expect(responseObj.result).toBe("success");
// Verify safe output was written normally
const outputLines=fs.readFileSync(tempOutputFile,"utf8").trim().split("\n"),lastOutput=JSON.parse(outputLines[outputLines.length-1]);expect(lastOutput.type).toBe("create_issue"),expect(lastOutput.body).toBe("This is a normal issue body."),resolve()},1e3)},1e3)})}),it("should detect JSON content and use .json extension",async()=>{process.env.GH_AW_SAFE_OUTPUTS=tempOutputFile,process.env.GH_AW_SAFE_OUTPUTS_CONFIG_PATH=tempConfigFile,process.env.GH_AW_SAFE_OUTPUTS_TOOLS_PATH=path.join(tempOutputDir,"tools.json");const serverPath=path.join(__dirname,"safe_outputs_mcp_server.cjs");return new Promise((resolve,reject)=>{const timeout=setTimeout(()=>{child.kill(),reject(new Error("Test timeout"))},1e4),child=spawn("node",[serverPath],{stdio:["pipe","pipe","pipe"],env:{...process.env}});let stderr="",stdout="";child.stderr.on("data",data=>{stderr+=data.toString()}),child.stdout.on("data",data=>{stdout+=data.toString()}),child.on("error",error=>{clearTimeout(timeout),reject(error)});
// Send initialization
const initMessage=JSON.stringify({jsonrpc:"2.0",id:1,method:"initialize",params:{protocolVersion:"2024-11-05",capabilities:{},clientInfo:{name:"test-client",version:"1.0.0"}}})+"\n";child.stdin.write(initMessage),
// Wait for initialization, then send JSON content
setTimeout(()=>{
// Create large JSON content (> 16000 tokens)
const largeArray=Array(2e3).fill(null).map((_,i)=>({id:i,name:`Item ${i}`,data:"X".repeat(30)})),largeBody=JSON.stringify(largeArray,null,2),toolCallMessage=JSON.stringify({jsonrpc:"2.0",id:2,method:"tools/call",params:{name:"create_issue",arguments:{title:"Test Issue",body:largeBody}}})+"\n";child.stdin.write(toolCallMessage),
// Wait for response
setTimeout(()=>{child.kill(),clearTimeout(timeout);
// Parse the response
const toolCallResponse=stdout.trim().split("\n").find(line=>{try{return 2===JSON.parse(line).id}catch{return!1}});expect(toolCallResponse).toBeDefined();const responseText=JSON.parse(toolCallResponse).result.content[0].text,responseObj=JSON.parse(responseText);
// Check that filename has .json extension (always JSON now)
expect(responseObj.filename).toMatch(/\.json$/),
// Verify description contains schema info for array
expect(responseObj.description).toBeDefined(),expect(responseObj.description).toContain("items"),// Should mention number of items
expect(responseObj.description).toContain("id, name, data");// Should list keys
// Verify file was created with JSON extension
const expectedFilePath=path.join("/tmp/gh-aw/safeoutputs",responseObj.filename);expect(fs.existsSync(expectedFilePath)).toBe(!0);
// Verify content is valid JSON
const fileContent=fs.readFileSync(expectedFilePath,"utf8");expect(()=>JSON.parse(fileContent)).not.toThrow(),
// Clean up
fs.existsSync(expectedFilePath)&&fs.unlinkSync(expectedFilePath),resolve()},1e3)},1e3)})}),it("should always use .json extension even for non-JSON content",async()=>{process.env.GH_AW_SAFE_OUTPUTS=tempOutputFile,process.env.GH_AW_SAFE_OUTPUTS_CONFIG_PATH=tempConfigFile,process.env.GH_AW_SAFE_OUTPUTS_TOOLS_PATH=path.join(tempOutputDir,"tools.json");const serverPath=path.join(__dirname,"safe_outputs_mcp_server.cjs");return new Promise((resolve,reject)=>{const child=spawn("node",[serverPath],{stdio:["pipe","pipe","pipe"],env:{...process.env}}),timeout=setTimeout(()=>{child.kill(),reject(new Error("Test timeout"))},1e4);let stderr="",stdout="";child.stderr.on("data",data=>{stderr+=data.toString()}),child.stdout.on("data",data=>{stdout+=data.toString()}),child.on("error",error=>{clearTimeout(timeout),reject(error)});
// Send initialization
const initMessage=JSON.stringify({jsonrpc:"2.0",id:1,method:"initialize",params:{protocolVersion:"2024-11-05",capabilities:{},clientInfo:{name:"test-client",version:"1.0.0"}}})+"\n";child.stdin.write(initMessage),
// Wait for initialization, then send Markdown content
setTimeout(()=>{
// Create large Markdown content (> 16000 tokens) - not valid JSON
let largeBody="# Large Markdown Document\n\n";for(let i=0;i<1e3;i++)largeBody+=`## Section ${i}\n\n`,largeBody+="Lorem ipsum dolor sit amet, consectetur adipiscing elit. ".repeat(20),largeBody+="\n\n",largeBody+="```javascript\n",largeBody+="const example = 'code block';\n",largeBody+="console.log(example);\n",largeBody+="```\n\n";const toolCallMessage=JSON.stringify({jsonrpc:"2.0",id:2,method:"tools/call",params:{name:"create_issue",arguments:{title:"Test Issue",body:largeBody}}})+"\n";child.stdin.write(toolCallMessage),
// Wait for response
setTimeout(()=>{child.kill(),clearTimeout(timeout);
// Parse the response
const toolCallResponse=stdout.trim().split("\n").find(line=>{try{return 2===JSON.parse(line).id}catch{return!1}});expect(toolCallResponse).toBeDefined();const responseText=JSON.parse(toolCallResponse).result.content[0].text,responseObj=JSON.parse(responseText);
// Check that filename has .json extension (always .json now)
expect(responseObj.filename).toMatch(/\.json$/),
// For non-JSON content, description should be "text content"
expect(responseObj.description).toBe("text content");
// Verify file was created with .json extension
const expectedFilePath=path.join("/tmp/gh-aw/safeoutputs",responseObj.filename);expect(fs.existsSync(expectedFilePath)).toBe(!0);
// Verify content is preserved
const fileContent=fs.readFileSync(expectedFilePath,"utf8");expect(fileContent).toContain("# Large Markdown Document"),expect(fileContent).toContain("```javascript"),
// Clean up
fs.existsSync(expectedFilePath)&&fs.unlinkSync(expectedFilePath),resolve()},1e3)},1e3)})})});