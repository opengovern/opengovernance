#!/bin/bash

cd "$(dirname "$0")/.."
swag fmt -g ../cmd/swagger-ui/main.go --dir "$(find {pkg,cmd} -type d | paste -d',' -s -)"
swag init --parseDependency -g ../cmd/swagger-ui/main.go --dir "$(find {pkg,cmd} -type d | paste -d',' -s -)" --output pkg/docs
