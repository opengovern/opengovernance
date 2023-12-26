#!/usr/bin/env bash

# https://stackoverflow.com/questions/3822621/how-to-exit-if-a-command-failed
set -eu
set -o pipefail

cd "$(dirname "$0")/.." || exit

swag fmt -g ../cmd/swagger-ui/main.go --dir "$(find {pkg,cmd,services} -type d | paste -d',' -s -)"
swag init --parseDependency -g ../cmd/swagger-ui/main.go --dir "$(find {pkg,cmd} -type d | paste -d',' -s -)" --output pkg/docs
#context=`cat pkg/docs/tag-groups.yaml`
#echo "$context" | cat - pkg/docs/swagger.yaml > temp && mv temp pkg/docs/swagger.yaml
sed -i '/kaytu-admin/d' pkg/docs/swagger.yaml
sed -i '/KaytuAdminRole/d' pkg/docs/swagger.yaml
