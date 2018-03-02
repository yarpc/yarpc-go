package relay

import (
	"context"

	"go.uber.org/multierr"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/iopool"
	"go.uber.org/zap"
)

type handlerOpts struct {
	serviceName string
	logger      *zap.Logger
}

func newHandlerOpts() handlerOpts {
	return handlerOpts{
		logger: zap.NewNop(),
	}
}

// HandlerOption are options for configuring a Proxy.
type HandlerOption interface {
	apply(opts *handlerOpts)
}

type handlerOptionFunc func(opts *handlerOpts)

func (u handlerOptionFunc) apply(opts *handlerOpts) { u(opts) }

// WithLogger overrides the logger used to log errors in the service.
func WithLogger(logger *zap.Logger) HandlerOption {
	return handlerOptionFunc(func(opts *handlerOpts) {
		opts.logger = logger
	})
}

// WithServiceName overrides the service name for outbound calls.
func WithServiceName(service string) HandlerOption {
	return handlerOptionFunc(func(opts *handlerOpts) {
		opts.serviceName = service
	})
}

// UnaryProxyHandler creates a unary proxy handler to redirect traffic
// to the specified outbound
func UnaryProxyHandler(out transport.UnaryOutbound, options ...HandlerOption) transport.UnaryHandler {
	opts := newHandlerOpts()
	for _, option := range options {
		option.apply(&opts)
	}
	return &unaryProxyHandler{out: out, opts: opts}
}

// unaryProxyHandler implements the transport.UnaryHandler interface and routes
// all requests to the UnaryOutbound.
type unaryProxyHandler struct {
	out  transport.UnaryOutbound
	opts handlerOpts
}

// Handle implements YARPC's transport.UnaryHandler interface.
func (p *unaryProxyHandler) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter) error {
	if p.opts.serviceName != "" {
		req.Service = p.opts.serviceName
	}
	req.RoutingKey = ""
	req.RoutingDelegate = ""

	resp, err := p.out.Call(ctx, req)
	if err != nil {
		p.opts.logger.Error(
			"error proxying unary request",
			zap.String("caller", req.Caller),
			zap.String("service", req.Service),
			zap.String("procedure", req.Procedure),
			zap.String("shardkey", req.ShardKey),
			zap.Error(err),
		)
		return err
	}

	if resp.ApplicationError {
		resw.SetApplicationError()
	}

	resw.AddHeaders(resp.Headers)

	_, err = iopool.Copy(resw, resp.Body)
	err = multierr.Append(err, resp.Body.Close())
	return err
}

// OnewayProxyHandler creates a oneway proxy handler to redirect traffic
// to the specified outbound.
func OnewayProxyHandler(out transport.OnewayOutbound, options ...HandlerOption) transport.OnewayHandler {
	opts := newHandlerOpts()
	for _, option := range options {
		option.apply(&opts)
	}
	return &onewayProxyHandler{out: out, opts: opts}
}

// onewayProxyHandler implements the transport.OnewayHandler interface and
// routes all requests to the OnewayOutbound.
type onewayProxyHandler struct {
	out  transport.OnewayOutbound
	opts handlerOpts
}

// HandleOneway implements YARPC's transport.OnewayHandler interface.
func (p *onewayProxyHandler) HandleOneway(ctx context.Context, req *transport.Request) error {
	if p.opts.serviceName != "" {
		req.Service = p.opts.serviceName
	}
	req.RoutingKey = ""
	req.RoutingDelegate = ""

	_, err := p.out.CallOneway(ctx, req)
	if err != nil {
		p.opts.logger.Error(
			"error proxying oneway request",
			zap.String("caller", req.Caller),
			zap.String("service", req.Service),
			zap.String("procedure", req.Procedure),
			zap.String("shardkey", req.ShardKey),
			zap.Error(err),
		)
	}
	return err
}
