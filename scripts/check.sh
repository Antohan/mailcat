#!/bin/sh
# Quality gate: gofmt, go vet, go build, go test.
# Prints only what failed; "OK" on success. One command, minimal output.
set -u
cd "$(dirname "$0")/.." || exit 2

unformatted=$(gofmt -l . 2>&1)
if [ -n "$unformatted" ]; then
    printf 'FAIL gofmt (unformatted files):\n%s\n' "$unformatted"
    exit 1
fi

if ! out=$(go vet ./... 2>&1); then
    printf 'FAIL go vet:\n%s\n' "$out" | tail -n 20
    exit 1
fi

if ! out=$(go build ./... 2>&1); then
    printf 'FAIL go build:\n%s\n' "$out" | tail -n 20
    exit 1
fi

if ! out=$(go test ./... 2>&1); then
    printf 'FAIL go test:\n%s\n' "$out" | grep -E '^(FAIL|---|\s*[a-zA-Z0-9_/.-]+\.go:)' | tail -n 30
    exit 1
fi

echo "OK: gofmt, vet, build, test"
