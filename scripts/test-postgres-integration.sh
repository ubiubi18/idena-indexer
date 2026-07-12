#!/usr/bin/env sh
set -eu

repo_root=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd -P)
cd "$repo_root"

tmp_root=${TMPDIR:-/tmp}
tmp_dir=$(mktemp -d "$tmp_root/idena-indexer-postgres-tests.XXXXXX")

cleanup() {
  rm -rf "$tmp_dir"
}
trap cleanup EXIT HUP INT TERM

run_isolated_suite() {
  binary=$1
  list_file=$2
  package_dir=$3

  (cd "$package_dir" && "$binary" -test.list '^Test') >"$list_file"
  while IFS= read -r test_name; do
    case "$test_name" in
      Test*[!A-Za-z0-9_]*)
        printf 'Invalid Go test name: %s\n' "$test_name" >&2
        return 1
        ;;
      Test*) ;;
      *)
        printf 'Unexpected test-list output: %s\n' "$test_name" >&2
        return 1
        ;;
    esac

    printf '=== ISOLATED RUN   %s\n' "$test_name"
    (
      cd "$package_dir"
      "$binary" \
        -test.run "^${test_name}$" \
        -test.count=1 \
        -test.timeout=2m
    )
  done <"$list_file"
}

# Legacy tests intentionally reuse one schema and leave background resources
# alive. Process isolation releases those resources without changing runtime
# retry or consensus behavior.
go test -c -o "$tmp_dir/indexer-tests" ./tests
go test -c -o "$tmp_dir/postgres-tests" ./tests/postgres
run_isolated_suite "$tmp_dir/indexer-tests" "$tmp_dir/indexer-tests.list" "$repo_root/tests"
run_isolated_suite "$tmp_dir/postgres-tests" "$tmp_dir/postgres-tests.list" "$repo_root/tests/postgres"
