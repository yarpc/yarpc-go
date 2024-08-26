module go.uber.org/yarpc

go 1.21

toolchain go1.22.2

require (
	github.com/dgryski/go-farm v0.0.0-20200201041132-a6ae2369ad13
	github.com/gogo/googleapis v1.4.1
	github.com/gogo/protobuf v1.3.2
	github.com/gogo/status v1.1.0
	github.com/golang/mock v1.7.0-rc.1
	github.com/golang/protobuf v1.5.4
	github.com/golang/snappy v0.0.4
	github.com/kisielk/errcheck v1.7.0
	github.com/mattn/go-shellwords v1.0.12
	github.com/opentracing/opentracing-go v1.2.0
	github.com/stretchr/testify v1.9.0
	github.com/uber-go/mapdecode v1.0.0
	github.com/uber-go/tally v3.5.8+incompatible
	github.com/uber/jaeger-client-go v2.30.0+incompatible
	github.com/uber/ringpop-go v0.8.5
	github.com/uber/tchannel-go v1.34.4
	go.uber.org/atomic v1.11.0
	go.uber.org/fx v1.22.0
	go.uber.org/goleak v1.3.0
	go.uber.org/multierr v1.11.0
	go.uber.org/net/metrics v1.4.0
	go.uber.org/thriftrw v1.32.0
	go.uber.org/tools v0.0.0-20190618225709-2cfd321de3ee
	go.uber.org/yarpc/internal/examples v0.0.0-20230831212929-ccef8c01afa8
	go.uber.org/zap v1.27.0
	golang.org/x/lint v0.0.0-20210508222113-6edffad5e616
	golang.org/x/net v0.27.0
	golang.org/x/tools v0.22.0
	google.golang.org/genproto v0.0.0-20221118155620-16455021b5e6
	google.golang.org/grpc v1.50.1
	google.golang.org/protobuf v1.34.1
	gopkg.in/yaml.v2 v2.4.0
	honnef.co/go/tools v0.4.3
)

require (
	github.com/BurntSushi/toml v1.2.1 // indirect
	github.com/anmitsu/go-shlex v0.0.0-20200514113438-38f4b401e2be // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cactus/go-statsd-client/statsd v0.0.0-20191106001114-12b4e2b38748 // indirect
	github.com/cespare/xxhash/v2 v2.1.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/fatih/structtag v1.2.0 // indirect
	github.com/jessevdk/go-flags v1.5.0 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_golang v1.4.1 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.9.1 // indirect
	github.com/prometheus/procfs v0.0.9 // indirect
	github.com/samuel/go-thrift v0.0.0-20191111193933-5165175b40af // indirect
	github.com/sirupsen/logrus v1.4.2 // indirect
	github.com/twmb/murmur3 v1.1.8 // indirect
	github.com/uber-common/bark v1.2.1 // indirect
	github.com/uber/jaeger-lib v2.4.1+incompatible // indirect
	go.uber.org/dig v1.17.1 // indirect
	golang.org/x/exp/typeparams v0.0.0-20221208152030-732eee02a75a // indirect
	golang.org/x/mod v0.18.0 // indirect
	golang.org/x/sync v0.7.0 // indirect
	golang.org/x/sys v0.22.0 // indirect
	golang.org/x/text v0.16.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// google.golang.org/genproto upgrade leads to grpc upgrade, which we're not ready for yet.
replace google.golang.org/grpc => google.golang.org/grpc v1.52.3

replace google.golang.org/genproto => google.golang.org/genproto v0.0.0-20221014173430-6e2ab493f96b
