#!/bin/bash

SVER_VERSION=$(/app/sver -n patch);
SVER_VERSION=$(echo "$SVER_VERSION" | sed '$!s/$/\,/' | tr -d '\n')
echo "SVER_VERSION=$SVER_VERSION"
