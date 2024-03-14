package runner

import (
	"fmt"
	"strings"
)

// Command represents a command
type Command struct {
	Stdin  string
	Index  int
	Stdout []string
	Error  []string
}

// Output returns command output
func (c *Command) Output() string {
	return strings.Join(c.Stdout, "\n")
}

// Err returns command error
func (c *Command) Err() error {
	if len(c.Error) == 0 {
		return nil
	}
	return fmt.Errorf(strings.Join(c.Error, "\n"))
}

// NewCommand creates a new command
func NewCommand(command string, output string, err error) *Command {
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	ret := &Command{
		Stdin: command,
	}
	if output != "" {
		ret.Stdout = strings.Split(output, "\n")
	}
	if errStr != "" {
		ret.Stdout = strings.Split(errStr, "\n")
	}
	return ret
}
