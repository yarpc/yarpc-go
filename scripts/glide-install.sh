if [ "$(glide --version)" != "glide version 0.12.3" ]; then
	mkdir -p "$GOPATH/src/github.com/Masterminds/glide"
	cd "$GOPATH/src/github.com/Masterminds/glide" || exit -1
	git clone https://github.com/Masterminds/glide.git . || true
	git checkout v0.12.3
	go install
fi
glide --version
