package mcp_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/viant/gosh/mcp"
)

// TestLocalTerminalTool_Echo verifies that the local terminal tool can execute simple commands
func TestLocalTerminalTool_Echo(t *testing.T) {
	// Initialize tool with local terminal
	tool, err := mcp.New()
	assert.NoError(t, err)

	// Prepare a command sequence
	cmd := &mcp.Command{
		Commands: []string{"echo hello", "echo world"},
	}
	// Execute commands
	result, rpcErr := tool.Call(context.Background(), cmd)
	// No protocol-level error
	assert.Nil(t, rpcErr)
	assert.NotNil(t, result)
	// Should not be marked as error
	assert.Nil(t, result.IsError)
	// Expect a single text content element
	assert.Len(t, result.Content, 1)

	elem := result.Content[0]
	// Content type should be "text"
	assert.Equal(t, "text", elem.Type)
	// Output should include both lines
	assert.Equal(t, "hello\nworld\n", elem.Text)
}
