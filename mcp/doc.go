// Package mcp provides a Terminal implementation for the MCP (Model Context Protocol) server.
//
// This tool enables execution of terminal commands on both local and remote (SSH) hosts,
// and integrates with the MCP tool registry via JSON-RPC. It supports secure credential
// loading through Viant's scy/cred system.
//
// # Key Components:
//
//   - Target: Describes the host and credentials for remote access.
//   - Command: Represents a list of shell commands to execute.
//   - Terminal: Manages remote and local terminal sessions, supporting concurrent usage.
//
// # Features:
//
//   - Local and remote terinal command execution
//   - Secure SSH credential handling via scy.Resource and cred.SSH
//   - JSON-RPC integration for remote tool invocation
//
// # Registration:
//
// Use Register to bind the tool to a MCP server:
//
//	import "github.com/viant/mcp-tools/mcp"
//
//	err := mcp.Register(implementer)
//
// This registers the tool under the "terminal" name.
//
// # Example:
//
//	cmd := &mcp.Command{
//	    Target: &mcp.Target{Host: "remote-host", Secret: ...},
//	    Commands: []string{"ls", "pwd"},
//	}
//
//	result, err := terminalTool.Call(ctx, cmd)
package mcp
