package debug

import "go.uber.org/zap"

// Option is an interface for customizing debug handlers.
type Option interface {
	apply(*options)
}

type optionFunc func(*options)

// opts represents the combined options supplied by the user.
type options struct {
	logger *zap.Logger
	tmpl   templateIface
}

// Logger specifies the logger that should be used to log.
// Default value is noop zap logger.
func Logger(logger *zap.Logger) Option {
	return optionFunc(func(opts *options) {
		opts.logger = logger
	})
}

func tmpl(tmpl templateIface) Option {
	return optionFunc(func(opts *options) {
		opts.tmpl = tmpl
	})
}
func (f optionFunc) apply(options *options) { f(options) }

// applyOptions creates new opts based on the given options.
func applyOptions(opts ...Option) options {
	options := options{
		logger: zap.NewNop(),
		tmpl:   _defaultTmpl,
	}
	for _, opt := range opts {
		opt.apply(&options)
	}
	return options
}
