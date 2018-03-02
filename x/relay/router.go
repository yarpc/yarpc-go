package relay

import (
	"context"
	"fmt"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/zap"
)

var (
	_caller    = "Caller"
	_component = "Component"
	_frontCar  = "FrontCar"
	_procedure = "Procedure"
	_service   = "Service"
	_shardKey  = "ShardKey"
)

// ServiceHandler is a handler that gets routed all requests for a specific service.
type ServiceHandler struct {
	// Service is the inbound `service` name that will determine if
	// requests are routed to this handler.
	Service string

	// HandlerSpec that will be used for all calls to the service.
	HandlerSpec transport.HandlerSpec

	// Signature to allow introspection into the Handler (for debugging).
	Signature string
}

// ShardKeyHandler is a handler that gets routed all requests with a shard key.
type ShardKeyHandler struct {
	// ShardKey is the inbound `shard key` name that will determine if
	// requests are routed to this handler.
	ShardKey string

	// HandlerSpec that will be used for all calls to the service.
	HandlerSpec transport.HandlerSpec

	// Signature to allow introspection into the Handler (for debugging).
	Signature string
}

// Router is a YARPC Router Middleware that
// can be used to create service-based routing for requests.
// The Router's routes will only be used if there is no
// procedure available from the `regular` transport.Router
type Router struct {
	// serviceRoutes is a map from service name to associated transport.Procedure.
	// The procedures in this map are only used for service-based routing.
	serviceRoutes map[string]transport.Procedure

	// shardRoutes is a map from shard key to associated transport.Procedure.
	// The procedures in this map are only used for shard-based routing.
	shardRoutes map[string]transport.Procedure

	// hasDefault indicates that there is a default Route that will be used in
	// fallback cases.
	hasDefault bool

	// defaultRoute is the fallback procedure that will be called if a request
	// does not meet any of the serviceRoutes or shardRoutes.
	defaultRoute transport.Procedure

	// logger is a zap logger used for logging events / errors into.
	logger *zap.Logger

	// observer is a struct for recording metrics to tally.
	observer *observer
}

// NewRouter creates a new Router.
func NewRouter(options ...Option) *Router {
	opts := applyOptions(options...)

	return &Router{
		serviceRoutes: make(map[string]transport.Procedure),
		shardRoutes:   make(map[string]transport.Procedure),
		logger:        opts.logger,
		observer:      newObserver(opts.scope),
	}
}

// RegisterService registers service handlers that will be used for calls to the
// specified services.
func (r *Router) RegisterService(handlers []ServiceHandler) {
	for _, handler := range handlers {
		if _, ok := r.serviceRoutes[handler.Service]; ok {
			panic(fmt.Errorf("service %q already registered on Router", handler.Service))
		}

		r.serviceRoutes[handler.Service] = transport.Procedure{
			Name:        "*", // `*` means that we are a proxy
			Service:     handler.Service,
			HandlerSpec: handler.HandlerSpec,
			Signature:   handler.Signature,
		}
	}
}

// RegisterShard registeres shard handlers that will be used for calls to the
// specified shard.
func (r *Router) RegisterShard(handlers []ShardKeyHandler) {
	for _, handler := range handlers {
		if _, ok := r.shardRoutes[handler.ShardKey]; ok {
			panic(fmt.Errorf("shard key %q already registered on Router", handler.ShardKey))
		}

		r.shardRoutes[handler.ShardKey] = transport.Procedure{
			Name:        "*", // `*` means that we are a proxy
			Service:     "*", // `*` means that we are a proxy
			HandlerSpec: handler.HandlerSpec,
			Signature:   handler.Signature,
		}
	}
}

// RegisterDefault registeres the default handler that will be used when no
// service or shard matches.
func (r *Router) RegisterDefault(proc transport.Procedure) {
	r.hasDefault = true
	r.defaultRoute = proc
}

// Procedures returns a list of supported procedures.  This includes procedures
// from the passed in router as well as the procedures we've created for the
// Router
func (r *Router) Procedures(router transport.Router) []transport.Procedure {
	procs := router.Procedures()
	for _, v := range r.serviceRoutes {
		procs = append(procs, v)
	}
	for _, v := range r.shardRoutes {
		procs = append(procs, v)
	}
	return procs
}

// Choose returns a HandlerSpec for each request.  If the procedure is not
// known, a HandlerSpec will be returned that routes the request to a known
// service registered with RegisterService. If no known service knows how to
// handle the request, we will look if there is a procedure for the shard key,
// if there is no shard key registered, an error will be returned.
func (r *Router) Choose(ctx context.Context, req *transport.Request, router transport.Router) (transport.HandlerSpec, error) {
	r.logger.Debug("Choosing a router.",
		zap.String(_service, req.Service),
		zap.String(_caller, req.Caller),
		zap.String(_component, _frontCar),
		zap.String(_procedure, req.Procedure),
		zap.String(_shardKey, req.ShardKey),
	)
	r.observer.call()
	handlerSpec, err := router.Choose(ctx, req)
	if err == nil {
		r.logger.Debug("Service-Procedure match found.",
			zap.String(_service, req.Service),
			zap.String(_caller, req.Caller),
			zap.String(_component, _frontCar),
			zap.String(_procedure, req.Procedure),
			zap.String(_shardKey, req.ShardKey),
		)
		r.observer.serviceProcedureMatch()
		return handlerSpec, nil
	}

	// If the error is not UnrecognizedProcedure, return it immediately.
	if !transport.IsUnrecognizedProcedureError(err) {
		r.logger.Error("Unknown error.",
			zap.String(_service, req.Service),
			zap.String(_caller, req.Caller),
			zap.String(_component, _frontCar),
			zap.String(_procedure, req.Procedure),
			zap.String(_shardKey, req.ShardKey),
			zap.Error(err),
		)
		r.observer.unknownError()
		return handlerSpec, err
	}

	if proc, ok := r.serviceRoutes[req.Service]; ok {
		r.logger.Debug("Service match found.",
			zap.String(_service, req.Service),
			zap.String(_caller, req.Caller),
			zap.String(_component, _frontCar),
			zap.String(_procedure, req.Procedure),
			zap.String(_shardKey, req.ShardKey),
		)
		r.observer.serviceMatch()
		return proc.HandlerSpec, nil
	}

	if proc, ok := r.shardRoutes[req.ShardKey]; ok {
		r.logger.Debug("Shardkey match found.",
			zap.String(_service, req.Service),
			zap.String(_caller, req.Caller),
			zap.String(_component, _frontCar),
			zap.String(_procedure, req.Procedure),
			zap.String(_shardKey, req.ShardKey),
		)
		r.observer.shardKeyMatch()
		return proc.HandlerSpec, nil
	}

	if r.hasDefault {
		r.logger.Debug("No match found. Using default.",
			zap.String(_service, req.Service),
			zap.String(_caller, req.Caller),
			zap.String(_component, _frontCar),
			zap.String(_procedure, req.Procedure),
			zap.String(_shardKey, req.ShardKey),
		)
		r.observer.defaultMatch()
		return r.defaultRoute.HandlerSpec, nil
	}

	r.logger.Error("No handler found.",
		zap.String(_service, req.Service),
		zap.String(_caller, req.Caller),
		zap.String(_component, _frontCar),
		zap.String(_procedure, req.Procedure),
		zap.String(_shardKey, req.ShardKey),
		zap.Error(err),
	)
	r.observer.noHandleError()
	return transport.HandlerSpec{}, err
}
