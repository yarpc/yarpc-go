package sync

import (
	"sync"

	"github.com/uber-go/atomic"
)

// OnceWithError is a wrapper around sync.Once in order to simplify returning the
// same error multiple times from the same function
type OnceWithError struct {
	finished    atomic.Bool
	once        sync.Once
	returnedErr error
}

// IsFinished returns whether the finished flag has been set and thus sync.Once has been run
func (o *OnceWithError) IsFinished() bool {
	return o.finished.Load()
}

// Do is a wrapper around the sync.Once `Do` method. This version takes a function that
// returns an error, and every subsequent call to the `Do` function will be returned the
// `returnedErr` of the `Do` func
func (o *OnceWithError) Do(f func() error) error {
	o.once.Do(func() {
		o.finished.Swap(true)
		o.returnedErr = f()
	})

	return o.returnedErr
}
