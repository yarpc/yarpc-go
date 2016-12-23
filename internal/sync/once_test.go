package sync

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/uber-go/atomic"
)

func TestOnce(t *testing.T) {
	var once Once
	onceCalls := atomic.NewInt32(0)
	expectedErr := errors.New("test error")

	var wait ErrorWaiter
	for i := 0; i < 10; i++ {
		wait.Submit(func() error {
			return once.Do(func() error {
				onceCalls.Inc()
				return expectedErr
			})
		})
	}
	errs := wait.Wait()

	assert.Equal(t, 1, int(onceCalls.Load()), "number of executions of once was not 1")
	for _, err := range errs {
		assert.Equal(t, expectedErr, err)
	}
	assert.True(t, once.Done())
}

func TestOnceNotFinished(t *testing.T) {
	var once Once

	assert.False(t, once.Done())
}

func TestOnceDoWithNil(t *testing.T) {
	var once Once

	once.Do(nil)

	assert.True(t, once.Done())
}
