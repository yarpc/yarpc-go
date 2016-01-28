FROM golang
ENV GO15VENDOREXPERIMENT 1
ADD . /go/src/github.com/yarpc/yarpc-go
RUN go install github.com/yarpc/yarpc-go/xlang
ENTRYPOINT /go/bin/xlang
EXPOSE 8080
