.PHONY: all hello kvserver kvclient

all: kvserver kvclient

gen:
	cd keyvalue && go generate

kvclient:
	cd keyvalue/client && go build -i

kvserver:
	cd keyvalue/server && go build -i
