package local

import (
	"context"
	"fmt"
	"github.com/viant/gosh/runner"
	"io"
	"os"
	"os/exec"
	"sync/atomic"
	"time"
)

// Runner represents local runner
type Runner struct {
	inited   uint32
	cmd      *exec.Cmd
	options  *runner.Options
	pipeline *runner.Pipeline
	stdin    io.WriteCloser
	counter  int32
}

// Send sends data to stdin
func (r *Runner) Send(ctx context.Context, data []byte) (int, error) {
	r.ensureCommandStarted()
	if atomic.LoadUint32(&r.inited) == 0 {
		return 0, fmt.Errorf("command not started")
	}
	return r.stdin.Write(data)
}

func (r *Runner) ensureCommandStarted() {
	for i := 0; i < 1000; i++ {
		if atomic.LoadInt32(&r.counter) > 0 {
			break
		}
		time.Sleep(time.Millisecond * 100)
	}
}

// Run runs supplied command
func (r *Runner) Run(ctx context.Context, command string, options ...runner.Option) (string, int, error) {

	if err := r.initIfNeeded(ctx); err != nil {
		return "", 0, err
	}
	if !r.pipeline.Running() {
		return "", 0, r.pipeline.Err()
	}
	r.pipeline.Drain(ctx)

	if r.options.AsPipeline() {
		return r.runAsPipeline(ctx, command, options)
	}

	err := r.runCommand(command)
	atomic.AddInt32(&r.counter, 1)
	if err != nil {
		return "", 0, err
	}
	output, _, code, err := r.pipeline.Read(ctx, options...)
	if r.options.History != nil {
		r.options.History.Commands = append(r.options.History.Commands, runner.NewCommand(command, output, err))
	}
	return output, code, err
}

func (r *Runner) runAsPipeline(ctx context.Context, command string, options []runner.Option) (string, int, error) {
	cmd := runner.EnsureLineTermination(command)
	_, err := r.stdin.Write([]byte(cmd))
	atomic.AddInt32(&r.counter, 1)
	if err != nil {
		return "", -1, err
	}
	err = r.pipeline.Listen(ctx, options...)
	return "", -1, err
}

// PID returns process id
func (r *Runner) PID() int {
	if r.cmd == nil || r.cmd.Process == nil {
		return 0
	}
	return r.cmd.Process.Pid
}

func (r *Runner) runCommand(command string) error {
	var cmd = r.pipeline.FormatCmd(command)
	_, err := r.stdin.Write([]byte(cmd))
	if err != nil {
		return fmt.Errorf("failed to execute command: %v, err: %v", command, err)
	}
	return nil
}

func (r *Runner) initIfNeeded(ctx context.Context) error {
	if !atomic.CompareAndSwapUint32(&r.inited, 0, 1) {
		return nil
	}
	if err := r.init(ctx); err != nil {
		return err
	}
	return nil
}

func (r *Runner) init(ctx context.Context) error {
	r.cmd = exec.Command(r.options.Shell)
	// Apply OS-specific process attributes
	r.cmd.SysProcAttr = newSysProcAttr()

	// Working directory
	if r.options.Path != "" {
		r.cmd.Dir = r.options.Path
	}

	// Environment: start from current env, apply overrides, and extend PATH
	r.cmd.Env = r.buildEnv()
	var err error
	r.stdin, err = r.cmd.StdinPipe()
	if err != nil {
		return err
	}

	stdout, err := r.cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := r.cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := r.cmd.Start(); err != nil {
		return err
	}
	r.pipeline, err = runner.NewPipeline(ctx, r.stdin, stdout, stderr, r.options)
	return err
}

// Close closes runner
func (r *Runner) Close() error {
	if r.cmd.Process != nil {
		r.cmd.Process.Kill()
	}
	if r.pipeline != nil {
		r.pipeline.Close()
	}
	if r.stdin != nil {
		r.stdin.Close()
	}
	return nil
}

// New creates a new local runner
func New(options ...runner.Option) *Runner {
	opts := runner.NewOptions(options)
	return &Runner{options: opts}
}

// buildEnv constructs the environment for the shell process by:
// - starting from the current environment
// - applying any explicit overrides from options.Env
// - appending SystemPaths to PATH in an OS-appropriate way
func (r *Runner) buildEnv() []string {
	base := os.Environ()

	// Convert base to map for easy mutation (preserve key casing for PATH when present)
	envMap := make(map[string]string, len(base))
	order := make([]string, 0, len(base))
	for _, kv := range base {
		if idx := indexOfEqual(kv); idx != -1 {
			k := kv[:idx]
			v := kv[idx+1:]
			envMap[k] = v
			order = append(order, k)
		}
	}

	// Apply explicit env overrides
	if len(r.options.Environ()) > 0 {
		for k, v := range r.options.Env {
			if _, ok := envMap[k]; !ok {
				order = append(order, k)
			}
			envMap[k] = v
		}
	}

	// Extend PATH with SystemPaths if provided
	if len(r.options.SystemPaths) > 0 {
		// Find existing PATH key respecting Windows case-insensitivity
		pathKey := "PATH"
		for k := range envMap {
			if equalFold(k, "PATH") {
				pathKey = k
				break
			}
		}
		sep := string(os.PathListSeparator)
		current := envMap[pathKey]
		// Append system paths to existing PATH
		for _, p := range r.options.SystemPaths {
			if current == "" {
				current = p
			} else {
				current += sep + p
			}
		}
		// If PATH wasn't in base env, ensure order includes it once
		if _, ok := envMap[pathKey]; !ok {
			order = append(order, pathKey)
		}
		envMap[pathKey] = current
	}

	// Rebuild slice preserving original order (with new/overridden keys at end)
	out := make([]string, 0, len(envMap))
	seen := map[string]bool{}
	for _, k := range order {
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, k+"="+envMap[k])
	}
	// Add any keys not in order (shouldn't happen, but be safe)
	for k, v := range envMap {
		if !seen[k] {
			out = append(out, k+"="+v)
		}
	}
	return out
}

func indexOfEqual(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == '=' {
			return i
		}
	}
	return -1
}

func equalFold(a, b string) bool {
	if a == b {
		return true
	}
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca := a[i]
		cb := b[i]
		if ca == cb {
			continue
		}
		// ASCII-only case fold sufficient for env keys
		if 'A' <= ca && ca <= 'Z' {
			ca = ca + ('a' - 'A')
		}
		if 'A' <= cb && cb <= 'Z' {
			cb = cb + ('a' - 'A')
		}
		if ca != cb {
			return false
		}
	}
	return true
}
