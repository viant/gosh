package replay

import (
	"context"
	"fmt"
	"github.com/viant/gosh/runner"
)

// Runner represents a command runner
type Runner struct {
	from []*runner.Command
	pid  int
}

func (r *Runner) Close() error {
	return nil
}

func (r *Runner) matchRemove(input string) *runner.Command {
	var matched *runner.Command
	var from []*runner.Command
	for i, cmd := range r.from {
		if cmd.Stdin == input {
			matched = r.from[i]
			continue
		}
		from = append(from, cmd)
	}
	r.from = from
	return matched
}

// PID returns process id
func (r *Runner) PID() int {
	return r.pid
}

// Run runs supplied command
func (r *Runner) Run(ctx context.Context, command string, options ...runner.Option) (string, error) {
	if len(r.from) == 0 {
		return "", nil
	}
	cmd := r.matchRemove(command)
	if cmd == nil {
		return "", fmt.Errorf("no found")
	}
	return cmd.Output(), cmd.Err()
}

// New creates a new runner
func New(pid int, from []*runner.Command) *Runner {
	return &Runner{from: from, pid: pid}
}
