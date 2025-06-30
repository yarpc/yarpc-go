package metricstagdecorators

type DecoratorProperties struct{}

// MetricsTagsDecorators is used to add custom tags to YARPC metrics.
type MetricsTagsDecorators interface {
	ProvideTags(request DecoratorProperties) map[string]string
}
