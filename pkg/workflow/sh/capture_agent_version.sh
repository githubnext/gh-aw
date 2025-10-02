# Extract semantic version pattern (e.g., 1.2.3, v1.2.3-beta)
CLEAN_VERSION=$(echo "$VERSION_OUTPUT" | grep -oE 'v?[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9]+)?' | head -n1 || echo "unknown")
echo "AGENT_VERSION=$CLEAN_VERSION" >> $GITHUB_ENV
echo "Agent version: $VERSION_OUTPUT"
