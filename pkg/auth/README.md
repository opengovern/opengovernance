# Service for Keibi Access Control

## Local development

Create your own `.dev.env` file with proper env variables.

* run `set -o allexport; source .dev.env; set +o allexport`
* run `docker compose up -d` to start local DB
* run `go run ./cmd/auth-service/main.go`
