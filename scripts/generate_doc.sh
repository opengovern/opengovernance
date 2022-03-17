#!/bin/bash

cd "$(dirname "$0")/.."
swag init -g cmd/swagger-ui/main.go --dir "$(find {pkg,cmd} -type d | paste -d',' -s -)" --output pkg/docs