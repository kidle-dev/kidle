#!/usr/bin/env bash
#
# Script is meant to build multi-arch container images and publish them to docer.io/kidledev registry
#
# Script is:
# - figuring out the image tag (aka version) based on GITHUB_REF value
#
# Script is not:
# - directly executing `docker build`. This is done in Makefile
# - logging to registries
#

# exit immediately when a command fails
set -e
# only exit with zero if all commands of the pipeline exit successfully
set -o pipefail

CPU_ARCHS="amd64 arm64 arm"


# Build images
for arch in ${CPU_ARCHS}; do
	make WHAT=$WHAT docker GOARCH="$arch"
done

# Compose multi-arch images and push them to remote registry
export DOCKER_CLI_EXPERIMENTAL=enabled
# Create manifest to join all images under one virtual tag
docker manifest create -a "${IMAGE}:${TAG}" \
        "${IMAGE}:${TAG}-amd64" \
        "${IMAGE}:${TAG}-arm64" \
        "${IMAGE}:${TAG}-arm"

# Annotate to set which image is build for which CPU architecture
for arch in $CPU_ARCHS; do
  docker manifest annotate --arch "$arch" "${IMAGE}:${TAG}" "${IMAGE}:${TAG}-$arch"
done
docker manifest push "${IMAGE}:${TAG}"
