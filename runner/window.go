package runner

import "time"

type window struct {
	checkpoint *time.Time
	elapsedMs  int
	stdout     string
	options    *Options
}

func (t *window) flush() {
	if t.options.listener == nil {
		return
	}
	if t.stdout != "" {
		t.options.listener(t.stdout, true)
	}
	t.options.listener("", false)
}

func (t *window) notify(stdout string) {
	var now = time.Now()
	if t.options.listener == nil {
		return
	}
	t.stdout += stdout
	t.elapsedMs += int(now.Sub(*t.checkpoint) / time.Millisecond)
	t.checkpoint = &now
	if t.elapsedMs > t.options.flashIntervalMs || t.options.flashIntervalMs == 0 {
		t.options.listener(t.stdout, true)
		t.stdout = ""
		t.elapsedMs = 0
	}
}

func newWindow(options *Options) *window {
	var now = time.Now()
	return &window{
		checkpoint: &now,
		options:    options,
	}
}
