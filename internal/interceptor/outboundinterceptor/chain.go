package outboundinterceptor

import (
	"go.uber.org/yarpc/internal/interceptor"
)

// UnaryChain combines a series of `UnaryOutbound`s into a single `UnaryOutbound`.
func UnaryChain(mw ...interceptor.UnaryOutbound) interceptor.UnaryOutbound {
	// TODO: implement
	return interceptor.NopUnaryOutbound
}
