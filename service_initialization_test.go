package gosh_test

import (
	"context"
	"errors"
	"testing"

	"github.com/viant/gosh"
	"github.com/viant/gosh/runner"
)

type failingProbeRunner struct {
	commands []string
	failAt   int
	err      error
}

func (r *failingProbeRunner) Run(_ context.Context, command string, _ ...runner.Option) (string, int, error) {
	r.commands = append(r.commands, command)
	if len(r.commands) == r.failAt {
		return "", 0, r.err
	}
	return "linux", 0, nil
}

func (*failingProbeRunner) Send(context.Context, []byte) (int, error) { return 0, nil }
func (*failingProbeRunner) PID() int                                  { return 0 }
func (*failingProbeRunner) Close() error                              { return nil }

func TestServiceInitializationStopsAtFirstProbeError(t *testing.T) {
	for _, failAt := range []int{1, 2} {
		t.Run([]string{"system", "hardware"}[failAt-1], func(t *testing.T) {
			original := errors.New("runner initialization failed")
			probe := &failingProbeRunner{failAt: failAt, err: original}
			_, err := gosh.New(context.Background(), probe)
			if err != original {
				t.Fatalf("got error %v, want original error %v", err, original)
			}
			if len(probe.commands) != failAt {
				t.Fatalf("continued probing after failure: %v", probe.commands)
			}
		})
	}
}
