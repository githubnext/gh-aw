const fs = require('fs');

// Read the engine output to find threat detection results
let verdict = {
  prompt_injection: false,
  secret_leak: false,
  malicious_patch: false,
  reasons: []
};

try {
  // Try to read engine output file
  const outputPath = '/tmp/threat-detection/agent_output.json';
  if (fs.existsSync(outputPath)) {
    const outputContent = fs.readFileSync(outputPath, 'utf8');
    
    // Look for JSON response in the output
    const jsonMatch = outputContent.match(/{[^}]*"prompt_injection"[^}]*}/g);
    if (jsonMatch && jsonMatch.length > 0) {
      const parsedVerdict = JSON.parse(jsonMatch[jsonMatch.length - 1]);
      verdict = { ...verdict, ...parsedVerdict };
    }
  }
} catch (error) {
  core.warning(`Failed to parse threat detection results: ${error.message}`);
}

// Log the verdict
core.info(`Threat detection verdict: ${JSON.stringify(verdict)}`);

// Check for threats and fail if any are detected
if (verdict.prompt_injection || verdict.secret_leak || verdict.malicious_patch) {
  const threats = [];
  if (verdict.prompt_injection) threats.push('prompt injection');
  if (verdict.secret_leak) threats.push('secret leak');
  if (verdict.malicious_patch) threats.push('malicious patch');
  
  const reasonsText = verdict.reasons && verdict.reasons.length > 0 
    ? `\nReasons: ${verdict.reasons.join('; ')}` 
    : '';
  
  core.setFailed(`❌ Security threats detected: ${threats.join(', ')}${reasonsText}`);
} else {
  core.info('✅ No security threats detected. Safe outputs may proceed.');
}