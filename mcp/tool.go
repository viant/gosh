package mcp

import (
	"context"
	"github.com/viant/gosh"
	"github.com/viant/gosh/runner"
	"github.com/viant/gosh/runner/local"
	"github.com/viant/gosh/runner/ssh"
	"github.com/viant/jsonrpc"
	"github.com/viant/mcp-protocol/schema"
	"github.com/viant/scy"
	"github.com/viant/scy/cred"
	"reflect"
	"strings"
	"sync"
)

// Target specifies a host and its associated credentials
type Target struct {
	Host   string        `json:"host"`
	Secret *scy.Resource `json:"secret"`
}

// Command represents a request to run shell commands on a target
type Command struct {
	Target   *Target           `json:"target,omitempty"`
	Commands []string          `json:"commands"`
	Env      map[string]string `json:"env,omitempty"`
}

type Terminal struct {
	localhost           *gosh.Service
	remote              map[string]*gosh.Service
	mux                 sync.RWMutex
	remoteRunnerOptions []runner.Option
}

func (t *Terminal) TargetTerminal(ctx context.Context, target *Target) (*gosh.Service, error) {
	if target == nil || target.Host == "" || strings.Contains(target.Host, "localhost") {
		return t.localhost, nil
	}
	t.mux.RLock()
	_, ok := t.remote[target.Host]
	t.mux.RUnlock()

	if !ok {

		secretSecret := scy.New()
		target.Secret.SetTarget(reflect.TypeOf(cred.SSH{}))
		secret, err := secretSecret.Load(ctx, target.Secret)
		if err != nil {
			return nil, err
		}
		sshCred, ok := secret.Target.(*cred.SSH)
		if !ok {
			return nil, jsonrpc.NewInternalError("invalid target type", []byte(target.Host))
		}
		sshConfig, err := sshCred.Config(ctx)
		if err != nil {
			return nil, err
		}
		remote := ssh.New(target.Host, sshConfig, t.remoteRunnerOptions...)

		// Create a new remote terminal service
		service, err := gosh.New(ctx, remote)
		if err != nil {
			return nil, err
		}
		t.mux.Lock()
		t.remote[target.Host] = service
		t.mux.Unlock()
		return service, nil
	}

	t.mux.Lock()
	defer t.mux.Unlock()
	_, ok = t.remote[target.Host]
	if ok {
		return t.remote[target.Host], nil
	}
	return t.remote[target.Host], nil
}

func (t *Terminal) Call(ctx context.Context, input *Command) (*schema.CallToolResult, *jsonrpc.Error) {

	term, err := t.TargetTerminal(ctx, input.Target)
	if err != nil {
		return nil, jsonrpc.NewInternalError(err.Error(), []byte(input.Target.Host))
	}

	// Convert commands to a single string command
	cmdString := ""
	if len(input.Commands) > 0 {
		cmdString = input.Commands[0]
		for _, cmd := range input.Commands[1:] {
			cmdString += " && " + cmd
		}
	}
	// prepare runner options (e.g., environment variables)
	var opts []runner.Option
	if len(input.Env) > 0 {
		opts = append(opts, runner.WithEnvironment(input.Env))
	}
	// execute command
	output, code, runErr := term.Run(ctx, cmdString, opts...)
	result := &schema.CallToolResult{
		Content: []schema.CallToolResultContentElem{
			{Text: output, Type: "text"},
		},
	}
	// handle execution errors as tool-level errors
	if runErr != nil {
		isErr := true
		result.IsError = &isErr
		result.Content = append(result.Content, schema.CallToolResultContentElem{
			Text: runErr.Error(), Type: "error",
		})
		return result, nil
	}
	if code != 0 {
		isErr := true
		result.IsError = &isErr
	}
	return result, nil
}

// New creates a new Terminal instance
func New(options ...runner.Option) (*Terminal, error) {
	localTerm, err := gosh.New(context.Background(), local.New())
	if err != nil {
		return nil, err
	}
	return &Terminal{
		localhost:           localTerm,
		remote:              make(map[string]*gosh.Service),
		remoteRunnerOptions: options,
	}, nil
}
