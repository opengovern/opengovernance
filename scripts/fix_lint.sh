#!/bin/bash

set -x

git status --porcelain | xargs -I{} bash -c 'echo "{}" | tail -c +1' | cut -d ' ' -f 2 | grep -E '.*\.go$' | xargs -I{} bash -c "gofmt -s -w {} || echo"
git status --porcelain | xargs -I{} bash -c 'echo "{}" | tail -c +1' | cut -d ' ' -f 2 | grep -E '.*\.go$' | xargs -I{} bash -c "goimports -w {} || echo"
