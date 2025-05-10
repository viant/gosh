package mcp

import (
	"context"
	"github.com/viant/gosh/runner"
	"github.com/viant/jsonrpc"
	"github.com/viant/mcp-protocol/schema"
	"github.com/viant/mcp-protocol/server"
)

// Register binds the Terminal tool to the MCP server
func Register(implementer *server.DefaultImplementer, options ...runner.Option) error {
	terminal, err := New(options...)
	if err != nil {
		return err
	}
	return server.RegisterTool[*Command](implementer, "terminal", "Run terminal commands on remote or local host (when target is empty)", func(ctx context.Context, input *Command) (*schema.CallToolResult, *jsonrpc.Error) {
		return terminal.Call(ctx, input)
	})
}
