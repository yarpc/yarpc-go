package metricstagdecorators

// DecoratorProperties is used for passing request-specific properties
// to metrics decorators.
type DecoratorProperties struct{}

// MetricsTagsDecorators is used for adding custom tags to YARPC metrics.
type MetricsTagsDecorators interface {
	ProvideTags(request DecoratorProperties) map[string]string
}
