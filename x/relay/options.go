package relay

import (
	"github.com/uber-go/tally"
	"go.uber.org/zap"
)

// Option describes a func that can modify options.
type Option interface {
	apply(*options)
}

type optionFunc func(*options)

// opts represents the combined options supplied by the user.
type options struct {
	logger *zap.Logger
	scope  tally.Scope
}

// Scope specifies the scope to be used along
func Scope(s tally.Scope) Option {
	return optionFunc(func(opts *options) {
		opts.scope = s
	})
}

// Logger specifies the logger that should be used to log.
func Logger(logger *zap.Logger) Option {
	return optionFunc(func(opts *options) {
		opts.logger = logger
	})
}

func (f optionFunc) apply(options *options) { f(options) }

// applyOptions creates new opts based on the given options.
func applyOptions(opts ...Option) options {
	options := options{
		logger: zap.NewNop(),
		scope:  tally.NoopScope,
	}
	for _, opt := range opts {
		opt.apply(&options)
	}
	return options
}
