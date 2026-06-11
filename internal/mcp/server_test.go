package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	s := New()
	if s == nil {
		t.Fatal("New() returned nil")
	}
	if len(s.tools) != 0 {
		t.Errorf("expected 0 tools, got %d", len(s.tools))
	}
}

func TestRegisterTool(t *testing.T) {
	s := New()
	s.RegisterTool("test-tool", ToolDefinition{
		Name:        "test-tool",
		Description: "test description",
		InputSchema: InputSchema{Type: "object"},
	}, nil)

	tools := s.Tools()
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}
	if tools[0].Name != "test-tool" {
		t.Errorf("name = %q, want %q", tools[0].Name, "test-tool")
	}
}

func TestInitHandshake(t *testing.T) {
	s := New()
	var buf bytes.Buffer
	s.SetOutput(&buf)

	in := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}` + "\n")
	s.Run(context.Background(), in)

	var resp struct {
		JSONRPC string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Result  struct {
			ServerInfo struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"serverInfo"`
		} `json:"result"`
	}
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v\nraw: %s", err, buf.String())
	}
	if resp.Result.ServerInfo.Name != "pi-google-services" {
		t.Errorf("server name = %q, want %q", resp.Result.ServerInfo.Name, "pi-google-services")
	}
}

func TestToolsList(t *testing.T) {
	s := New()
	s.RegisterTool("my-tool", ToolDefinition{
		Name: "my-tool", Description: "desc", InputSchema: InputSchema{Type: "object"},
	}, nil)

	var buf bytes.Buffer
	s.SetOutput(&buf)

	in := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}` + "\n")
	s.Run(context.Background(), in)

	var resp struct {
		Result struct {
			Tools []ToolDefinition `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v\nraw: %s", err, buf.String())
	}
	if len(resp.Result.Tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(resp.Result.Tools))
	}
	if resp.Result.Tools[0].Name != "my-tool" {
		t.Errorf("tool name = %q, want %q", resp.Result.Tools[0].Name, "my-tool")
	}
}

func TestToolCall(t *testing.T) {
	s := New()
	s.RegisterTool("echo", ToolDefinition{Name: "echo"}, func(ctx context.Context, params json.RawMessage) (interface{}, *RPCError) {
		return map[string]interface{}{
			"content": []map[string]interface{}{{"type": "text", "text": "pong"}},
		}, nil
	})

	var buf bytes.Buffer
	s.SetOutput(&buf)

	in := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"echo","arguments":{}}}` + "\n")
	s.Run(context.Background(), in)

	var resp struct {
		Result struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
	}
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v\nraw: %s", err, buf.String())
	}
	if len(resp.Result.Content) == 0 || resp.Result.Content[0].Text != "pong" {
		t.Errorf("content = %+v, want 'pong'", resp.Result.Content)
	}
}

func TestToolCallError(t *testing.T) {
	s := New()
	s.RegisterTool("fail", ToolDefinition{Name: "fail"}, func(ctx context.Context, params json.RawMessage) (interface{}, *RPCError) {
		return nil, &RPCError{Code: -32000, Message: "boom"}
	})

	var buf bytes.Buffer
	s.SetOutput(&buf)

	in := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"fail","arguments":{}}}` + "\n")
	s.Run(context.Background(), in)

	var resp struct {
		Error *RPCError `json:"error"`
	}
	json.Unmarshal(buf.Bytes(), &resp)
	if resp.Error == nil || resp.Error.Code != -32000 || resp.Error.Message != "boom" {
		t.Errorf("error = %+v, want code=-32000 msg=boom", resp.Error)
	}
}

func TestUnknownMethod(t *testing.T) {
	s := New()
	var buf bytes.Buffer
	s.SetOutput(&buf)

	in := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"nope"}` + "\n")
	s.Run(context.Background(), in)

	var resp struct {
		Error *RPCError `json:"error"`
	}
	json.Unmarshal(buf.Bytes(), &resp)
	if resp.Error == nil || resp.Error.Code != -32601 {
		t.Errorf("error = %+v, want code -32601", resp.Error)
	}
}

func TestUnknownTool(t *testing.T) {
	s := New()
	var buf bytes.Buffer
	s.SetOutput(&buf)

	in := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"ghost","arguments":{}}}` + "\n")
	s.Run(context.Background(), in)

	var resp struct {
		Error *RPCError `json:"error"`
	}
	json.Unmarshal(buf.Bytes(), &resp)
	if resp.Error == nil || resp.Error.Code != -32601 {
		t.Errorf("error = %+v, want code -32601", resp.Error)
	}
}

func TestMultipleCalls(t *testing.T) {
	s := New()
	s.RegisterTool("ping", ToolDefinition{Name: "ping"}, func(ctx context.Context, params json.RawMessage) (interface{}, *RPCError) {
		return map[string]interface{}{"content": []map[string]interface{}{{"type": "text", "text": "pong"}}}, nil
	})

	var buf bytes.Buffer
	s.SetOutput(&buf)

	in := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}
{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"ping","arguments":{}}}
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"ping","arguments":{}}}
`)
	s.Run(context.Background(), in)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 responses, got %d: %s", len(lines), buf.String())
	}
}
