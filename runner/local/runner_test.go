package local

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/viant/gosh/runner"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestService_Run(t *testing.T) {
	runner := New()
	output, code, err := runner.Run(context.Background(), "ls /")
	assert.Nil(t, err)
	assert.Equal(t, 0, code)
	assert.Truef(t, len(output) > 0, "output was empty")
	assert.True(t, runner.PID() > 0)

}

func TestService_Run_PipelineWithRqEmptyResult(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires POSIX shell and wc")
	}
	if _, err := exec.LookPath("rq"); err != nil {
		t.Skip("rq is not installed")
	}

	r := New()
	defer func() { _ = r.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output, code, err := r.Run(ctx, `printf '{"items":[]}' | rq '.items[]' | wc -l`, runner.WithTimeout(4000))
	require.NoError(t, err)
	assert.Equal(t, 0, code)
	assert.Equal(t, "0", strings.TrimSpace(output))
}

func TestService_Run_ConcurrentCallsSerialized(t *testing.T) {
	r := New()
	defer func() { _ = r.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	const workers = 8
	var wg sync.WaitGroup
	wg.Add(workers)

	errCh := make(chan error, workers)
	outCh := make(chan string, workers)
	for i := 0; i < workers; i++ {
		i := i
		go func() {
			defer wg.Done()
			out, code, err := r.Run(ctx, "echo worker-"+strconv.Itoa(i), runner.WithTimeout(3000))
			if err != nil {
				errCh <- err
				return
			}
			if code != 0 {
				errCh <- fmt.Errorf("unexpected exit code: %d", code)
				return
			}
			outCh <- strings.TrimSpace(out)
		}()
	}
	wg.Wait()
	close(errCh)
	close(outCh)

	for err := range errCh {
		require.NoError(t, err)
	}
	seen := map[string]bool{}
	for out := range outCh {
		seen[out] = true
	}
	assert.Equal(t, workers, len(seen))
}

func TestServiceRunIncompleteCommandReturnsTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires POSIX shell and sleep")
	}
	r := New()
	t.Cleanup(func() { _ = r.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, code, err := r.Run(ctx, "printf pending; sleep 0.2", runner.WithTimeout(20))
	require.ErrorIs(t, err, runner.ErrTimeout)
	assert.Equal(t, -1, code)
	assert.Equal(t, "pending", out)
}
