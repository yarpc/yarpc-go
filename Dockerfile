FROM golang
ADD . /go/src/github.com/yarpc/yarpc-go
RUN go install github.com/yarpc/yarpc-go
ENTRYPOINT /go/bin/yarpc-go
EXPOSE 8080
