package sync

// LifecycleOnce is an abstraction around a lifecycle interface
type LifecycleOnce struct {
	start Once
	stop  Once
}

// Start will run the `f` function once and return the error
func (l *LifecycleOnce) Start(f func() error) error {
	return l.start.Do(f)
}

// SetStarted will set the start `Once` flag to true
func (l *LifecycleOnce) SetStarted() {
	l.start.SetDone()
}

// Stop will run the `f` function once and return the error
func (l *LifecycleOnce) Stop(f func() error) error {
	return l.stop.Do(f)
}

// SetStopped will set the stop `Once` flag to true
func (l *LifecycleOnce) SetStopped() {
	l.stop.SetDone()
}

// IsRunning will return true if the start has been run, and the stop has not
func (l *LifecycleOnce) IsRunning() bool {
	return l.start.Done() && !l.stop.Done()
}
