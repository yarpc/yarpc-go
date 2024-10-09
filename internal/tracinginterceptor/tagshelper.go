package tracinginterceptor

import (
	"github.com/opentracing/opentracing-go"
	"runtime"
)

const (
	tracingComponentName = "yarpc-go"
)

// Static tracing tags to be used across spans
var commonTracingTags = opentracing.Tags{
	"go.version": runtime.Version(),
	"component":  tracingComponentName,
}

// CommonTracingTags is the exported variable containing static tracing tags.
var CommonTracingTags = commonTracingTags
