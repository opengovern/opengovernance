#!/bin/bash

set -x

if [[ -z $1 ]]; then 
  git status --porcelain | xargs -I{} bash -c 'echo "{}" | tail -c +1' | cut -d ' ' -f 2 | grep -E '.*\.go$' | xargs -I{} bash -c "gofmt -s -w {} || echo"
  git status --porcelain | xargs -I{} bash -c 'echo "{}" | tail -c +1' | cut -d ' ' -f 2 | grep -E '.*\.go$' | xargs -I{} bash -c "goimports -w {} || echo"
else 
  gofmt -s -w $1
  goimports -w $1
fi
