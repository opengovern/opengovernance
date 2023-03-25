.PHONY: build build-all docker clean compliance

#build:
#	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags '-w -extldflags -static' -o ./build/ ./cmd/...
build-all:
	export CGO_ENABLED=0
	export GOOS=linux
	export GOARCH=amd64
	ls cmd | xargs -I{} bash -c 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags "-w -extldflags -static" -o ./build/ ./cmd/{}'

build:
	export CGO_ENABLED=0
	export GOOS=linux
	export GOARCH=amd64
	./scripts/list_services | xargs -I{} bash -c 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags "-w -extldflags -static" -o ./build/ ./cmd/{}'

docker:
	podman login -u "${DO_API_TOKEN}" -p "${DO_API_TOKEN}" "registry.digitalocean.com/keibi"
	SERVICES=$(./scripts/list_services)
	if [[ $SERVICES == *"auth-service"* ]]; then
		podman build -f  docker/AuthServiceDockerfile . -t registry.digitalocean.com/keibi/auth-service:$SVER_VERSION
		podman push registry.digitalocean.com/keibi/auth-service:$SVER_VERSION
	fi
	if [[ $SERVICES == *"checkup-worker"* ]]; then
		podman build -f  docker/CheckupWorkerDockerfile . -t registry.digitalocean.com/keibi/checkup-worker:$SVER_VERSION
		podman push registry.digitalocean.com/keibi/checkup-worker:$SVER_VERSION
	fi
	if [[ $SERVICES == *"compliance-report-worker"* ]]; then
		podman build -f  docker/ComplianceReportWorkerDockerfile . -t registry.digitalocean.com/keibi/compliance-report-worker:$SVER_VERSION
		podman push registry.digitalocean.com/keibi/compliance-report-worker:$SVER_VERSION
	fi
	if [[ $SERVICES == *"compliance-service"* ]]; then
		podman build -f  docker/ComplianceServiceDockerfile . -t registry.digitalocean.com/keibi/compliance-service:$SVER_VERSION
		podman push registry.digitalocean.com/keibi/compliance-service:$SVER_VERSION
	fi
	if [[ $SERVICES == *"describe-cleanup-worker"* ]]; then
		podman build -f  docker/DescribeCleanupWorkerDockerfile . -t registry.digitalocean.com/keibi/describe-cleanup-worker:$SVER_VERSION
		podman push registry.digitalocean.com/keibi/describe-cleanup-worker:$SVER_VERSION
	fi
	if [[ $SERVICES == *"describe-connection-worker"* ]]; then
		podman build -f  docker/DescribeConnectionWorkerDockerfile . -t registry.digitalocean.com/keibi/describe-connection-worker:$SVER_VERSION
		podman push registry.digitalocean.com/keibi/describe-connection-worker:$SVER_VERSION
	fi
	if [[ $SERVICES == *"describe-scheduler"* ]]; then
		podman build -f  docker/DescribeSchedulerDockerfile . -t registry.digitalocean.com/keibi/describe-scheduler:$SVER_VERSION
		podman push registry.digitalocean.com/keibi/describe-scheduler:$SVER_VERSION
	fi
	if [[ $SERVICES == *"describe-worker"* ]]; then
		podman build -f  docker/DescribeWorkerDockerfile . -t registry.digitalocean.com/keibi/describe-worker:$SVER_VERSION
		podman push registry.digitalocean.com/keibi/describe-worker:$SVER_VERSION
	fi
	if [[ $SERVICES == *"insight-worker"* ]]; then
		podman build -f  docker/InsightWorkerDockerfile . -t registry.digitalocean.com/keibi/insight-worker:$SVER_VERSION
		podman push registry.digitalocean.com/keibi/insight-worker:$SVER_VERSION
	fi
	if [[ $SERVICES == *"inventory-service"* ]]; then
		podman build -f  docker/InventoryServiceDockerfile . -t registry.digitalocean.com/keibi/inventory-service:$SVER_VERSION
		podman push registry.digitalocean.com/keibi/inventory-service:$SVER_VERSION
	fi
	if [[ $SERVICES == *"metadata-service"* ]]; then
		podman build -f  docker/MetadataServiceDockerfile . -t registry.digitalocean.com/keibi/metadata-service:$SVER_VERSION
		podman push registry.digitalocean.com/keibi/metadata-service:$SVER_VERSION
	fi
	if [[ $SERVICES == *"migrator-worker"* ]]; then
		podman build -f  docker/MigratorDockerfile . -t registry.digitalocean.com/keibi/migrator:$SVER_VERSION
		podman push registry.digitalocean.com/keibi/migrator:$SVER_VERSION
	fi
	if [[ $SERVICES == *"onboard-service"* ]]; then
		podman build -f  docker/OnboardServiceDockerfile . -t registry.digitalocean.com/keibi/onboard-service:$SVER_VERSION
		podman push registry.digitalocean.com/keibi/onboard-service:$SVER_VERSION
	fi
	if [[ $SERVICES == *"summarizer-worker"* ]]; then
		podman build -f  docker/SummarizerWorkerDockerfile . -t registry.digitalocean.com/keibi/summarizer-worker:$SVER_VERSION
		podman push registry.digitalocean.com/keibi/summarizer-worker:$SVER_VERSION
	fi
	if [[ $SERVICES == *"swagger-ui"* ]]; then
		podman build -f  docker/SwaggerUIDockerfile . -t registry.digitalocean.com/keibi/swagger-ui:$SVER_VERSION
		podman push registry.digitalocean.com/keibi/swagger-ui:$SVER_VERSION
	fi
	if [[ $SERVICES == *"workspace-service"* ]]; then
		podman build -f  docker/WorkspaceServiceDockerfile . -t registry.digitalocean.com/keibi/workspace-service:$SVER_VERSION
		podman push registry.digitalocean.com/keibi/workspace-service:$SVER_VERSION
	fi

docker-all:
	podman login -u "${DO_API_TOKEN}" -p "${DO_API_TOKEN}" "registry.digitalocean.com/keibi"
	podman build -f  docker/SteampipeServiceDockerfile . -t registry.digitalocean.com/keibi/steampipe-service:$SVER_VERSION
	podman build -f  docker/AuthServiceDockerfile . -t registry.digitalocean.com/keibi/auth-service:$SVER_VERSION
	podman build -f  docker/CheckupWorkerDockerfile . -t registry.digitalocean.com/keibi/checkup-worker:$SVER_VERSION
	podman build -f  docker/ComplianceReportWorkerDockerfile . -t registry.digitalocean.com/keibi/compliance-report-worker:$SVER_VERSION
	podman build -f  docker/ComplianceServiceDockerfile . -t registry.digitalocean.com/keibi/compliance-service:$SVER_VERSION
	podman build -f  docker/DescribeCleanupWorkerDockerfile . -t registry.digitalocean.com/keibi/describe-cleanup-worker:$SVER_VERSION
	podman build -f  docker/DescribeConnectionWorkerDockerfile . -t registry.digitalocean.com/keibi/describe-connection-worker:$SVER_VERSION
	podman build -f  docker/DescribeSchedulerDockerfile . -t registry.digitalocean.com/keibi/describe-scheduler:$SVER_VERSION
	podman build -f  docker/DescribeWorkerDockerfile . -t registry.digitalocean.com/keibi/describe-worker:$SVER_VERSION
	podman build -f  docker/InsightWorkerDockerfile . -t registry.digitalocean.com/keibi/insight-worker:$SVER_VERSION
	podman build -f  docker/InventoryServiceDockerfile . -t registry.digitalocean.com/keibi/inventory-service:$SVER_VERSION
	podman build -f  docker/MetadataServiceDockerfile . -t registry.digitalocean.com/keibi/metadata-service:$SVER_VERSION
	podman build -f  docker/MigratorDockerfile . -t registry.digitalocean.com/keibi/migrator:$SVER_VERSION
	podman build -f  docker/OnboardServiceDockerfile . -t registry.digitalocean.com/keibi/onboard-service:$SVER_VERSION
	podman build -f  docker/SummarizerWorkerDockerfile . -t registry.digitalocean.com/keibi/summarizer-worker:$SVER_VERSION
	podman build -f  docker/SwaggerUIDockerfile . -t registry.digitalocean.com/keibi/swagger-ui:$SVER_VERSION
	podman build -f  docker/WorkspaceServiceDockerfile . -t registry.digitalocean.com/keibi/workspace-service:$SVER_VERSION
	podman push registry.digitalocean.com/keibi/steampipe-service:$SVER_VERSION
	podman push registry.digitalocean.com/keibi/auth-service:$SVER_VERSION
	podman push registry.digitalocean.com/keibi/checkup-worker:$SVER_VERSION
	podman push registry.digitalocean.com/keibi/compliance-report-worker:$SVER_VERSION
	podman push registry.digitalocean.com/keibi/compliance-service:$SVER_VERSION
	podman push registry.digitalocean.com/keibi/describe-cleanup-worker:$SVER_VERSION
	podman push registry.digitalocean.com/keibi/describe-connection-worker:$SVER_VERSION
	podman push registry.digitalocean.com/keibi/describe-scheduler:$SVER_VERSION
	podman push registry.digitalocean.com/keibi/describe-worker:$SVER_VERSION
	podman push registry.digitalocean.com/keibi/insight-worker:$SVER_VERSION
	podman push registry.digitalocean.com/keibi/inventory-service:$SVER_VERSION
	podman push registry.digitalocean.com/keibi/metadata-service:$SVER_VERSION
	podman push registry.digitalocean.com/keibi/migrator:$SVER_VERSION
	podman push registry.digitalocean.com/keibi/onboard-service:$SVER_VERSION
	podman push registry.digitalocean.com/keibi/summarizer-worker:$SVER_VERSION
	podman push registry.digitalocean.com/keibi/swagger-ui:$SVER_VERSION
	podman push registry.digitalocean.com/keibi/workspace-service:$SVER_VERSION

clean:
	rm -r ./build
