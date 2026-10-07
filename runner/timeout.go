package runner

import "errors"

// ErrTimeout means the read became idle before receiving a completion marker or
// configured terminator. The returned exit code is -1 (unknown), and any output
// received before the timeout remains available. It does not stop the shell command.
var ErrTimeout = errors.New("shell completion timed out")
