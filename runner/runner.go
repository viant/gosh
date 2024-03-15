package runner

import "context"

// Runner represents a command runner
type Runner interface {
	Run(ctx context.Context, command string, options ...Option) (string, int, error)
	PID() int
	Close() error
}
