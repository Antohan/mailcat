#!/bin/sh
# Release gate: govulncheck and gosec are mandatory — a missing tool is a FAIL.
# Prints only what failed; "OK" on success.
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

gv=$(find_tool govulncheck) || {
    printf 'FAIL govulncheck: not found.\nInstall: go install golang.org/x/vuln/cmd/govulncheck@latest\n'
    exit 1
}
if ! out=$("$gv" ./... 2>&1); then
    printf 'FAIL govulncheck:\n%s\n' "$out" | tail -n 30
    exit 1
fi

gs=$(find_tool gosec) || {
    printf 'FAIL gosec: not found.\nInstall: go install github.com/securego/gosec/v2/cmd/gosec@latest\n'
    exit 1
}
if ! out=$("$gs" ./... 2>&1); then
    printf 'FAIL gosec:\n%s\n' "$out" | tail -n 30
    exit 1
fi

echo "OK: govulncheck, gosec"
