#!/usr/bin/env sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$repo_root"

# Both packages intentionally use the same disposable PostgreSQL schema.
exec go test -p=1 ./tests ./tests/postgres -count=1
