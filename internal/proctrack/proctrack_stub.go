//go:build windows
// +build windows

package proctrack

// On Windows we fall back to a no-op implementation so that the package builds.
// The runner will rely on other mechanisms (to be implemented separately).

func RegisterGroup(pgid int) <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}
