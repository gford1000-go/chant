[![Go Doc](https://pkg.go.dev/badge/github.com/gford1000-go/chant.svg)](https://pkg.go.dev/github.com/gford1000-go/chant)
[![Go Report Card](https://goreportcard.com/badge/github.com/gford1000-go/chant)](https://goreportcard.com/report/github.com/gford1000-go/chant)

chant | Creating robust channel based communication
===================================================

Simplified chan behaviour, in particular helping manage behaviour on attempts
to send or receive after the chan has closed.

`Channel` wraps an underlying chan instance, which can be buffered or unbuffered.
Additionally a timeout can be specified for listening on the chan, which is applied
if the supplied `time.Duration` is greater than zero.

Sending to the chan should always be via the `Send` function.

Receiving from the chan should usually be via the `Recv` function, but access to the
underlying chan is provided via `RawChan` for use in range or select scenarios - 
however the function should be called on each use and not to send to the chan.
