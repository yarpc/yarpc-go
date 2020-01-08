module go.uber.org/yarpc/internal/examples

go 1.13

require (
	github.com/gogo/protobuf v1.3.1
	github.com/golang/mock v1.3.1
	github.com/stretchr/testify v1.4.0
	go.uber.org/fx v1.10.0
	go.uber.org/multierr v1.4.0
	go.uber.org/thriftrw v1.21.0
	go.uber.org/yarpc v1.42.1
	go.uber.org/zap v1.13.0
	google.golang.org/genproto v0.0.0-20191230161307-f3c370f40bfb // indirect
	google.golang.org/grpc v1.26.0
	gopkg.in/yaml.v2 v2.2.7 // indirect
)

replace go.uber.org/yarpc => ../..
