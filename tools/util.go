package tools

import (
	"fmt"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func toolText(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}
}

func toolError(format string, args ...any) *mcp.CallToolResult {
	msg := fmt.Sprintf(format, args...)
	slog.Warn(msg)
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
	}
}
