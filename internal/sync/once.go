package sync

import (
	"sync"

	"github.com/uber-go/atomic"
)

// Once is a wrapper around sync.Once in order to simplify returning the
// same error multiple times from the same function
type Once struct {
	done atomic.Bool
	once sync.Once
	err  error
}

// Do is a wrapper around the sync.Once `Do` method. This version takes a function that
// returns an error, and every subsequent call to the `Do` function will be returned the
// `err` of the `f` func
func (o *Once) Do(f func() error) error {
	o.once.Do(func() {
		o.err = f()
		o.done.Store(true)
	})

	return o.err
}

// SetDone will complete the `once` sync and will set the `done` flag to true
func (o *Once) SetDone() {
	o.once.Do(func() {})
	o.done.Store(true)
}

// Done returns whether the finished flag has been set and thus sync.Once has been run
func (o *Once) Done() bool {
	return o.done.Load()
}
