#!/bin/sh
# Quality gate: gofmt, golangci-lint, go vet, go build, go test.
# Prints only what failed; "OK" on success. One command, minimal output.
set -u
cd "$(dirname "$0")/.." || exit 2

# Lookup order: PATH, then $GOBIN, then the first $(go env GOPATH) entry.
find_tool() {
    if command -v "$1" >/dev/null 2>&1; then
        command -v "$1"
        return 0
    fi
    gobin=${GOBIN:-$(go env GOBIN 2>/dev/null)}
    if [ -n "$gobin" ] && [ -x "$gobin/$1" ]; then
        printf '%s\n' "$gobin/$1"
        return 0
    fi
    gopath=$(go env GOPATH 2>/dev/null | cut -d: -f1)
    if [ -n "$gopath" ] && [ -x "$gopath/bin/$1" ]; then
        printf '%s\n' "$gopath/bin/$1"
        return 0
    fi
    return 1
}

if ! out=$(gofmt -l . 2>&1); then
    printf 'FAIL gofmt:\n%s\n' "$out" | tail -n 20
    exit 1
fi
if [ -n "$out" ]; then
    printf 'FAIL gofmt (unformatted files):\n%s\n' "$out"
    exit 1
fi

lint=$(find_tool golangci-lint) || {
    printf 'FAIL lint: golangci-lint not found.\nInstall: go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest\n'
    exit 1
}
if ! out=$("$lint" run ./... 2>&1); then
    printf 'FAIL lint:\n%s\n' "$out" | tail -n 30
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
    printf 'FAIL go test:\n'
    printf '%s\n' "$out" | grep -E 'panic:|fatal error:' | head -n 3
    printf '%s\n' "$out" | grep -E '^(FAIL|---|[[:space:]]*[a-zA-Z0-9_/.-]+\.go:)' | tail -n 30
    exit 1
fi

echo "OK: gofmt, lint, vet, build, test"
