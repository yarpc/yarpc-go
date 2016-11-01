package internal

import "go.uber.org/yarpc"

func RemoveVariableHeaderKeys(headers yarpc.Headers) yarpc.Headers {
	headers.Del("$tracing$uber-trace-id")
	if headers.Len() == 0 {
		return yarpc.NewHeaders()
	}
	return headers
}

func RemoveVariableMapKeys(headers map[string]string) map[string]string {
	delete(headers, "$tracing$uber-trace-id")
	return headers
}
