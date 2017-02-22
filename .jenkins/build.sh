#!/bin/bash

export BUILD_DATE=$(date +%y%m%d%H%M)

for platform in linux windows darwin; do
  echo "Building and pushing for $platform"

  make build_${platform} push_${platform}
done
