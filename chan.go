package chant

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// New creates a wrapped chan of the given type, with or without buffering
// and with the specified Timeout for listening on the chan
func New[T any](opts ...func(*Options)) *Channel[T] {

	o := Options{
		SendRetries: 3,
		SendTimeout: 100 * time.Microsecond,
	}
	for _, opt := range opts {
		opt(&o)
	}

	var ch chan T
	if o.Size == 0 {
		ch = make(chan T)
	} else {
		ch = make(chan T, o.Size)
	}
	return &Channel[T]{
		c:       ch,
		d:       o.RecvTimeout,
		retries: o.SendRetries,
		rd:      o.SendTimeout,
	}
}

// Channel implements a wrapped chan
type Channel[T any] struct {
	l       sync.RWMutex
	c       chan T
	d       time.Duration
	retries int
	rd      time.Duration
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

// ErrUnableToSendRequest returned when a request cannot be sent after multiple attempts
var ErrUnableToSendRequest = errors.New("unable to send request")

// ErrUncaughtSendPanic returned if a send attempt generates a panic
var ErrUncaughtSendPanic = errors.New("recovered panic during send")

// Send will publish the specified value onto the underlying chan,
// unless it is already closed, when an error will be returned.
func (c *Channel[T]) Send(t T) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%w: %v", ErrUncaughtSendPanic, r)
		}
	}()

	c.l.RLock()
	defer c.l.RUnlock()

	if c.c == nil {
		return ErrChannelClosed
	}

	retry := true
	attempts := 0
	maxAttempts := c.retries
	for retry {
		var err error
		submitTimer := acquireTimer(c.rd)

		select {
		case c.c <- t:
			retry = false // only put the req onto the c.c once
		case <-submitTimer.C:
			// There is a possibility that a large number of concurrent Send() calls
			// could fill up c.c before it can be closed.
			// This could mean that a Send() could block indefinitely trying to write to c.c
			// even though the Responder has closed.
			// Retrying should detect done has closed, and so return an error
			//
			// The Send() might also be blocked trying to write to c.c
			// if the receiver is taking too long to process.
			attempts++
			if attempts >= maxAttempts {
				err = ErrUnableToSendRequest
			}
		}

		releaseTimer(submitTimer)

		if err != nil {
			return err
		}
	}

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

	timer := acquireTimer(c.d)
	defer releaseTimer(timer)

	select {
	case v := <-c.c:
		return v, nil
	case <-timer.C:
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
