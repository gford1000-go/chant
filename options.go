package chant

import "time"

// Options allows the behaviour of Channel to be modified
type Options struct {
	RecvTimeout time.Duration
	Size        int
}

// WithRecvTimeout sets the Recv timeout
func WithRecvTimeout(d time.Duration) func(*Options) {
	return func(o *Options) {
		if d > o.RecvTimeout {
			o.RecvTimeout = d
		}
	}
}

// WithSize sets the chan buffer size, if zero then will be blocking
func WithSize(size int) func(*Options) {
	return func(o *Options) {
		if size > o.Size {
			o.Size = size
		}
	}
}
