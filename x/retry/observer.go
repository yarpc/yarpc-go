package retry

import "github.com/uber-go/tally"

type observer struct {
	unretryableErrorCounter tally.Counter
	yarpcErrorCounter  tally.Counter
	noTimeErrorCounter      tally.Counter
	maxAttemptsErrorCounter tally.Counter
	successCounter          tally.Counter
	callCounter          tally.Counter
}

func newObserver(scope tally.Scope) *observer {
	unretryableErrScope := scope.Tagged(map[string]string{"error": "unretryable"})
	yarpcErrScope := scope.Tagged(map[string]string{"error": "yarpc"})
	noTimeErrScope := scope.Tagged(map[string]string{"error": "notime"})
	maxAttemptsErrScope := scope.Tagged(map[string]string{"error": "max_attempts"})
	return &observer{
		unretryableErrorCounter: unretryableErrScope.Counter("retry_failures"),
		yarpcErrorCounter:  yarpcErrScope.Counter("retry_failures"),
		noTimeErrorCounter:      noTimeErrScope.Counter("retry_failures"),
		maxAttemptsErrorCounter: maxAttemptsErrScope.Counter("retry_failures"),
		successCounter:          scope.Counter("retry_successes"),
		callCounter:          scope.Counter("retry_calls"),
	}
}

func (o *observer) unretryableError() {
	o.unretryableErrorCounter.Inc(1)
}

func (o *observer) yarpcError() {
	o.yarpcErrorCounter.Inc(1)
}

func (o *observer) noTimeError() {
	o.noTimeErrorCounter.Inc(1)
}

func (o *observer) maxAttemptsError() {
	o.maxAttemptsErrorCounter.Inc(1)
}

func (o *observer) success() {
	o.successCounter.Inc(1)
}

func (o *observer) call() {
	o.callCounter.Inc(1)
}
