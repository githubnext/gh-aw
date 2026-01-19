package stringutil

import (
	"net/url"
	"strings"
)

// ExtractDomainFromURL extracts the domain name from a URL string.
// Handles various URL formats including full URLs with protocols, URLs with ports,
// and plain domain names.
//
// This function uses net/url.Parse for proper URL parsing when a protocol is present,
// and falls back to string manipulation for other formats.
//
// Examples:
//
//	ExtractDomainFromURL("https://mcp.tavily.com/mcp/")           // returns "mcp.tavily.com"
//	ExtractDomainFromURL("http://api.example.com:8080/path")      // returns "api.example.com"
//	ExtractDomainFromURL("mcp.example.com")                       // returns "mcp.example.com"
//	ExtractDomainFromURL("github.com:443")                        // returns "github.com"
//	ExtractDomainFromURL("http://sub.domain.com:8080/path")       // returns "sub.domain.com"
//	ExtractDomainFromURL("localhost:8080")                        // returns "localhost"
func ExtractDomainFromURL(urlStr string) string {
	// Handle full URLs with protocols (http://, https://)
	if strings.HasPrefix(urlStr, "http://") || strings.HasPrefix(urlStr, "https://") {
		// Parse full URL
		parsedURL, err := url.Parse(urlStr)
		if err != nil {
			// Fall back to string manipulation if parsing fails
			return extractDomainFallback(urlStr)
		}
		return parsedURL.Hostname()
	}

	// For URLs without protocol, use string manipulation
	return extractDomainFallback(urlStr)
}

// extractDomainFallback extracts domain using string manipulation.
// This handles URLs without protocols, CONNECT requests (domain:port format),
// and plain domain names.
func extractDomainFallback(urlStr string) string {
	// Remove protocol if present (in case it wasn't http/https)
	urlStr = strings.TrimPrefix(urlStr, "https://")
	urlStr = strings.TrimPrefix(urlStr, "http://")

	// Remove port and path
	if idx := strings.Index(urlStr, ":"); idx != -1 {
		urlStr = urlStr[:idx]
	}
	if idx := strings.Index(urlStr, "/"); idx != -1 {
		urlStr = urlStr[:idx]
	}

	return strings.TrimSpace(urlStr)
}
