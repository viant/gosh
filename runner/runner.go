package runner

import "context"

// Runner represents a command runner
type Runner interface {
	//Run runs supplied command
	Run(ctx context.Context, command string, options ...Option) (string, int, error)
	//PID returns process id
	PID() int
	//Close closes runner
	Close() error
}
