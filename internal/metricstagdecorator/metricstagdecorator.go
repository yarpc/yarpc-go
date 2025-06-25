package metricstagdecorator

type DecoratorProperties struct{}

// WARNING : To avoid high cardinality in metrics, any new metrics tag decorator
// must reviewed and approved by the RPC or Observability teams before being added.
// It defines a contract for types that can provide tags for metrics.
type MetricsTagsDecorator interface {
	ProvideTags(request DecoratorProperties) map[string]string
}
