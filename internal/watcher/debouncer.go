package watcher

import (
	"sync"
	"time"
)

type Debouncer struct {
	interval time.Duration
	timer    *time.Timer
	mu       sync.Mutex
	callback func()
}

func NewDebouncer(interval time.Duration, callback func()) *Debouncer {
	return &Debouncer{
		interval: interval,
		callback: callback,
	}
}

func (d *Debouncer) Trigger() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.timer != nil {
		d.timer.Stop()
	}
	d.timer = time.AfterFunc(d.interval, d.callback)
}

func (d *Debouncer) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.timer != nil {
		d.timer.Stop()
	}
}
