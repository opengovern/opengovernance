.PHONY: build clean

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags '-w -extldflags -static' -o ./build/ ./cmd/...

clean:
	rm -r ./build
