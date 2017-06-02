package retry

import (
	"context"
	"time"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/ioutil"
)

// MiddlewareOptions enumerates the options for retry middleware.
type MiddlewareOptions struct {
	// Retries is the number of attempts we will retry (after the
	// initial attempt.
	Retries int

	// Timeout is the Timeout we will enforce per request (if this
	// is less than the context deadline, we'll use that instead).
	Timeout time.Duration
}

// NewUnaryMiddleware creates a new Retry Middleware
func NewUnaryMiddleware(opts MiddlewareOptions) *OutboundMiddleware {
	return &OutboundMiddleware{opts}
}

// OutboundMiddleware is a retry middleware that wraps a UnaryOutbound with
// Middleware.
type OutboundMiddleware struct {
	opts MiddlewareOptions
}

// Call implements the middleware.UnaryOutbound interface.
func (r *OutboundMiddleware) Call(ctx context.Context, request *transport.Request, out transport.UnaryOutbound) (resp *transport.Response, err error) {
	rereader, finish := ioutil.NewRereader(request.Body)
	defer finish()

	for i := 0; i < r.opts.Retries+1; i++ {
		request.Body = rereader

		subCtx, cancel := context.WithTimeout(ctx, r.getTimeout(ctx))
		resp, err = out.Call(subCtx, request)
		cancel() // Clear the new ctx immdediately after the call

		if err == nil || !isRetryable(err) {
			return resp, err
		}

		// Reset the rereader so we can do another request.
		if resetErr := rereader.Reset(); resetErr != nil {
			// TODO find a way to wrap the resetErr and the err
			err = resetErr
			return resp, err
		}

		// TODO add backoff semantics
	}
	return resp, err
}

func (r *OutboundMiddleware) getTimeout(ctx context.Context) time.Duration {
	ctxDeadline, ok := ctx.Deadline()
	if !ok {
		return r.opts.Timeout
	}
	now := time.Now()
	if ctxDeadline.After(now.Add(r.opts.Timeout)) {
		return r.opts.Timeout
	}
	return ctxDeadline.Sub(now)
}

func isRetryable(err error) bool {
	return transport.IsUnexpectedError(err) || transport.IsTimeoutError(err)
}
