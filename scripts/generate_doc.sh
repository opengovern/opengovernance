#!/usr/bin/env bash

# https://stackoverflow.com/questions/3822621/how-to-exit-if-a-command-failed
set -eu
set -o pipefail

cd "$(dirname "$0")/.." || exit

directories="$(find {pkg,cmd,services} -type d | paste -d',' -s -)"

swag fmt -g ../cmd/swagger-ui/main.go --dir "$directories"
swag init --parseDependency -g ../cmd/swagger-ui/main.go --dir "$directories" --output pkg/docs
sed -i '/opengovernance-admin/d' pkg/docs/swagger.yaml
sed -i '/AdminRole/d' pkg/docs/swagger.yaml
