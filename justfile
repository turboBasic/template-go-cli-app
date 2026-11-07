default:
    @just --list

lint:
    prek run --all-files

fmt:
    prek run gofmt --all-files
    prek run goimports --all-files

test:
    go test ./...

test-race:
    go test -race ./...

coverage:
    go test -cover ./...

build:
    @mkdir -p dist
    go build -o dist/cli-app ./cmd/cli-app

# Wipes dist/, including the binary `just build` leaves there.
release-snapshot:
    mise exec -- goreleaser release --snapshot --clean
    ./scripts/check-release.sh cli-app

check-init:
    mise exec -- ./scripts/check-init.sh
