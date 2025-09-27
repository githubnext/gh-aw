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
    
    // Parse each line looking for THREAT_DETECTION_RESULT prefix
    const lines = outputContent.split('\n');
    for (const line of lines) {
      const trimmedLine = line.trim();
      if (trimmedLine.startsWith('THREAT_DETECTION_RESULT:')) {
        try {
          const jsonPart = trimmedLine.substring('THREAT_DETECTION_RESULT:'.length);
          const parsedVerdict = JSON.parse(jsonPart);
          verdict = { ...verdict, ...parsedVerdict };
          core.info('Found threat detection result in engine output');
          break; // Use the first valid result found
        } catch (parseError) {
          core.warning(`Failed to parse threat detection JSON: ${parseError.message}`);
        }
      }
    }
  }
} catch (error) {
  core.warning(`Failed to read threat detection results: ${error.message}`);
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