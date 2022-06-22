# Service for Keibi Access Control

## Local development

Create your own `.dev.env` file with proper env variables.

* run `set -o allexport; source .dev.env; set +o allexport`
* run `docker compose up -d` to start local DB
* run psql -v ON_ERROR_STOP=1 --username=keibi --dbname=keibi_auth 'CREATE EXTENSION IF NOT EXISTS "uuid-ossp"'; 
* run `go run ./cmd/auth-service/main.go`
