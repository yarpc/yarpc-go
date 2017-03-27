package sync

import (
	"go.uber.org/yarpc/api/transport"
)

// NewNopLifecycle returns a new one-time no-op lifecycle
func NewNopLifecycle() transport.Lifecycle {
	return &nopLifecycle{once: Once()}
}

type nopLifecycle struct {
	once LifecycleOnce
}

func (n *nopLifecycle) Start() error {
	return n.once.Start(nil)
}

func (n *nopLifecycle) Stop() error {
	return n.once.Stop(nil)
}

func (n *nopLifecycle) IsRunning() bool {
	return n.once.IsRunning()
}
