package http

import (
	"context"
	"net/http"
)

type httpRequestKey struct{} // context key for the *http.Request

func withRequest(ctx context.Context, req *http.Request) context.Context {
	return context.WithValue(ctx, httpRequestKey{}, req)
}

func requestFromContext(ctx context.Context) (_ *http.Request, ok bool) {
	req, ok := ctx.Value(httpRequestKey{}).(*http.Request)
	return req, ok
}
