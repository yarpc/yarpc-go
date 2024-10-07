package tracinginterceptor

import (
	"github.com/opentracing/opentracing-go"
	"go.uber.org/yarpc/api/transport"
	"runtime"
)

const (
	TracingComponentName = "yarpc"
	Version              = "1.74.0-dev"
)

// ExtractTracingTags extracts common tracing tags from a transport request.
func ExtractTracingTags(req *transport.Request) opentracing.Tags {
	return opentracing.Tags{
		"yarpc.version": Version,
		"go.version":    runtime.Version(),
		"component":     TracingComponentName,
	}
}
