#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

go build -o pg2 ./cmd/pg
echo "built $ROOT_DIR/pg2"
