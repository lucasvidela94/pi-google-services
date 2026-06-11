// Package services provides pluggable Google service implementations
// for the MCP server. Each service defines its tools, OAuth scopes,
// and handlers independently.
package services

import (
	"context"
	"encoding/json"

	"github.com/sombi/pi-google-services/internal/mcp"
)

// Service is the interface that every Google service (Calendar, Gmail, etc.)
// must implement to register with the MCP server.
type Service interface {
	// Name returns a unique identifier for this service.
	Name() string

	// Scopes returns the OAuth2 scopes required by this service.
	Scopes() []string

	// Tools returns the tool definitions this service provides.
	Tools() []mcp.ToolDefinition

	// Handle dispatches a tool call to the appropriate handler.
	Handle(ctx context.Context, toolName string, params json.RawMessage) (interface{}, *mcp.RPCError)
}

// BaseService provides common fields for service implementations.
type BaseService struct {
	name   string
	scopes []string
	tools  []mcp.ToolDefinition
}
