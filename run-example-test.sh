#!/usr/bin/env sh

set -e

export GOEXPERIMENT=simd
export BYKE_RUN_OFFSCREEN_TEST=true
export WGPU_FORCE_FALLBACK_ADAPTER=1

rm -f /tmp/image-*.png || true

cd "$1"
exec go run --tags fakeClock .
