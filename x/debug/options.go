package debug

import "go.uber.org/zap"

// Option describes a func that can modify options.
type Option interface {
	apply(*options)
}

type optionFunc func(*options)

// opts represents the combined options supplied by the user.
type options struct {
	logger   *zap.Logger
	template templateIface
}

// Logger specifies the logger that should be used to log.
func Logger(logger *zap.Logger) Option {
	return optionFunc(func(opts *options) {
		opts.logger = logger
	})
}

// Template specifies the template to be used for debug pages.
func Template(template templateIface) Option {
	return optionFunc(func(opts *options) {
		opts.template = template
	})
}

func (f optionFunc) apply(options *options) { f(options) }

// applyOptions creates new opts based on the given options.
func applyOptions(opts ...Option) options {
	options := options{
		logger:   zap.NewNop(),
		template: _defaultTmpl,
	}
	for _, opt := range opts {
		opt.apply(&options)
	}
	return options
}
