.PHONY: build clean

build:
	GOCACHE=$(GOCACHE)/linux/ CGO_ENABLED=0 GOOS=linux go build -v -ldflags '-w -extldflags -static' -o ./build/ ./cmd/...

docker-build:
	docker build -f  docker/DescribeSchedulerDockerfile . -t registry.digitalocean.com/keibi/describe-scheduler:0.0.1
	docker build -f  docker/DescribeWorkerDockerfile . -t registry.digitalocean.com/keibi/describe-worker:0.0.1
	docker build -f  docker/OnboardServiceDockerfile . -t registry.digitalocean.com/keibi/onboard-service:0.0.1
	docker build -f  docker/ComplianceReportWorkerDockerfile . -t registry.digitalocean.com/keibi/compliance-report-worker:0.0.1

docker-push:
	docker push registry.digitalocean.com/keibi/describe-scheduler:0.0.1
	docker push registry.digitalocean.com/keibi/describe-worker:0.0.1
	docker push registry.digitalocean.com/keibi/onboard-service:0.0.1
	docker push registry.digitalocean.com/keibi/compliance-report-worker:0.0.1

podman-build:
	podman build -f  docker/DescribeSchedulerDockerfile . -t registry.digitalocean.com/keibi/describe-scheduler:0.0.1
	podman build -f  docker/DescribeWorkerDockerfile . -t registry.digitalocean.com/keibi/describe-worker:0.0.1
	podman build -f  docker/OnboardServiceDockerfile . -t registry.digitalocean.com/keibi/onboard-service:0.0.1
	podman build -f  docker/ComplianceReportWorkerDockerfile . -t registry.digitalocean.com/keibi/compliance-report-worker:0.0.1

podman-push:
	podman push registry.digitalocean.com/keibi/describe-scheduler:0.0.1
	podman push registry.digitalocean.com/keibi/describe-worker:0.0.1
	podman push registry.digitalocean.com/keibi/onboard-service:0.0.1
	podman push registry.digitalocean.com/keibi/compliance-report-worker:0.0.1

clean:
	rm -r ./build