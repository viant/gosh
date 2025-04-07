package runner

import (
	"context"
	"io"
)

// Runner represents a command runner
type Runner interface {
	//Run runs supplied command
	Run(ctx context.Context, command string, options ...Option) (string, int, error)
	//Stdin returns stdin writer
	Stdin() io.Writer
	//PID returns process id
	PID() int
	//Close closes runner
	Close() error
}
