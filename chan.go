package chant

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"
)

// New creates a wrapped chan of the given type, with or without buffering
// and with the specified Timeout for listening on the chan
func New[T any](ctx context.Context, opts ...func(*Options)) *Channel[T] {

	o := Options{
		SendRetries:       3,
		SendTimeout:       100 * time.Microsecond,
		ReceiverDoneChans: []chan struct{}{},
	}
	for _, opt := range opts {
		opt(&o)
	}

	receiversDeclared := len(o.ReceiverDoneChans) > 0

	var ch chan T
	if o.Size == 0 {
		ch = make(chan T)
	} else {
		ch = make(chan T, o.Size)
	}
	return &Channel[T]{
		c:                 ch,
		d:                 o.RecvTimeout,
		retries:           o.SendRetries,
		rd:                o.SendTimeout,
		ctx:               ctx,
		checkForReceivers: receiversDeclared,
		receivers:         o.ReceiverDoneChans,
	}
}

// Channel implements a wrapped chan
type Channel[T any] struct {
	l                 sync.RWMutex
	c                 chan T
	d                 time.Duration
	retries           int
	rd                time.Duration
	ctx               context.Context
	checkForReceivers bool
	receivers         []chan struct{}
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

// ErrContextCompleted returned if the request is being attempted but the context has completed
var ErrContextCompleted = errors.New("context is completed")

// ErrNoReceiversForRequest returned if no receiver chan struct{} remain unclosed
var ErrNoReceiversForRequest = errors.New("all receivers have left")

// Send will publish the specified value onto the underlying chan,
// unless it is already closed, when an error will be returned.
func (c *Channel[T]) Send(ctx context.Context, t T) (err error) {
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

	// Here we are allowing receivers to have notification chans, which
	// they close as they stop listening.
	// Once all receivers have stopped listening, there is no point
	// adding messages to the Channel.
	if c.checkForReceivers {
		var chosen int
		for chosen < len(c.receivers) {
			// Create select cases for all done channels
			cases := make([]reflect.SelectCase, 0, len(c.receivers)+1)

			for _, receiverDoneChan := range c.receivers {
				cases = append(cases, reflect.SelectCase{
					Dir:  reflect.SelectRecv,
					Chan: reflect.ValueOf(receiverDoneChan),
				})
			}

			// Add a default case to make this non-blocking
			cases = append(cases, reflect.SelectCase{
				Dir: reflect.SelectDefault,
			})

			// Check to any done channels are ready
			chosen, _, _ = reflect.Select(cases)

			// If any of the chans have been closed, then one will be chosen, so
			// remove that chan and try again, proceeding only once they have
			// all been closed (and hence default is triggered)
			if chosen < len(c.receivers) {
				c.receivers = append(c.receivers[:chosen], c.receivers[chosen+1:]...)
				chosen = 0
			}
		}

		if len(c.receivers) == 0 {
			return ErrNoReceiversForRequest
		}
	}

	retry := true
	attempts := 0
	maxAttempts := c.retries
	for retry {
		var err error
		submitTimer := acquireTimer(c.rd)

		select {
		case <-c.ctx.Done():
			err = ErrContextCompleted
		case <-ctx.Done():
			err = ErrContextCompleted
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
func (c *Channel[T]) Recv(ctx context.Context) (T, error) {
	c.l.RLock()
	defer c.l.RUnlock()

	var t T
	if c.c == nil {
		return t, ErrChannelClosed
	}

	if c.d == 0 {
		select {
		case v := <-c.c:
			return v, nil
		case <-c.ctx.Done():
			return t, ErrContextCompleted
		case <-ctx.Done():
			return t, ErrContextCompleted
		}
	}

	timer := acquireTimer(c.d)
	defer releaseTimer(timer)

	select {
	case v := <-c.c:
		return v, nil
	case <-c.ctx.Done():
		return t, ErrContextCompleted
	case <-ctx.Done():
		return t, ErrContextCompleted
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
