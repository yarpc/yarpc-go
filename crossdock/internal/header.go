package internal

import "go.uber.org/yarpc"

// RemoveVariableHeaderKeys removes any headers that might have been added by tracing
func RemoveVariableHeaderKeys(headers yarpc.Headers) yarpc.Headers {
	headers.Del("$tracing$uber-trace-id")
	if headers.Len() == 0 {
		return yarpc.NewHeaders()
	}
	return headers
}

// RemoveVariableMapKeys removes any headers that might have been added by tracing
func RemoveVariableMapKeys(headers map[string]string) map[string]string {
	delete(headers, "$tracing$uber-trace-id")
	return headers
}
