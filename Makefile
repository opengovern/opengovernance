.PHONY: build clean compliance

#build:
#	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags '-w -extldflags -static' -o ./build/ ./cmd/...
build:
	export CGO_ENABLED=0
	export GOOS=linux
	export GOARCH=amd64
	./scripts/list_services
	ls cmd | xargs -I{} bash -c 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags "-w -extldflags -static" -o ./build/ ./cmd/{}'

compliance:
	export CGO_ENABLED=0
	export GOOS=linux
	export GOARCH=amd64
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags "-w -extldflags -static" -o ./build/ ./cmd/compliance-service
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags "-w -extldflags -static" -o ./build/ ./cmd/compliance-report-worker

clean:
	rm -r ./build
