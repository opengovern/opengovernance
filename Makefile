.PHONY: build build-all docker clean compliance

build-all:
	export GOOS=linux
	export GOARCH=amd64
	ls cmd | xargs -I{} bash -c "CC=/usr/bin/musl-gcc GOPRIVATE=\"github.com/kaytu-io\" GOOS=linux GOARCH=amd64 go build -tags musl -v -ldflags \"-linkmode external -extldflags '-static' -s -w\" -tags musl -o ./build/ ./cmd/{}"
	for f in $(ls ./cmd); do echo "$f=true" >> "$GITHUB_OUTPUT"; done

build:
	export GOOS=linux
	export GOARCH=amd64
	./scripts/list_services > ./services
	cat ./services
	cat ./services | grep -v "steampipe" | grep -v "redoc" | xargs -P 4 -I{} bash -c "CC=/usr/bin/musl-gcc GOPRIVATE=\"github.com/kaytu-io\" GOOS=linux GOARCH=amd64 go build -v -ldflags \"-linkmode external -extldflags '-static' -s -w\" -tags musl -o ./build/ ./cmd/{}"
	for f in $(cat ./services | grep -v "steampipe" | grep -v "redoc"); do echo "$f=true" >> "$GITHUB_OUTPUT"; done
clean:
	rm -r ./build
