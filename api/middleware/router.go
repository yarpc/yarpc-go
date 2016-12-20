package middleware

import (
	"context"

	"go.uber.org/yarpc/api/transport"
)

//go:generate mockgen -destination=middlewaretest/router.go -package=middlewaretest go.uber.org/yarpc/api/middleware Router

// Router is a middleware for defining a customized routing experience for procedures
type Router interface {
	// Procedures returns the list of procedures that can be called on this router.
	// Procedures MUST call into router that is passed in.
	Procedures(transport.Router) []transport.Procedure

	// Choose returns a handlerspec for the given request and transport.
	// If the Router cannot determine what to call it should call into the router that was
	// passed in.
	Choose(context.Context, *transport.Request, transport.Router) (transport.HandlerSpec, error)
}

// ApplyRouter applies the given Router middleware to
// the given Router.
func ApplyRouter(r transport.RouteTable, m Router) transport.RouteTable {
	if m == nil {
		return r
	}
	return routeTableWithMiddleware{r: r, m: m}
}

type routeTableWithMiddleware struct {
	r transport.RouteTable
	m Router
}

func (r routeTableWithMiddleware) Procedures() []transport.Procedure {
	return r.m.Procedures(r.r)
}

func (r routeTableWithMiddleware) Choose(ctx context.Context, req *transport.Request) (transport.HandlerSpec, error) {
	return r.m.Choose(ctx, req, r.r)
}

func (r routeTableWithMiddleware) Register(procedures []transport.Procedure) {
	r.r.Register(procedures)
}
