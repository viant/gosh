package option

import (
	"github.com/viant/gosh/runner"
)

const (
	defaultShell     = "/bin/sh"
	defaultTimeoutMs = 20000
)

type (
	options struct {
		shell       string
		shellPrompt string
		timeoutSec  int
		runner      runner.Runner
		history     *runner.History
	}
	//Option represents a shh option
	Option func(*options)
)

func newOptions(opts []Option) *options {
	opt := &options{}
	for _, o := range opts {
		o(opt)
	}
	if opt.shell == "" {
		opt.shell = defaultShell
	}
	if opt.timeoutSec == 0 {
		opt.timeoutSec = defaultTimeoutMs
	}
	return opt
}

// WithShell creates with shell option
func WithShell(shell string) Option {
	return func(o *options) {
		o.shell = shell
	}
}

// WithShellPrompt creates with shell prompt option
func WithShellPrompt(shellPrompt string) Option {
	return func(o *options) {
		o.shellPrompt = shellPrompt
	}
}

// WithTimeout creates with timeout option
func WithTimeout(timeoutSec int) Option {
	return func(o *options) {
		o.timeoutSec = timeoutSec
	}
}

// WithRunner creates with runner option
func WithRunner(runner runner.Runner) Option {
	return func(o *options) {
		o.runner = runner
	}
}

// WithHistory creates with history option
func WithHistory(history *runner.History) Option {
	return func(o *options) {
		o.history = history
	}
}
