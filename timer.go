package chant

import (
	"sync"
	"time"
)

// time.Timer instances (embedded in time.After) are memory heavy
// so use a Pool to reuse and reduce allocation overhead
var timerPool = sync.Pool{
	New: func() any { return time.NewTimer(0) },
}

func acquireTimer(d time.Duration) *time.Timer {
	t := timerPool.Get().(*time.Timer)
	t.Reset(d)
	return t
}

func releaseTimer(t *time.Timer) {
	if !t.Stop() {
		// Use select with default to avoid blocking
		select {
		case <-t.C:
		default:
		}
	}
	timerPool.Put(t)
}
