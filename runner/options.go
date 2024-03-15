package runner

import (
	"github.com/viant/gosh/term"
)

const (
	defaultShell      = "/bin/sh"
	defaultTimeoutMs  = 20000
	defaultBufferSize = 128 * 1024
)

type (
	//Options represents runner options
	Options struct {
		Runner             Runner
		Shell              string
		Term               string
		Cols               int
		Rows               int
		Path               string
		Env                map[string]string
		SystemPaths        []string
		shellPrompt        string
		escapedShellPrompt string
		History            *History
		bufferSize         int
		listener           Listener
		timeoutMs          int
		flashIntervalMs    int
		terminators        []string
	}

	//Option represents runner option
	Option func(*Options)
)

// Environ returns environment variables
func (p *Options) Environ() []string {
	var result []string
	if len(p.Env) == 0 {
		return result
	}
	for k, v := range p.Env {
		result = append(result, k+"="+v)
	}
	return result

}

// Apply applies options
func (p *Options) Apply(options []Option) *Options {
	ret := *p
	for _, o := range options {
		o(&ret)
	}
	if ret.timeoutMs == 0 {
		ret.timeoutMs = defaultTimeoutMs
	}
	return &ret
}

// NewOptions creates a new options
func NewOptions(opts []Option) *Options {
	opt := &Options{}
	for _, o := range opts {
		o(opt)
	}
	if opt.Shell == "" {
		opt.Shell = defaultShell
	}
	if opt.timeoutMs == 0 {
		opt.timeoutMs = defaultTimeoutMs
	}
	if opt.bufferSize == 0 {
		opt.bufferSize = defaultBufferSize
	}
	if opt.Term == "" {
		opt.Term = "xterm"
	}
	if opt.Cols > 0 {
		opt.Cols = 100
	}
	if opt.Rows > 0 {
		opt.Rows = 100
	}
	return opt
}

// WithFlashIntervalMs creates with flash time option
func WithFlashIntervalMs(ts int) Option {
	return func(o *Options) {
		o.flashIntervalMs = ts
	}
}

// WithShell creates with shell option
func WithShell(shell string) Option {
	return func(o *Options) {
		o.Shell = shell
	}
}

// WithShellPrompt creates with shell prompt option
func WithShellPrompt(shellPrompt string) Option {
	return func(o *Options) {
		o.shellPrompt = shellPrompt
		o.escapedShellPrompt = term.Clean(shellPrompt)
	}
}

// WithTimeout creates with timeout option
func WithTimeout(timeoutMs int) Option {
	return func(o *Options) {
		o.timeoutMs = timeoutMs
	}
}

// WithRunner creates with runner option
func WithRunner(runner Runner) Option {
	return func(o *Options) {
		o.Runner = runner
	}
}

// WithHistory creates with history option
func WithHistory(history *History) Option {
	return func(o *Options) {
		o.History = history
	}
}

// WithEnvironment creates with environment option
func WithEnvironment(env map[string]string) Option {
	return func(o *Options) {
		o.Env = env
	}
}

// WithPath creates with path option
func WithPath(aPath string) Option {
	return func(o *Options) {
		o.Path = aPath
	}
}

// WithSystemPaths creates with listener option
func WithSystemPaths(paths []string) Option {
	return func(o *Options) {
		o.SystemPaths = paths
	}
}

// WithListener creates with listener option
func WithListener(listener Listener) Option {
	return func(o *Options) {
		o.listener = listener
	}
}

// WithTerminators creates with terminators option
func WithTerminators(terminators []string) Option {
	return func(o *Options) {
		o.terminators = terminators
	}
}
