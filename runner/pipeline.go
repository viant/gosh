package runner

import (
	"bytes"
	"context"
	"fmt"
	"github.com/viant/gosh/term"
	"io"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultTickFrequency = 100
	drainTimeoutMs       = 20
)

type (
	//Pipeline represents a command pipeline
	Pipeline struct {
		err        error
		bufferSize int
		mux        sync.Mutex
		running    int32
		stdout     io.Reader
		stderr     io.Reader
		output     chan string
		error      chan string
		options    *Options
	}
)

// FormatCmd formats command
func (p *Pipeline) FormatCmd(cmd string) string {
	if !strings.HasSuffix(cmd, "\n") {
		cmd += "\n"
	}
	return cmd + "echo 'status:'$?\n"
}

// Drain reads any outstanding output
func (p *Pipeline) Drain(ctx context.Context, opts ...Option) {
	//read any outstanding output
	for {
		_, has, _, _ := p.Read(ctx, WithTimeout(drainTimeoutMs))
		if !has {
			return
		}
	}
}

// Err returns error
func (p *Pipeline) Err() error {
	return p.err
}

// Running returns true if pipeline is running
func (p *Pipeline) Running() bool {
	return atomic.LoadInt32(&p.running) == 1
}

// Close closes pipeline
func (p *Pipeline) Close() (err error) {
	if !atomic.CompareAndSwapInt32(&p.running, 1, 0) {
		return nil
	}
	close(p.output)
	close(p.error)
	if closer, ok := p.stdout.(io.Closer); ok {
		err = closer.Close()
	}
	if closer, ok := p.stderr.(io.Closer); ok {
		if e := closer.Close(); e != nil {
			err = e
		}
	}
	return err
}

func (p *Pipeline) copy(reader io.Reader, dest chan string, notification *sync.WaitGroup) error {
	var written int64 = 0
	buf := make([]byte, p.options.bufferSize)
	var err error
	var bytesRead int
	notification.Done()
	for {
		writer := new(bytes.Buffer)
		if atomic.LoadInt32(&p.running) == 0 {
			return nil
		}
		bytesRead, err = reader.Read(buf)
		if bytesRead > 0 {
			bytesWritten, writeErr := writer.Write(buf[:bytesRead])
			if writeErr != nil {
				return p.closeIfError(writeErr)
			}

			if bytesWritten > 0 {
				written += int64(bytesWritten)
			}

			if bytesRead != bytesWritten {
				return p.closeIfError(io.ErrShortWrite)
			}
			dest <- string(writer.Bytes())
		}
		if err != nil {
			return p.closeIfError(err)
		}
	}
}

func (p *Pipeline) closeIfError(writeError error) error {
	p.err = writeError
	return p.Close()
}

var defaultCode = 0

// Read reads output
func (p *Pipeline) Read(ctx context.Context, opts ...Option) (output string, has bool, code int, err error) {
	options := p.options.Apply(opts)
	timeoutMs := options.timeoutMs
	var hasPrompt, hasTerminator bool
	window := newWindow(options)
	defer window.flush()
	var done int32
	defer atomic.StoreInt32(&done, 1)
	var errOut string
	var hasOutput bool

	var waitTimeMs = 0
	var tickFrequencyMs = defaultTickFrequency
	if tickFrequencyMs > timeoutMs {
		tickFrequencyMs = timeoutMs
	}
	var timeoutDuration = time.Duration(tickFrequencyMs) * time.Millisecond
	out := ""
	var statusCode *int
outer:
	for {
		select {
		case partialOutput := <-p.output:
			waitTimeMs = 0
			if code := p.extractStatusCode(&partialOutput); code != nil {
				statusCode = code
			}

			out += partialOutput
			if statusCode != nil {
				break outer
			}

			hasTerminator = p.hasTerminator(out, options.terminators...)
			if len(partialOutput) > 0 {
				if hasTerminator {
					partialOutput = addLineBreakIfNeeded(partialOutput)
				}
				window.notify(p.removePromptIfNeeded(partialOutput))
			}
			if hasTerminator || len(partialOutput) == 0 {
				break outer
			}
			if code := p.extractStatusCode(&out); code != nil {
				statusCode = code
			}
			if (hasTerminator || statusCode != nil) && len(p.output) == 0 {
				break outer
			}
		case e := <-p.error:
			errOut += e
			window.notify(p.removePromptIfNeeded(e))
			if code := p.extractStatusCode(&out); code != nil {
				statusCode = code
			}
			if (hasTerminator || statusCode != nil) && len(p.error) == 0 {
				break outer
			}
			hasTerminator = p.hasTerminator(errOut, options.terminators...)
			if (hasPrompt || hasTerminator) && len(p.error) == 0 {
				break outer
			}
		case <-ctx.Done():
			// Context was cancelled or timed out
		case <-time.After(timeoutDuration):
			waitTimeMs += tickFrequencyMs
			if waitTimeMs >= timeoutMs {
				break outer
			}
		}
	}
	if errOut != "" {
		err = fmt.Errorf(errOut)
	}

	if len(out) > 0 {
		hasOutput = true
		out = p.removePromptIfNeeded(out)
	}
	if statusCode == nil {
		statusCode = &defaultCode
	}
	return out, hasOutput, *statusCode, err
}

var zeroPos = 0

func (p *Pipeline) extractStatusCode(out *string) *int {
	if index := strings.LastIndex(*out, "\n"); index != -1 {
		var candidate string
		var update *int
		if prevIndex := strings.LastIndex((*out)[:index], "\n"); prevIndex != -1 {
			candidate = strings.TrimSpace(term.Clean((*out)[prevIndex:index]))
			index = prevIndex
			update = &index

		} else {
			candidate = strings.TrimSpace(term.Clean((*out)[:index]))
			update = &zeroPos
		}
		candidate = p.removePromptIfNeeded(candidate)
		if !strings.HasPrefix(candidate, "status:") {
			return nil
		}
		candidate = candidate[7:]
		code, err := strconv.Atoi(candidate)
		if err == nil {
			*out = (*out)[:*update]
			return &code
		}
	}
	return nil
}

func (p *Pipeline) hasPrompt(input string) bool {
	if p.options.shellPrompt == "" {
		return false
	}
	if strings.HasSuffix(input, p.options.shellPrompt) {
		return true
	}
	if p.options.escapedShellPrompt == "" {
		return false
	}
	escapedInput := term.Clean(input)
	return strings.HasSuffix(escapedInput, p.options.escapedShellPrompt)
}

func (p *Pipeline) removePromptIfNeeded(stdout string) string {
	if strings.Contains(stdout, p.options.shellPrompt) {
		stdout = strings.Replace(stdout, p.options.shellPrompt, "", 1)
		var lines = []string{}
		for _, line := range strings.Split(stdout, "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			lines = append(lines, line)
		}
		stdout = strings.Join(lines, "\n")
	}
	return stdout
}

func (p *Pipeline) hasTerminator(input string, terminators ...string) bool {
	if len(p.options.terminators) == 0 {
		return false
	}
	escapedInput := term.Clean(input)
	input = escapedInput
	for _, candidate := range terminators {
		candidateLen := len(candidate)
		if candidateLen == 0 {
			continue
		}
		if candidate[0:1] == "^" && strings.HasPrefix(input, candidate[1:]) {
			return true
		}
		if candidate[candidateLen-1:] == "$" && strings.HasSuffix(input, candidate[:candidateLen-1]) {
			return true
		}
		if strings.Contains(input, candidate) {
			return true
		}
	}
	return false
}

func (p *Pipeline) init(ctx context.Context, input io.WriteCloser) error {
	started := sync.WaitGroup{}
	started.Add(2)
	go p.copy(p.stdout, p.output, &started)
	go p.copy(p.stderr, p.error, &started)
	started.Wait()
	if p.options.shellPrompt == "" {
		return nil
	}
	p.Drain(ctx)
	cmd := `PS1="` + p.options.shellPrompt + "\"\n"
	_, err := input.Write([]byte(cmd))
	if err == nil {
		p.Read(ctx, WithTimeout(600))
	}
	return err
}

func addLineBreakIfNeeded(text string) string {
	index := strings.LastIndex(text, "\n")
	if index == -1 {
		return text + "\n"
	}
	lastFragment := string(text[index:])
	if strings.TrimSpace(lastFragment) != "" {
		return text + "\n"
	}
	return text
}

// NewPipeline creates a new pipeline
func NewPipeline(ctx context.Context, in io.WriteCloser, stdout io.Reader, stderr io.Reader, options *Options) (*Pipeline, error) {
	ret := &Pipeline{
		running: 1,
		options: options,
		stdout:  stdout,
		stderr:  stderr,
		output:  make(chan string, 1),
		error:   make(chan string, 1),
	}
	return ret, ret.init(ctx, in)
}
