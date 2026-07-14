package cli

import (
	"encoding/base64"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func mcpToolIcons(emoji string) []mcp.Icon {
	return []mcp.Icon{{Source: mcpEmojiIconSource(emoji)}}
}

func mcpEmojiIconSource(emoji string) string {
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32"><text x="16" y="24" font-size="24" text-anchor="middle">%s</text></svg>`, emoji)
	return "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(svg))
}
