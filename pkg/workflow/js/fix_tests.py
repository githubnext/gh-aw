import re

with open('safe_outputs_mcp_server_defaults.test.cjs', 'r') as f:
    lines = f.readlines()

output_lines = []
i = 0
temp_config_counter = 0

while i < len(lines):
    line = lines[i]
    
    # Check if this line contains the old env var pattern
    if 'GH_AW_SAFE_OUTPUTS_CONFIG: JSON.stringify(config),' in line:
        # Replace with new pattern
        new_line = line.replace(
            'GH_AW_SAFE_OUTPUTS_CONFIG: JSON.stringify(config),',
            'GH_AW_SAFE_OUTPUTS_CONFIG_FILE: tempConfigPath,'
        )
        
        # Check if we already added temp config creation for this test
        # by looking backwards for the marker
        needs_temp_config = True
        for j in range(max(0, i-20), i):
            if 'Write config to temp file' in lines[j]:
                needs_temp_config = False
                break
        
        if needs_temp_config:
            # Find where to insert the temp config creation
            # Look backwards for "const config = {" and insert after the closing }
            for j in range(i-1, max(0, i-50), -1):
                if 'const config = {' in lines[j]:
                    # Find the closing brace
                    brace_count = 0
                    start_found = False
                    for k in range(j, i):
                        for char in lines[k]:
                            if char == '{':
                                brace_count += 1
                                start_found = True
                            elif char == '}' and start_found:
                                brace_count -= 1
                                if brace_count == 0:
                                    # Insert after this line
                                    insertion_point = k + 1
                                    temp_config_counter += 1
                                    temp_lines = [
                                        '\n',
                                        '    // Write config to temp file\n',
                                        f'    const tempConfigPath = path.join("/tmp", `test_config_{temp_config_counter}_${{Date.now()}}_${{Math.random().toString(36).substring(7)}}.json`);\n',
                                        '    fs.writeFileSync(tempConfigPath, JSON.stringify(config));\n',
                                        '\n'
                                    ]
                                    # Insert the temp config lines
                                    lines = lines[:insertion_point] + temp_lines + lines[insertion_point:]
                                    i += len(temp_lines)
                                    break
                    break
        
        output_lines.append(new_line)
    else:
        output_lines.append(line)
    
    i += 1

with open('safe_outputs_mcp_server_defaults.test.cjs', 'w') as f:
    f.writelines(output_lines)

print(f"Fixed {temp_config_counter} tests")
