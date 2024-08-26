module go.uber.org/yarpc/internal/examples

go 1.21

toolchain go1.22.2

require (
	github.com/gogo/protobuf v1.3.2
	github.com/golang/mock v1.7.0-rc.1
	github.com/stretchr/testify v1.9.0
	go.uber.org/fx v1.22.0
	go.uber.org/multierr v1.11.0
	go.uber.org/thriftrw v1.32.0
	go.uber.org/yarpc v1.42.1
	go.uber.org/zap v1.27.0
	google.golang.org/grpc v1.50.1
)

require (
	github.com/BurntSushi/toml v1.2.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/gogo/googleapis v1.4.1 // indirect
	github.com/gogo/status v1.1.0 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_golang v1.4.1 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.9.1 // indirect
	github.com/prometheus/procfs v0.0.9 // indirect
	github.com/twmb/murmur3 v1.1.8 // indirect
	github.com/uber-go/mapdecode v1.0.0 // indirect
	github.com/uber-go/tally v3.5.8+incompatible // indirect
	github.com/uber/tchannel-go v1.34.4 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/dig v1.17.1 // indirect
	go.uber.org/net/metrics v1.4.0 // indirect
	golang.org/x/exp/typeparams v0.0.0-20221208152030-732eee02a75a // indirect
	golang.org/x/lint v0.0.0-20210508222113-6edffad5e616 // indirect
	golang.org/x/mod v0.18.0 // indirect
	golang.org/x/net v0.27.0 // indirect
	golang.org/x/sync v0.7.0 // indirect
	golang.org/x/sys v0.22.0 // indirect
	golang.org/x/text v0.16.0 // indirect
	golang.org/x/tools v0.22.0 // indirect
	google.golang.org/genproto v0.0.0-20221118155620-16455021b5e6 // indirect
	google.golang.org/protobuf v1.34.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	honnef.co/go/tools v0.4.3 // indirect
)

replace go.uber.org/yarpc => ../..

// google.golang.org/genproto upgrade leads to grpc upgrade, which we're not ready for yet.
replace google.golang.org/grpc => google.golang.org/grpc v1.52.3

replace google.golang.org/genproto => google.golang.org/genproto v0.0.0-20221014173430-6e2ab493f96b
