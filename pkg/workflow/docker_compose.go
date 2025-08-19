package workflow

// generateDockerCompose generates the Docker Compose configuration
func generateDockerCompose(containerImage string, envVars map[string]any, toolName string) string {
	compose := `services:
  squid-proxy:
    image: ubuntu/squid:latest
    container_name: squid-proxy-` + toolName + `
    ports:
      - "3128:3128"
    volumes:
      - ./squid.conf:/etc/squid/squid.conf:ro
      - ./allowed_domains.txt:/etc/squid/allowed_domains.txt:ro
      - squid-logs:/var/log/squid
    healthcheck:
      test: ["CMD", "squid", "-k", "check"]
      interval: 30s
      timeout: 10s
      retries: 3
    restart: unless-stopped

  ` + toolName + `:
    image: ` + containerImage + `
    container_name: ` + toolName + `-mcp
    environment:
      - PROXY_HOST=squid-proxy-` + toolName + `
      - PROXY_PORT=3128`

	// Add environment variables
	if envVars != nil {
		for key, value := range envVars {
			if valueStr, ok := value.(string); ok {
				compose += "\n      - " + key + "=" + valueStr
			}
		}
	}

	compose += `
    depends_on:
      squid-proxy-` + toolName + `:
        condition: service_healthy

volumes:
  squid-logs:
`

	return compose
}
