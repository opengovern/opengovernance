#!/bin/bash

SVER_VERSION=$(/app/sver tags -s "${SVER_DOCKER_SERVER}" -u "${SVER_DOCKER_USERNAME}" -p "${SVER_DOCKER_PASSWORD}" "${SVER_DOCKER_REGISTRY}");
SVER_VERSION=$(echo "$SVER_VERSION" | sed '$!s/$/\,/' | tr -d '\n')
echo "SVER_VERSION=$SVER_VERSION"
