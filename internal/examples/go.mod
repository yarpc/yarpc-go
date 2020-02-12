module go.uber.org/yarpc/internal/examples

go 1.13

require (
	github.com/gogo/protobuf v1.3.1
	github.com/golang/mock v1.4.0
	github.com/stretchr/testify v1.4.0
	go.uber.org/fx v1.10.0
	go.uber.org/multierr v1.4.0
	go.uber.org/thriftrw v1.22.0
	go.uber.org/yarpc v1.42.1
	go.uber.org/zap v1.13.0
	google.golang.org/grpc v1.27.1
)

replace go.uber.org/yarpc => ../..
