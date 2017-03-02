FROM golang:1.8.0

EXPOSE 8080-8087
WORKDIR /go/src/go.uber.org/yarpc
ADD glide.yaml glide.lock /go/src/go.uber.org/yarpc/
RUN go get github.com/Masterminds/glide && glide install
ADD . /go/src/go.uber.org/yarpc/
RUN go install go.uber.org/yarpc/internal/crossdock
CMD ["/go/bin/crossdock"]
