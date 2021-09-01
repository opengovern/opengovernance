.PHONY: build clean

build:
	go build -v -o ./build/ ./cmd/...

clean:
	rm -r ./build