package local

import (
	"context"
	"fmt"
	"github.com/viant/gosh/runner"
	"io"
	"os/exec"
	"strings"
	"sync/atomic"
	"syscall"
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

	r.cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	r.cmd.Env = r.options.Environ()
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
	if r.options.Path != "" {
		_, _, err = r.Run(ctx, "cd "+r.options.Path)
	}
	if len(r.options.SystemPaths) > 0 {
		_, _, err = r.Run(ctx, "export PATH=$PATH:"+strings.Join(r.options.SystemPaths, ":"))
	}
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
