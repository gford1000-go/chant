package chant

import "time"

// Options allows the behaviour of Channel to be modified
type Options struct {
	RecvTimeout time.Duration
	Size        int
	SendRetries int
	SendTimeout time.Duration
}

// WithRecvTimeout sets the Recv timeout
// Default: 0 (indefinite wait)
func WithRecvTimeout(d time.Duration) func(*Options) {
	return func(o *Options) {
		if d > o.RecvTimeout {
			o.RecvTimeout = d
		}
	}
}

// WithSize sets the chan buffer size, if zero then will be blocking
// Default: 0
func WithSize(size int) func(*Options) {
	return func(o *Options) {
		if size > o.Size {
			o.Size = size
		}
	}
}

// WithSendRetries sets the number of times a Send() will retry
// adding a message to the chan.
// Default: 3
func WithSendRetries(maxRetries int) func(*Options) {
	return func(o *Options) {
		if maxRetries > o.SendRetries {
			o.SendRetries = maxRetries
		}
	}
}

// WithSendTimeout sets the timeout duration for Send() retries
// Default: 100 * time.Microsecond
func WithSendTimeout(d time.Duration) func(*Options) {
	return func(o *Options) {
		if d > o.SendTimeout {
			o.SendTimeout = d
		}
	}
}
