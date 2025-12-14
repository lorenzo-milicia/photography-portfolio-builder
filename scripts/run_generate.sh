#!/usr/bin/env bash
set -euo pipefail

# Run the static site generation using the built binary. This script is intended
# to be executed by the air runner (it will run the generate command once and
# then block to keep the process alive until air restarts it on file changes).

WORKDIR="$(pwd)"
BIN_PATH="$WORKDIR/bin/builder"

if [ ! -x "$BIN_PATH" ]; then
  echo "Builder binary not found at $BIN_PATH" >&2
  exit 1
fi

echo "Running site generation..."
"$BIN_PATH" generate -base-url ""
echo "Generation finished. Waiting for changes..."

# Block indefinitely so air keeps the process alive until a file change triggers
# a rebuild+restart. Use tail -f /dev/null which is portable enough.
exec tail -f /dev/null
