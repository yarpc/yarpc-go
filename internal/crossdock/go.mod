module go.uber.org/yarpc/internal/crossdock

go 1.21

toolchain go1.22.2

require (
	github.com/apache/thrift v0.13.0
	github.com/crossdock/crossdock-go v0.0.0-20160816171116-049aabb0122b
	github.com/gogo/protobuf v1.3.1
	github.com/golang/mock v1.6.0
	github.com/opentracing/opentracing-go v1.1.0
	github.com/stretchr/testify v1.7.0
	github.com/uber/jaeger-client-go v2.22.1+incompatible
	github.com/uber/tchannel-go v1.33.0
	go.uber.org/fx v1.10.0
	go.uber.org/multierr v1.4.0
	go.uber.org/thriftrw v1.31.0
	go.uber.org/yarpc v1.42.1
	go.uber.org/zap v1.13.0
	golang.org/x/net v0.23.0
	google.golang.org/grpc v1.40.1
)

require (
	github.com/BurntSushi/toml v1.2.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/gogo/googleapis v1.3.2 // indirect
	github.com/gogo/status v1.1.0 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_golang v1.4.1 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.9.1 // indirect
	github.com/prometheus/procfs v0.0.9 // indirect
	github.com/uber-go/mapdecode v1.0.0 // indirect
	github.com/uber-go/tally v3.3.15+incompatible // indirect
	github.com/uber/jaeger-lib v2.4.1+incompatible // indirect
	go.uber.org/atomic v1.6.0 // indirect
	go.uber.org/dig v1.8.0 // indirect
	go.uber.org/net/metrics v1.3.0 // indirect
	go.uber.org/tools v0.0.0-20190618225709-2cfd321de3ee // indirect
	golang.org/x/exp/typeparams v0.0.0-20221208152030-732eee02a75a // indirect
	golang.org/x/lint v0.0.0-20200130185559-910be7a94367 // indirect
	golang.org/x/mod v0.9.0 // indirect
	golang.org/x/sys v0.18.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	golang.org/x/tools v0.7.0 // indirect
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013 // indirect
	google.golang.org/protobuf v1.26.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	honnef.co/go/tools v0.4.3 // indirect
)

replace go.uber.org/yarpc => ../..

// Pin to v0.10.0; 0.11 added context arguments which breaks TChannel Go.
//
// We're pinning to hash because before 0.12, Apache Thrift did not include a
// 'v' prefix for their SemVer releases, which is incompatible with Go
// modules.
replace github.com/apache/thrift => github.com/apache/thrift v0.0.0-20161221203622-b2a4d4ae21c7
