FROM golang
ENV GO15VENDOREXPERIMENT 1
ADD . /go/src/github.com/yarpc/yarpc-go
RUN go install github.com/yarpc/yarpc-go/crossdock
ENTRYPOINT /go/bin/crossdock
EXPOSE 8080
