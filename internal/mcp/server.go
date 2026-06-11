// Package mcp implements the Model Context Protocol (JSON-RPC 2.0 over stdio)
// for exposing service tools to MCP clients like Pi.
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
)

// Request represents a JSON-RPC 2.0 request.
type Request struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string           `json:"method"`
	Params  json.RawMessage  `json:"params,omitempty"`
}

// Response represents a JSON-RPC 2.0 response.
type Response struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Result  interface{}      `json:"result,omitempty"`
	Error   *RPCError        `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC error.
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ToolDefinition describes an MCP tool for the tools/list response.
type ToolDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

// InputSchema is a JSON Schema for tool parameters.
type InputSchema struct {
	Type       string                    `json:"type"`
	Properties map[string]PropertySchema `json:"properties,omitempty"`
	Required   []string                  `json:"required,omitempty"`
}

// PropertySchema describes a single parameter.
type PropertySchema struct {
	Type        string      `json:"type"`
	Description string      `json:"description,omitempty"`
	Default     interface{} `json:"default,omitempty"`
}

// ToolHandler processes a tool call and returns content or an error.
type ToolHandler func(ctx context.Context, params json.RawMessage) (interface{}, *RPCError)

// ToolEntry holds a tool's definition and handler.
type ToolEntry struct {
	Definition ToolDefinition
	Handler    ToolHandler
}

// Server implements the stdio-based MCP transport.
type Server struct {
	tools  map[string]ToolEntry
	closed bool
	w      io.Writer // output writer; defaults to os.Stdout
}

// New creates an empty MCP server. Tools are added via RegisterTool.
func New() *Server {
	return &Server{
		tools: make(map[string]ToolEntry),
		w:     os.Stdout,
	}
}

// SetOutput changes the output writer (used in tests).
func (s *Server) SetOutput(w io.Writer) {
	s.w = w
}

// RegisterTool adds a single tool to the server.
func (s *Server) RegisterTool(name string, def ToolDefinition, handler ToolHandler) {
	s.tools[name] = ToolEntry{Definition: def, Handler: handler}
}

// Tools returns all registered tool definitions (thread-safe after init).
func (s *Server) Tools() []ToolDefinition {
	defs := make([]ToolDefinition, 0, len(s.tools))
	for _, entry := range s.tools {
		defs = append(defs, entry.Definition)
	}
	return defs
}

// Run starts the read loop, processing JSON-RPC messages from r.
// When r is nil, defaults to os.Stdin.
func (s *Server) Run(ctx context.Context, r ...io.Reader) error {
	var reader io.Reader = os.Stdin
	if len(r) > 0 && r[0] != nil {
		reader = r[0]
	}

	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 256*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if s.closed {
			return nil
		}

		var req Request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			log.Printf("Invalid JSON-RPC: %v", err)
			continue
		}
		s.handleMessage(ctx, &req)
	}
	return scanner.Err()
}

func (s *Server) handleMessage(ctx context.Context, req *Request) {
	switch req.Method {
	case "initialize":
		s.handleInitialize(req)
	case "notifications/initialized":
		// protocol expects this notification; no-op
	case "tools/list":
		s.handleToolsList(req)
	case "tools/call":
		s.handleToolCall(ctx, req)
	default:
		if req.ID != nil {
			s.sendError(req.ID, -32601, fmt.Sprintf("Method not found: %s", req.Method), nil)
		}
	}
}

func (s *Server) handleInitialize(req *Request) {
	s.sendResult(req.ID, map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{},
		},
		"serverInfo": map[string]interface{}{
			"name":    "pi-google-services",
			"version": "0.1.0",
		},
	})
}

func (s *Server) handleToolsList(req *Request) {
	s.sendResult(req.ID, map[string]interface{}{
		"tools": s.Tools(),
	})
}

func (s *Server) handleToolCall(ctx context.Context, req *Request) {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendError(req.ID, -32602, "Invalid params", err.Error())
		return
	}

	entry, ok := s.tools[params.Name]
	if !ok {
		s.sendError(req.ID, -32601, fmt.Sprintf("Tool not found: %s", params.Name), nil)
		return
	}

	result, rpcErr := entry.Handler(ctx, params.Arguments)
	if rpcErr != nil {
		s.sendError(req.ID, rpcErr.Code, rpcErr.Message, rpcErr.Data)
		return
	}
	s.sendResult(req.ID, result)
}

func (s *Server) sendResult(id *json.RawMessage, result interface{}) {
	resp := Response{JSONRPC: "2.0", ID: id, Result: result}
	s.writeJSON(resp)
}

func (s *Server) sendError(id *json.RawMessage, code int, message string, data interface{}) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &RPCError{Code: code, Message: message, Data: data},
	}
	s.writeJSON(resp)
}

func (s *Server) writeJSON(v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		log.Printf("Error marshaling response: %v", err)
		return
	}
	data = append(data, '\n')
	if _, err := s.w.Write(data); err != nil {
		log.Printf("Error writing response: %v", err)
	}
}
