//go:build !windows
// +build !windows

package proctrack

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// doneCh is closed when the corresponding process group terminates.
type group struct {
	pgid int
	done chan struct{}
	once sync.Once
}

var (
	groups   = make(map[int]*group)
	groupsMu sync.Mutex

	once sync.Once
)

// RegisterGroup registers a process group id and returns a channel that will be
// closed when the operating system reports that every member of that group has
// exited. The caller should launch the command with SysProcAttr.Setpgid = true
// so that pgid equals the parent pid.
func RegisterGroup(pgid int) <-chan struct{} {
	initListener()

	groupsMu.Lock()
	if g, ok := groups[pgid]; ok {
		groupsMu.Unlock()
		return g.done
	}

	g := &group{pgid: pgid, done: make(chan struct{})}
	groups[pgid] = g
	groupsMu.Unlock()

	// Start polling fallback goroutine for this group.
	go pollGroup(g)
	return g.done
}

// initListener sets up a singleton goroutine that reacts to SIGCHLD and checks
// whether registered process groups are still alive.
func initListener() {
	once.Do(func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGCHLD)

		go func() {
			var ws syscall.WaitStatus
			for {
				<-sigCh // wait for a signal

				// Drain all exited children.
				for {
					pid, err := syscall.Wait4(-1, &ws, syscall.WNOHANG, nil)
					if pid <= 0 || err == syscall.ECHILD {
						break
					}
				}

				groupsMu.Lock()
				for pgid, g := range groups {
					if !processGroupAlive(pgid) {
						finishGroup(g)
					}
				}
				groupsMu.Unlock()
			}
		}()
	})
}

// pollGroup periodically checks liveness as a safety-net in case signals are
// missed or the platform does not deliver them (e.g., within some containers).
// It stops automatically once the group is finished.
func pollGroup(g *group) {
	interval := 250 * time.Millisecond
	max := time.Second

	timer := time.NewTimer(interval)
	defer timer.Stop()

	for {
		<-timer.C
		if !processGroupAlive(g.pgid) {
			groupsMu.Lock()
			finishGroup(g)
			groupsMu.Unlock()
			return
		}

		// Exponential back-off up to max.
		if interval < max {
			interval *= 2
			if interval > max {
				interval = max
			}
		}
		timer.Reset(interval)
	}
}

// finishGroup closes the done channel exactly once and removes the group from
// the registry. Caller must hold groupsMu.
func finishGroup(g *group) {
	if _, ok := groups[g.pgid]; !ok {
		return // already finished
	}
	g.once.Do(func() { close(g.done) })
	delete(groups, g.pgid)
}

// processGroupAlive returns true if at least one process in the group is still
// running. It uses the conventional "signal 0" test.
func processGroupAlive(pgid int) bool {
	// syscall.Kill with a negative pid targets the process group.
	if err := syscall.Kill(-pgid, 0); err != nil {
		return false // ESRCH – no such process / group
	}
	return true
}
