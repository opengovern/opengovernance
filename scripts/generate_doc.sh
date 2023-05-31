#!/bin/bash

cd "$(dirname "$0")/.."
swag fmt -g ../cmd/swagger-ui/main.go --dir "$(find {pkg,cmd} -type d | paste -d',' -s -)"
swag init --parseDependency -g ../cmd/swagger-ui/main.go --dir "$(find {pkg,cmd} -type d | paste -d',' -s -)" --output pkg/docs
context=`cat pkg/docs/tag-groups.yaml`
echo "$context" | cat - pkg/docs/swagger.yaml > temp && mv temp pkg/docs/swagger.yaml
sed -i '/keibi-admin/d' pkg/docs/swagger.yaml
sed -i '/KeibiAdminRole/d' pkg/docs/swagger.yaml
