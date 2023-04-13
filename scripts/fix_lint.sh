#!/bin/bash
set -e
set -x

GOIMPORTS=$(which goimports || echo -n "")
if [[ -z GOIMPORTS ]]; then
  echo -n "Installing goimports ."
  go install golang.org/x/tools/cmd/goimports@latest
  echo " Done"
fi

if [[ -z $1 ]]; then 
  git status --porcelain | xargs -I{} bash -c 'echo "{}" | tail -c +1' | awk '{print $2}' | grep -E '.*\.go$' | xargs -I{} bash -c "gofmt -s -w {} || echo"
  git status --porcelain | xargs -I{} bash -c 'echo "{}" | tail -c +1' | awk '{print $2}' | grep -E '.*\.go$' | xargs -I{} bash -c "goimports -w {} || echo"
else 
  gofmt -s -w $1
  goimports -w $1
fi

echo "done"
