FROM golang:1.7.4

EXPOSE 8080-8087
RUN mkdir -p /go/src/go.uber.org/yarpc
WORKDIR /go/src/go.uber.org/yarpc
ADD glide.yaml /go/src/go.uber.org/yarpc/
ADD glide.lock /go/src/go.uber.org/yarpc/
RUN go get github.com/Masterminds/glide
RUN glide install
ADD . /go/src/go.uber.org/yarpc/
RUN go install go.uber.org/yarpc/internal/crossdock
CMD ["/go/bin/crossdock"]
