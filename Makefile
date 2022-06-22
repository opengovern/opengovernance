.PHONY: build clean

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags '-w -extldflags -static' -o ./build/ ./cmd/...

docker-build:
	docker build -f  docker/DescribeSchedulerDockerfile . -t registry.digitalocean.com/keibi/describe-scheduler:0.0.1
	docker build -f  docker/DescribeWorkerDockerfile . -t registry.digitalocean.com/keibi/describe-worker:0.0.1
	docker build -f  docker/DescribeCleanupWorkerDockerfile . -t registry.digitalocean.com/keibi/describe-cleanup-worker:0.0.1
	docker build -f  docker/OnboardServiceDockerfile . -t registry.digitalocean.com/keibi/onboard-service:0.0.1
	docker build -f  docker/InventoryServiceDockerfile . -t registry.digitalocean.com/keibi/inventory-service:0.0.1
	docker build -f  docker/ComplianceReportWorkerDockerfile . -t registry.digitalocean.com/keibi/compliance-report-worker:0.0.1
	docker build -f  docker/AuthServiceDockerfile . -t registry.digitalocean.com/keibi/auth-service:0.0.1
	docker build -f  docker/WorkspaceDockerfile . -t registry.digitalocean.com/keibi/workspace-service:0.0.1
	docker build -f  docker/InsightWorkerDockerfile . -t registry.digitalocean.com/keibi/insight-worker:0.0.1

docker-push:
	docker push registry.digitalocean.com/keibi/describe-scheduler:0.0.1
	docker push registry.digitalocean.com/keibi/describe-worker:0.0.1
	docker push registry.digitalocean.com/keibi/describe-cleanup-worker:0.0.1
	docker push registry.digitalocean.com/keibi/onboard-service:0.0.1
	docker push registry.digitalocean.com/keibi/inventory-service:0.0.1
	docker push registry.digitalocean.com/keibi/compliance-report-worker:0.0.1
	docker push registry.digitalocean.com/keibi/auth-service:0.0.1
	docker push registry.digitalocean.com/keibi/workspace-service:0.0.1
	docker push registry.digitalocean.com/keibi/insight-worker:0.0.1

clean:
	rm -r ./build
