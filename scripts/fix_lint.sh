#!/bin/bash

set -e
set -x

git status --porcelain | xargs -I{} bash -c 'echo "{}" | tail -c +1' | cut -d ' ' -f 2 | grep -E '.*\.go$' | xargs -I{} gofmt -s -w {}
git status --porcelain | xargs -I{} bash -c 'echo "{}" | tail -c +1' | cut -d ' ' -f 2 | grep -E '.*\.go$' | xargs -I{} goimports -w {}
