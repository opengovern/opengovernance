#!/bin/bash

cat do_repositories | xargs -P2 -I{} ./do_cleanup.sh --repository {}
