#!/usr/bin/env bash

set -e

export GOEXPERIMENT=simd
export BYKE_RUN_OFFSCREEN_TEST=true
export WGPU_FORCE_FALLBACK_ADAPTER=1

for name in "$@" ; do
    rm -f /tmp/image-*.png || true

    pushd "$name"
    go run --tags fakeClock .
    popd
done
