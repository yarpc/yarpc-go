package sync

// LifecycleOnce is a helper for implementing transport.Lifecycles
// with similar behavior.
type LifecycleOnce struct {
	start Once
	stop  Once
}

// Start will run the `f` function once and return the error
func (l *LifecycleOnce) Start(f func() error) error {
	return l.start.Do(f)
}

// Stop will run the `f` function once and return the error
func (l *LifecycleOnce) Stop(f func() error) error {
	return l.stop.Do(f)
}

// IsRunning will return true if the start has been run, and the stop has not
func (l *LifecycleOnce) IsRunning() bool {
	return l.start.Done() && !l.stop.Done()
}
