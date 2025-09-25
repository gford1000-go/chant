package chant

import "time"

type Options struct {
	Timeout time.Duration
	Size    int
}

func WithTimeout(d time.Duration) func(*Options) {
	return func(o *Options) {
		if d > o.Timeout {
			o.Timeout = d
		}
	}
}

func WithSize(size int) func(*Options) {
	return func(o *Options) {
		if size > o.Size {
			o.Size = size
		}
	}
}
