package chant

import (
	"errors"
	"sync"
	"time"
)

// NewWithTimeout creates a wrapped chan of the given type, with or without buffering
// and with the specified Timeout for listening on the chan
func NewWithTimeout[T any](d time.Duration, size ...int) *Channel[T] {
	return newChannel[T](d, size...)
}

// New creates a wrapped chan of the given type, with or without buffering
func New[T any](size ...int) *Channel[T] {
	return newChannel[T](0, size...)
}

func newChannel[T any](d time.Duration, size ...int) *Channel[T] {
	var ch chan T
	if len(size) == 0 {
		ch = make(chan T)
	} else {
		ch = make(chan T, size[0])
	}
	return &Channel[T]{
		c: ch,
		d: d,
	}
}

// Channel implements a wrapped chan
type Channel[T any] struct {
	l sync.RWMutex
	c chan T
	d time.Duration
}

// Close releases the underlying channel resources
func (c *Channel[T]) Close() error {
	c.l.Lock()
	defer c.l.Unlock()

	if c.c != nil {
		close(c.c)
		c.c = nil
	}
	return nil
}

// ErrChannelClosed raised when a call is made to a closed channel
var ErrChannelClosed = errors.New("chan is closed")

// ErrChannelTimeout raised when Recv times out
var ErrChannelTimeout = errors.New("chan timed out")

// Send will publish the specified value onto the underlying chan,
// unless it is already closed, when an error will be returned.
func (c *Channel[T]) Send(t T) error {
	c.l.RLock()
	defer c.l.RUnlock()

	if c.c == nil {
		return ErrChannelClosed
	}

	c.c <- t
	return nil
}

// Recv will listen on the channel for a value and return it, unless it
// is already closed, when a error will be returned.  If the instance
// includes a timeout for Recv, then that is applied and an error
// raised should the timeout be reached before a value is returned.
func (c *Channel[T]) Recv() (T, error) {
	c.l.RLock()
	defer c.l.RUnlock()

	var t T
	if c.c == nil {
		return t, ErrChannelClosed
	}

	if c.d == 0 {
		return <-c.c, nil
	}

	select {
	case v := <-c.c:
		return v, nil
	case <-time.After(c.d):
		return t, ErrChannelTimeout
	}
}

// RawChan provides access to the underlying chan.  The return
// of this function should be checked for nil prior to use, and
// not be retained as the chan may still be closed: it is
// primarily for use in select.
func (c *Channel[T]) RawChan() chan T {
	c.l.RLock()
	defer c.l.RUnlock()
	return c.c
}
