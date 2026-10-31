default:
    @just --list

lint:
    prek run --all-files

fmt:
    prek run gofmt --all-files
    prek run goimports --all-files

test:
    go test ./...

build:
    go build -o dist/cli-app ./...
