#!/usr/bin/env sh

set -e

EXAMPLES=$(for F in $(grep --files-with-matches -R shared.RunAppInTest byke2d/examples/) ; do dirname $F ; done)

mkdir -p images

FAILED=0
for EXAMPLE in $EXAMPLES ; do
  echo "Run example $EXAMPLE"

  rm -f /tmp/image-*.png || true

  if ! ./run-example-test.sh $EXAMPLE > $EXAMPLE/log 2>&1 ; then
    echo "::error::Example failed: $EXAMPLE"
    FAILED=1

    cat $EXAMPLE/log
  else
    echo "::info::Example successful: $EXAMPLE"
  fi

  NAME=$(echo $EXAMPLE | tr '/' '_')

  IMAGE=$(find /tmp -maxdepth 1 -name 'image-*.png' | sort | head -n1)
  if [ -n "$IMAGE" ]; then
    cp "$IMAGE" "images/$NAME.png"
  fi

  echo
  echo
  echo
done

exit $FAILED
