#!/bin/bash

export WORKINGDIR=$(pwd)

target=${1:-build}
if [ "$target" == "build" ]; then
  arch=$(go env GOHOSTARCH)
  platform=$(go env GOHOSTOS)
  binPath="$(pwd)/dist/pct_${platform}_${arch}"
  # Set goreleaser to build for current platform only
  goreleaser build --snapshot --rm-dist --single-target
  git clone -b main --depth 1 --single-branch https://github.com/puppetlabs/baker-round "$binPath/templates"
elif [ "$target" == "package" ]; then
  git clone -b main --depth 1 --single-branch https://github.com/puppetlabs/baker-round "templates"
  goreleaser --skip-publish --snapshot --rm-dist
fi
