.PHONY: build clean

#build:
#	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags '-w -extldflags -static' -o ./build/ ./cmd/...
build:
	export CGO_ENABLED=0
	export GOOS=linux
	export GOARCH=amd64
	ls cmd | xargs -I{} bash -c 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags "-w -extldflags -static" -o ./build/ ./cmd/{}'

clean:
	rm -r ./build
