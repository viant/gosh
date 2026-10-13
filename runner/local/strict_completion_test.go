package local

import (
	"context"
	"errors"
	"github.com/stretchr/testify/require"
	"github.com/viant/gosh/runner"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestStrictCompletionQuietAndPartialLines(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX commands")
	}
	for _, strict := range []bool{false, true} {
		r := New(runner.WithStrictCompletion(strict))
		t.Cleanup(func() { _ = r.Close() })
		out, code, err := r.Run(context.Background(), "true", runner.WithTimeout(1000))
		require.NoError(t, err)
		require.Zero(t, code)
		require.Empty(t, out)
		out, code, err = r.Run(context.Background(), "printf value", runner.WithTimeout(1000))
		require.NoError(t, err)
		require.Zero(t, code)
		require.Equal(t, "value", out)
	}
}
func TestStrictCompletionRejectsShellExitAndReuse(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell")
	}
	r := New(runner.WithStrictCompletion(true))
	defer r.Close()
	_, code, err := r.Run(context.Background(), "exit", runner.WithTimeout(1000))
	require.ErrorIs(t, err, runner.ErrIncompleteCommand)
	require.Equal(t, -1, code)
	_, code, err = r.Run(context.Background(), "printf must-not-run")
	require.ErrorIs(t, err, runner.ErrIncompleteCommand)
	require.Equal(t, -1, code)
}
func TestStrictCompletionBoundsContinuousOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell")
	}
	r := New(runner.WithStrictCompletion(true))
	defer r.Close()
	started := time.Now()
	out, code, err := r.Run(context.Background(), "while true; do printf tick; sleep 0.01; done", runner.WithTimeout(120))
	require.ErrorIs(t, err, runner.ErrTimeout)
	require.Equal(t, -1, code)
	require.Contains(t, out, "tick")
	require.Less(t, time.Since(started), 2*time.Second)
}
func TestStrictCompletionCancellationRetainsOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell")
	}
	r := New(runner.WithStrictCompletion(true))
	defer r.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	out, code, err := r.Run(ctx, "printf before-cancel; sleep 2", runner.WithTimeout(2000))
	require.True(t, errors.Is(err, context.DeadlineExceeded))
	require.Equal(t, -1, code)
	require.Contains(t, out, "before-cancel")
}
func TestSessionLifetimeIsIndependentOfCompletedCommands(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell")
	}
	lifetime, stop := context.WithCancel(context.Background())
	defer stop()
	r := New(runner.WithSessionContext(lifetime), runner.WithStrictCompletion(true))
	defer r.Close()
	op, cancel := context.WithCancel(context.Background())
	_, _, err := r.Run(op, "export GOSH_TEST_VALUE=retained")
	require.NoError(t, err)
	cancel()
	out, code, err := r.Run(context.Background(), "printf \"$GOSH_TEST_VALUE\"")
	require.NoError(t, err)
	require.Zero(t, code)
	require.Equal(t, "retained", strings.TrimSpace(out))
	stop()
	_, code, err = r.Run(context.Background(), "printf must-not-run")
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, -1, code)
}
func TestStrictCompletionRejectsPipelineMode(t *testing.T) {
	r := New(runner.AsPipeline(), runner.WithStrictCompletion(true))
	defer r.Close()
	_, code, err := r.Run(context.Background(), "unused")
	require.ErrorContains(t, err, "incompatible")
	require.Equal(t, -1, code)
}
