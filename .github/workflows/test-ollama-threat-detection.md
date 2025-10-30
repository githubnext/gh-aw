---
on:
  workflow_dispatch:

permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read

engine: copilot

safe-outputs:
  create-issue:
    title-prefix: "[test] "
    labels: [test]
    max: 1
  threat-detection:
    steps:
      - name: Ollama Llama Guard 3 Threat Scan
        id: ollama-scan
        uses: actions/github-script@60a0d83039c74a4aee543508d2ffcb1c3799cdea # v7.0.1
        with:
          script: |
            const fs = require('fs');
            const path = require('path');
            
            // ===== INSTALL OLLAMA =====
            core.info('🚀 Starting Ollama installation...');
            try {
              core.info('📥 Downloading Ollama installer...');
              await exec.exec('curl', ['-fsSL', 'https://ollama.com/install.sh', '-o', '/tmp/install-ollama.sh']);
              
              core.info('📦 Installing Ollama...');
              await exec.exec('sh', ['/tmp/install-ollama.sh']);
              
              core.info('✅ Verifying Ollama installation...');
              const versionOutput = await exec.getExecOutput('ollama', ['--version']);
              core.info(`Ollama version: ${versionOutput.stdout.trim()}`);
              core.info('✅ Ollama installed successfully');
            } catch (error) {
              core.setFailed(`Failed to install Ollama: ${error instanceof Error ? error.message : String(error)}`);
              throw error;
            }
            
            // ===== START OLLAMA SERVICE =====
            core.info('🚀 Starting Ollama service...');
            const logDir = '/tmp/gh-aw/ollama-logs';
            if (!fs.existsSync(logDir)) {
              fs.mkdirSync(logDir, { recursive: true });
            }
            
            // Start Ollama service in background
            const ollamaServeLog = fs.openSync(`${logDir}/ollama-serve.log`, 'w');
            const ollamaServeErrLog = fs.openSync(`${logDir}/ollama-serve-error.log`, 'w');
            exec.exec('ollama', ['serve'], {
              detached: true,
              silent: true,
              outStream: fs.createWriteStream(`${logDir}/ollama-serve.log`),
              errStream: fs.createWriteStream(`${logDir}/ollama-serve-error.log`)
            }).then(() => {
              core.info('Ollama service started in background');
            }).catch(err => {
              core.warning(`Ollama service background start: ${err.message}`);
            });
            
            // Wait for service to be ready
            core.info('⏳ Waiting for Ollama service to be ready...');
            let retries = 30;
            while (retries > 0) {
              try {
                await exec.exec('curl', ['-f', 'http://localhost:11434/api/version'], {
                  silent: true
                });
                core.info('✅ Ollama service is ready');
                break;
              } catch (e) {
                retries--;
                if (retries === 0) {
                  throw new Error('Ollama service did not become ready in time');
                }
                await new Promise(resolve => setTimeout(resolve, 1000));
              }
            }
            
            // ===== DOWNLOAD LLAMA GUARD 3 MODEL =====
            core.info('📥 Checking for Llama Guard 3:1b model...');
            try {
              // Check if model is already available
              const modelsOutput = await exec.getExecOutput('ollama', ['list']);
              const modelExists = modelsOutput.stdout.includes('llama-guard3');
              
              if (modelExists) {
                core.info('✅ Llama Guard 3 model already available');
              } else {
                core.info('📥 Downloading Llama Guard 3:1b model...');
                core.info('This may take several minutes...');
                const startTime = Date.now();
                await exec.exec('ollama', ['pull', 'llama-guard3:1b']);
                
                const elapsed = ((Date.now() - startTime) / 1000).toFixed(1);
                core.info(`✅ Model downloaded successfully in ${elapsed}s`);
                
                // Verify model is now available
                const verifyOutput = await exec.getExecOutput('ollama', ['list']);
                if (!verifyOutput.stdout.includes('llama-guard3')) {
                  throw new Error('Llama Guard 3 model not found after download');
                }
              }
              core.info('✅ Llama Guard 3 model ready');
            } catch (error) {
              core.setFailed(`Failed to prepare model: ${error instanceof Error ? error.message : String(error)}`);
              throw error;
            }
            
            // ===== SCAN SAFE OUTPUTS =====
            core.info('🔍 Starting Llama Guard 3 threat scan...');
            const scanDir = '/tmp/gh-aw/threat-detection';
            
            let threatsDetected = false;
            const results = [];
            
            // ===== SCAN AGENT OUTPUT ITEMS =====
            const agentOutputPath = path.join(scanDir, 'agent_output.json');
            core.info(`\n📄 Scanning Agent Output Items: ${agentOutputPath}`);
            
            if (fs.existsSync(agentOutputPath)) {
              try {
                const agentOutputContent = fs.readFileSync(agentOutputPath, 'utf8');
                const agentOutput = JSON.parse(agentOutputContent);
                
                if (agentOutput.items && Array.isArray(agentOutput.items)) {
                  core.info(`Found ${agentOutput.items.length} safe output items to scan`);
                  
                  for (let i = 0; i < agentOutput.items.length; i++) {
                    const item = agentOutput.items[i];
                    const itemName = `Agent Output Item #${i + 1} (${item.type || 'unknown'})`;
                    core.info(`\n📋 Scanning ${itemName}...`);
                    
                    try {
                      // Convert item to string for analysis
                      const itemContent = JSON.stringify(item, null, 2);
                      const itemSize = (itemContent.length / 1024).toFixed(2);
                      core.info(`Item size: ${itemSize} KB`);
                      
                      // Truncate very large items
                      const maxChars = 8000;
                      const content = itemContent.length > maxChars 
                        ? itemContent.substring(0, maxChars) + '\n\n[Content truncated for scanning]'
                        : itemContent;
                      
                      core.info('🤖 Running Llama Guard 3 analysis...');
                      const scanStart = Date.now();
                      
                      let output = '';
                      try {
                        const response = await exec.getExecOutput('curl', [
                          '-X', 'POST',
                          'http://localhost:11434/api/chat',
                          '-H', 'Content-Type: application/json',
                          '-d', JSON.stringify({
                            model: 'llama-guard3:1b',
                            messages: [{ role: 'user', content: content }],
                            stream: false
                          })
                        ]);
                        const apiResult = JSON.parse(response.stdout);
                        output = apiResult.message?.content || '';
                      } catch (error) {
                        core.warning(`Llama Guard 3 execution error: ${error instanceof Error ? error.message : String(error)}`);
                        output = error.stdout || '';
                      }
                      
                      const scanElapsed = ((Date.now() - scanStart) / 1000).toFixed(1);
                      core.info(`Analysis completed in ${scanElapsed}s`);
                      
                      core.info(`\n📊 Llama Guard 3 Response:\n${output}`);
                      
                      // Result must be exactly "safe" or "S8" (Intellectual Property, which we allow)
                      const outputTrimmed = output.trim().toLowerCase();
                      const isSafe = outputTrimmed === 'safe' || outputTrimmed === 's8';
                      
                      results.push({
                        file: itemName,
                        path: agentOutputPath,
                        itemIndex: i,
                        itemType: item.type,
                        safe: isSafe,
                        response: output.trim()
                      });
                      
                      if (!isSafe) {
                        threatsDetected = true;
                        core.warning(`⚠️  Potential threat detected in ${itemName}`);
                      }
                    } catch (error) {
                      core.error(`Error scanning ${itemName}: ${error instanceof Error ? error.message : String(error)}`);
                      results.push({
                        file: itemName,
                        path: agentOutputPath,
                        itemIndex: i,
                        safe: false,
                        error: error instanceof Error ? error.message : String(error)
                      });
                      threatsDetected = true;
                    }
                  }
                } else {
                  core.info('No items array found in agent_output.json');
                }
              } catch (error) {
                core.error(`Error reading agent_output.json: ${error instanceof Error ? error.message : String(error)}`);
                results.push({
                  file: 'Agent Output',
                  path: agentOutputPath,
                  safe: false,
                  error: error instanceof Error ? error.message : String(error)
                });
                threatsDetected = true;
              }
            } else {
              core.info(`⚠️  Agent output file not found, skipping: ${agentOutputPath}`);
            }
            
            // ===== SCAN CODE PATCH =====
            const patchPath = path.join(scanDir, 'aw.patch');
            core.info(`\n📄 Scanning Code Patch: ${patchPath}`);
            
            if (fs.existsSync(patchPath)) {
              try {
                const patchContent = fs.readFileSync(patchPath, 'utf8');
                const patchSize = (patchContent.length / 1024).toFixed(2);
                core.info(`Patch size: ${patchSize} KB`);
                
                // Truncate very large patches
                const maxChars = 8000;
                const content = patchContent.length > maxChars 
                  ? patchContent.substring(0, maxChars) + '\n\n[Content truncated for scanning]'
                  : patchContent;
                
                core.info('🤖 Running Llama Guard 3 analysis...');
                const scanStart = Date.now();
                
                let output = '';
                try {
                  const response = await exec.getExecOutput('curl', [
                    '-X', 'POST',
                    'http://localhost:11434/api/chat',
                    '-H', 'Content-Type: application/json',
                    '-d', JSON.stringify({
                      model: 'llama-guard3:1b',
                      messages: [{ role: 'user', content: content }],
                      stream: false
                    })
                  ]);
                  const apiResult = JSON.parse(response.stdout);
                  output = apiResult.message?.content || '';
                } catch (error) {
                  core.warning(`Llama Guard 3 execution error: ${error instanceof Error ? error.message : String(error)}`);
                  output = error.stdout || '';
                }
                
                const scanElapsed = ((Date.now() - scanStart) / 1000).toFixed(1);
                core.info(`Analysis completed in ${scanElapsed}s`);
                
                core.info(`\n📊 Llama Guard 3 Response:\n${output}`);
                
                // Result must be exactly "safe" or "S8" (Intellectual Property, which we allow)
                const outputTrimmed = output.trim().toLowerCase();
                const isSafe = outputTrimmed === 'safe' || outputTrimmed === 's8';
                
                results.push({
                  file: 'Code Patch',
                  path: patchPath,
                  safe: isSafe,
                  response: output.trim()
                });
                
                if (!isSafe) {
                  threatsDetected = true;
                  core.warning(`⚠️  Potential threat detected in Code Patch`);
                }
              } catch (error) {
                core.error(`Error scanning Code Patch: ${error instanceof Error ? error.message : String(error)}`);
                results.push({
                  file: 'Code Patch',
                  path: patchPath,
                  safe: false,
                  error: error instanceof Error ? error.message : String(error)
                });
                threatsDetected = true;
              }
            } else {
              core.info(`⚠️  Patch file not found, skipping: ${patchPath}`);
            }
            
            // Write results
            const resultsPath = '/tmp/gh-aw/threat-detection/ollama-scan-results.json';
            fs.writeFileSync(resultsPath, JSON.stringify(results, null, 2));
            core.info(`\n📝 Results written to: ${resultsPath}`);
            
            // Summary
            core.info('\n' + '='.repeat(60));
            core.info('🔍 Llama Guard 3 Scan Summary');
            core.info('='.repeat(60));
            for (const result of results) {
              const status = result.safe ? '✅ SAFE' : '❌ UNSAFE';
              core.info(`${status} - ${result.file}`);
              if (!result.safe && result.response) {
                core.info(`  Reason: ${result.response.substring(0, 200)}`);
              }
            }
            core.info('='.repeat(60));
            
            if (threatsDetected) {
              core.setFailed('❌ Llama Guard 3 detected potential security threats in the safe outputs or patches');
            } else {
              core.info('✅ All scanned content appears safe');
            }
      
      
      - name: Upload scan results
        if: always()
        uses: actions/upload-artifact@50769540e7f4bd5e21e526ee35c689e35e0d6874 # v4.4.0
        with:
          name: ollama-scan-results
          path: |
            /tmp/gh-aw/threat-detection/ollama-scan-results.json
            /tmp/gh-aw/ollama-logs/
          if-no-files-found: ignore

timeout_minutes: 20
---

# Test Ollama Threat Scanning

This is a test workflow to verify the Ollama Llama Guard 3 threat scanning configuration.

Create a simple test issue to demonstrate the workflow executes successfully with threat detection enabled.
