package sync

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/uber-go/atomic"
)

func TestOnce(t *testing.T) {
	once := Once{}
	onceCalls := atomic.NewInt32(0)
	expectedErr := errors.New("test error")

	wait := ErrorWaiter{}
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
	once := Once{}

	assert.False(t, once.Done())
}

func TestOnceSetDone(t *testing.T) {
	once := Once{}

	once.SetDone()

	assert.True(t, once.Done())
}
