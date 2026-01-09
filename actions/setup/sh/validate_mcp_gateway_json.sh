#!/usr/bin/env bash
# Validate MCP Gateway JSON Configuration
# This script validates that a JSON file conforms to the MCP Gateway Specification v1.0.0
# Usage: validate_mcp_gateway_json.sh <json-file-path>

set -e

if [ -z "$1" ]; then
  echo "ERROR: JSON file path required"
  echo "Usage: $0 <json-file-path>"
  exit 1
fi

JSON_FILE="$1"

if [ ! -f "$JSON_FILE" ]; then
  echo "ERROR: JSON file not found: $JSON_FILE"
  exit 1
fi

echo "Validating MCP gateway configuration: $JSON_FILE"

# Basic JSON validation
if ! jq empty "$JSON_FILE" 2>/dev/null; then
  echo "ERROR: Invalid JSON format"
  echo "File contents:"
  cat "$JSON_FILE"
  exit 1
fi

# Check for required mcpServers section
if ! jq -e '.mcpServers' "$JSON_FILE" >/dev/null 2>&1; then
  echo "ERROR: Missing required 'mcpServers' section in configuration"
  exit 1
fi

# Validate each MCP server configuration
SERVER_NAMES=$(jq -r '.mcpServers | keys[]' "$JSON_FILE" 2>/dev/null || echo "")

if [ -z "$SERVER_NAMES" ]; then
  echo "WARNING: No MCP servers defined in configuration"
fi

for SERVER_NAME in $SERVER_NAMES; do
  echo "Validating server: $SERVER_NAME"
  
  # Get server type (infer if not explicit)
  SERVER_TYPE=$(jq -r ".mcpServers[\"$SERVER_NAME\"].type // \"\"" "$JSON_FILE")
  HAS_URL=$(jq -e ".mcpServers[\"$SERVER_NAME\"].url" "$JSON_FILE" >/dev/null 2>&1 && echo "true" || echo "false")
  HAS_CONTAINER=$(jq -e ".mcpServers[\"$SERVER_NAME\"].container" "$JSON_FILE" >/dev/null 2>&1 && echo "true" || echo "false")
  HAS_COMMAND=$(jq -e ".mcpServers[\"$SERVER_NAME\"].command" "$JSON_FILE" >/dev/null 2>&1 && echo "true" || echo "false")
  
  # Infer type if not explicit
  if [ -z "$SERVER_TYPE" ] || [ "$SERVER_TYPE" = "null" ]; then
    if [ "$HAS_URL" = "true" ]; then
      SERVER_TYPE="http"
    elif [ "$HAS_CONTAINER" = "true" ] || [ "$HAS_COMMAND" = "true" ]; then
      SERVER_TYPE="stdio"
    else
      echo "ERROR: Server '$SERVER_NAME': unable to determine type (stdio or http)"
      echo "Must specify 'type', 'url', 'container', or 'command'"
      exit 1
    fi
  fi
  
  # Normalize "local" to "stdio"
  if [ "$SERVER_TYPE" = "local" ]; then
    SERVER_TYPE="stdio"
  fi
  
  # Validate based on type
  case "$SERVER_TYPE" in
    stdio)
      # Check for containerization (required per spec 3.2.1)
      COMMAND=$(jq -r ".mcpServers[\"$SERVER_NAME\"].command // \"\"" "$JSON_FILE")
      
      # Check if it's a docker command (transformed container)
      if [ "$COMMAND" = "docker" ]; then
        ARGS=$(jq -r ".mcpServers[\"$SERVER_NAME\"].args | length" "$JSON_FILE" 2>/dev/null || echo "0")
        if [ "$ARGS" -gt 0 ]; then
          # Valid: docker command with args (transformed container)
          echo "  ✓ Valid stdio server (docker command pattern)"
          continue
        fi
      fi
      
      # Check for direct command execution without container (not supported)
      if [ -n "$COMMAND" ] && [ "$COMMAND" != "null" ] && [ "$COMMAND" != "docker" ] && [ "$HAS_CONTAINER" = "false" ]; then
        echo "ERROR: Server '$SERVER_NAME': direct command execution is NOT supported"
        echo "Per MCP Gateway Specification v1.0.0 section 3.2.1, stdio-based MCP servers MUST be containerized"
        echo "Please specify a 'container' field with a Docker image instead of using 'command'"
        exit 1
      fi
      
      # Check for container field
      if [ "$HAS_CONTAINER" = "false" ]; then
        echo "ERROR: Server '$SERVER_NAME': stdio type requires 'container' field"
        echo "Per MCP Gateway Specification v1.0.0 section 3.2.1, stdio-based MCP servers MUST be containerized"
        exit 1
      fi
      
      echo "  ✓ Valid stdio server (containerized)"
      ;;
      
    http)
      # HTTP servers require URL
      if [ "$HAS_URL" = "false" ]; then
        echo "ERROR: Server '$SERVER_NAME': http type requires 'url' field"
        echo "HTTP MCP servers must specify a URL endpoint"
        exit 1
      fi
      
      # Validate URL format
      URL=$(jq -r ".mcpServers[\"$SERVER_NAME\"].url" "$JSON_FILE")
      if [[ ! "$URL" =~ ^https?:// ]]; then
        echo "ERROR: Server '$SERVER_NAME': invalid URL '$URL'"
        echo "URLs must start with http:// or https://"
        exit 1
      fi
      
      echo "  ✓ Valid HTTP server"
      ;;
      
    *)
      echo "ERROR: Server '$SERVER_NAME': unsupported type '$SERVER_TYPE'"
      echo "Valid types are: stdio, http"
      exit 1
      ;;
  esac
done

# Validate gateway configuration if present
if jq -e '.gateway' "$JSON_FILE" >/dev/null 2>&1; then
  echo "Validating gateway configuration"
  
  # Validate port if specified
  PORT=$(jq -r '.gateway.port // 0' "$JSON_FILE")
  if [ "$PORT" -ne 0 ] && ([ "$PORT" -lt 1 ] || [ "$PORT" -gt 65535 ]); then
    echo "ERROR: Gateway: invalid port $PORT. Port must be between 1 and 65535"
    exit 1
  fi
  
  # Validate timeouts if specified
  STARTUP_TIMEOUT=$(jq -r '.gateway.startupTimeout // 0' "$JSON_FILE")
  if [ "$STARTUP_TIMEOUT" -lt 0 ]; then
    echo "ERROR: Gateway: invalid startupTimeout $STARTUP_TIMEOUT. Timeout must be non-negative"
    exit 1
  fi
  
  TOOL_TIMEOUT=$(jq -r '.gateway.toolTimeout // 0' "$JSON_FILE")
  if [ "$TOOL_TIMEOUT" -lt 0 ]; then
    echo "ERROR: Gateway: invalid toolTimeout $TOOL_TIMEOUT. Timeout must be non-negative"
    exit 1
  fi
  
  echo "  ✓ Valid gateway configuration"
fi

echo "✓ MCP gateway configuration is valid"
exit 0
