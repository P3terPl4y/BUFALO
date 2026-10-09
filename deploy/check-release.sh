#!/bin/sh
set -eu
# Deliberately requires an isolated database and never reads .env.production.
case "${APP_ENV:-}:${DB_DATABASE:-}" in
  testing:*_test) ;;
  *) echo 'Set APP_ENV=testing and an explicit isolated DB_DATABASE ending in _test' >&2; exit 2 ;;
esac
export GOMAXPROCS="${GOMAXPROCS:-2}" GOMEMLIMIT="${GOMEMLIMIT:-512MiB}"
export TMPDIR="${TMPDIR:-$PWD/storage/_build-tmp}"
mkdir -p "$TMPDIR" storage/release
if [ -n "$(gofmt -l app bootstrap config database routes tests main.go production_config.go production_config_test.go)" ]; then
 echo 'Go formatting required' >&2;exit 1
fi
go vet -p 1 ./...
go test -p 1 -count=1 ./...
go test -race -p 1 ./app/services ./tests/feature ./tests/redteam
CGO_ENABLED=0 go build -p 1 -trimpath -ldflags='-s -w' -o storage/release/bufalo .
if go list -deps ./... | grep -E '^golang.org/x/crypto/openpgp($|/)'; then
 echo 'Unsafe OpenPGP package imported' >&2;exit 1
fi
sha256sum storage/release/bufalo
