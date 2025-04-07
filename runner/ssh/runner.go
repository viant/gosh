package ssh

import (
	"context"
	"fmt"
	"github.com/viant/gosh/runner"
	"golang.org/x/crypto/ssh"
	"io"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// Runner represents ssh runner
type Runner struct {
	inited   uint32
	client   *ssh.Client
	session  *ssh.Session
	host     string
	config   *ssh.ClientConfig
	options  *runner.Options
	pipeline *runner.Pipeline
	stdin    io.WriteCloser
	pid      int
}

// Stdin returns stdin writer
func (r *Runner) Stdin() io.Writer {
	return r.stdin
}

func (r *Runner) connect() (err error) {
	if r.client, err = ssh.Dial("tcp", r.host, r.config); err != nil {
		return fmt.Errorf("failed to dial: %v, %w", r.host, err)
	}
	return err
}

func (r *Runner) Close() (err error) {
	if r.pipeline != nil {
		_ = r.pipeline.Close()
	}
	if r.session != nil {
		_ = r.session.Close()
	}
	if r.client != nil {
		err = r.client.Close()
	}
	return err
}

func (r *Runner) start(ctx context.Context) (err error) {
	r.session, err = r.client.NewSession()
	for k, v := range r.options.Env {
		err = r.session.Setenv(k, v)
		if err != nil {
			return nil
		}
	}
	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // stdin speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	if err := r.session.RequestPty(r.options.Term, r.options.Rows, r.options.Cols, modes); err != nil {
		return err
	}
	if r.stdin, err = r.session.StdinPipe(); err != nil {
		return err
	}
	outPipe, err := r.session.StdoutPipe()
	if err != nil {
		return err
	}
	errPipe, err := r.session.StderrPipe()
	if err != nil {
		return err
	}
	if err = r.session.Start(r.options.Shell); err != nil {
		return err
	}
	r.pipeline, err = runner.NewPipeline(ctx, r.stdin, outPipe, errPipe, r.options)
	if err != nil {
		return err
	}
	var pid string
	pid, _, err = r.Run(ctx, "echo $$")
	if err == nil {
		pid = strings.TrimSpace(pid)
		r.pid, err = strconv.Atoi(pid)
	}
	if r.options.Path != "" {
		_, _, err = r.Run(ctx, "cd "+r.options.Path)
	}
	if len(r.options.SystemPaths) > 0 {
		_, _, err = r.Run(ctx, "export PATH=$PATH:"+strings.Join(r.options.SystemPaths, ":"))
	}

	return err
}

// PID returns process id
func (r *Runner) PID() int {
	return r.pid
}
func (r *Runner) init(ctx context.Context) (err error) {
	if r.client == nil {
		if err = r.connect(); err != nil {
			return err
		}
	}
	defer func() {
		if err != nil {
			r.client.Close()
		}
	}()
	err = r.start(ctx)
	return err
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
	err := r.runCommand(command)
	if err != nil {
		return "", 0, err
	}
	output, _, code, err := r.pipeline.Read(ctx, options...)
	if r.options.History != nil {
		r.options.History.Commands = append(r.options.History.Commands, runner.NewCommand(command, output, err))
	}
	return output, code, err
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

// New creates a new runner
func New(host string, config *ssh.ClientConfig, opts ...runner.Option) *Runner {
	opts = append([]runner.Option{runner.WithShellPrompt("shh-" + strconv.Itoa(int(time.Now().UnixMilli())) + "$")}, opts...)
	return &Runner{
		host:    host,
		config:  config,
		options: runner.NewOptions(opts),
	}
}
