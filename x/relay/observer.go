package relay

import "github.com/uber-go/tally"

var (
	_callsName           = "frontcar_choose_calls"
	_successesName       = "frontcar_choose_successes"
	_failureName         = "frontcar_choose_failures"
	_matchType           = "match_type"
	_errorType           = "error_type"
	_serviceProcedureTag = "service_procedure"
	_serviceTag          = "service"
	_shardKeyTag         = "shard_key"
	_defaultTag          = "default"
	_unknownTag          = "unknown"
	_noHandlerTag        = "no_handler"
)

type observer struct {
	calls                   tally.Counter
	serviceProcedureMatches tally.Counter
	serviceMatches          tally.Counter
	shardKeyMatches         tally.Counter
	defaultMatches          tally.Counter
	noHandlerErrs           tally.Counter
	unknownErrs             tally.Counter
}

func newObserver(scope tally.Scope) *observer {
	serviceProcedureMatchScope := scope.Tagged(map[string]string{_matchType: _serviceProcedureTag})
	serviceMatchScope := scope.Tagged(map[string]string{_matchType: _serviceTag})
	shardKeyMatchScope := scope.Tagged(map[string]string{_matchType: _shardKeyTag})
	defaultMatchScope := scope.Tagged(map[string]string{_matchType: _defaultTag})
	unknownTagErrScope := scope.Tagged(map[string]string{_errorType: _unknownTag})
	noHandlerTagScope := scope.Tagged(map[string]string{_errorType: _noHandlerTag})

	return &observer{
		calls: scope.Counter(_callsName),
		serviceProcedureMatches: serviceProcedureMatchScope.Counter(_successesName),
		serviceMatches:          serviceMatchScope.Counter(_successesName),
		shardKeyMatches:         shardKeyMatchScope.Counter(_successesName),
		defaultMatches:          defaultMatchScope.Counter(_successesName),
		unknownErrs:             unknownTagErrScope.Counter(_failureName),
		noHandlerErrs:           noHandlerTagScope.Counter(_failureName),
	}
}

func (o *observer) call() {
	o.calls.Inc(1)
}

func (o *observer) serviceProcedureMatch() {
	o.serviceProcedureMatches.Inc(1)
}

func (o *observer) serviceMatch() {
	o.serviceMatches.Inc(1)
}

func (o *observer) shardKeyMatch() {
	o.shardKeyMatches.Inc(1)
}

func (o *observer) defaultMatch() {
	o.defaultMatches.Inc(1)
}

func (o *observer) unknownError() {
	o.unknownErrs.Inc(1)
}

func (o *observer) noHandleError() {
	o.noHandlerErrs.Inc(1)
}
