package sync

import (
	"sync"

	"github.com/uber-go/atomic"
)

// Once is a wrapper around sync.Once in order to simplify returning the
// same error multiple times from the same function.
type Once struct {
	done atomic.Bool
	once sync.Once
	err  error
}

// Do is a wrapper around the sync.Once `Do` method. This version takes a function that
// returns an error, and every subsequent call to the `Do` function will be returned the
// `err` of the `f` func.
// If f is nil we will replace it with a noop function.
func (o *Once) Do(f func() error) error {
	if f == nil {
		f = func() error { return nil }
	}

	o.once.Do(func() {
		o.err = f()
		o.done.Store(true)
	})

	return o.err
}

// Done returns whether the finished flag has been set and thus sync.Once has been run.
func (o *Once) Done() bool {
	return o.done.Load()
}
