.PHONY:

build: caller/main.linux service/main.linux proxy/main.linux

clean:
	rm caller/main.linux service/main.linux proxy/main.linux

caller/main.linux: caller/main.go
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o caller/main.linux caller/main.go

service/main.linux: service/main.go
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o service/main.linux service/main.go

proxy/main.linux: proxy/main.go
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o proxy/main.linux proxy/main.go

generate:
	go get -v -u go.uber.org/thriftrw
	(cd service; go generate gen.go)
	(cd caller; go generate gen.go)
