FROM golang:1.7
ADD . /go/src/go.uber.org/yarpc
RUN go install go.uber.org/yarpc/crossdock
ENTRYPOINT /go/bin/crossdock
EXPOSE 8080-8087
