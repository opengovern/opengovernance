.PHONY: build clean

build:
	CGO_ENABLED=0 GOOS=linux go build -v -o ./build/ ./cmd/...

docker-build:
	docker build -f  docker/TaskPublisherDockerfile . -t registry.digitalocean.com/keibi/task-publisher:0.0.1
	docker build -f  docker/TaskWorkerDockerfile . -t registry.digitalocean.com/keibi/task-worker:0.0.1

docker-push:
	docker push registry.digitalocean.com/keibi/task-publisher:0.0.1
	docker push registry.digitalocean.com/keibi/task-worker:0.0.1

clean:
	rm -r ./build