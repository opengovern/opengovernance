.PHONY: install build clean

build:
	CC=/usr/bin/musl-gcc GOPRIVATE="github.com/opengovern" GOOS=linux GOARCH=amd64 go build -v -ldflags "-linkmode external -extldflags '-static' -s -w" -tags musl -o ./build/steampipe-plugin-opengovernance.plugin *.go

docker-build:
	docker build -f docker/Dockerfile . -t steampipe-plugin-opengovernance:0.0.1
	docker build -f docker/Dockerfile . -t steampipe-plugin-opengovernance:0.0.1-$CI_PIPELINE_IID

docker-push:
	docker push steampipe-plugin-opengovernance:0.0.1
	docker push steampipe-plugin-opengovernance:0.0.1-$CI_PIPELINE_IID

podman-build:
	podman build -f docker/Dockerfile . -t steampipe-plugin-opengovernance:0.0.1
	podman build -f docker/Dockerfile . -t steampipe-plugin-opengovernance:0.0.1-$CI_PIPELINE_IID

podman-push:
	podman push steampipe-plugin-opengovernance:0.0.1
	podman push steampipe-plugin-opengovernance:0.0.1-$CI_PIPELINE_IID

clean:
	rm -r ./build

install:
	go build -v -o ~/.steampipe/plugins/local/opengovernance/opengovernance.plugin *.go
