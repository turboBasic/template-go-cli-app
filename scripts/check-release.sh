#!/usr/bin/env bash
#
# Assert that a GoReleaser snapshot in dist/ holds the four release archives, their four SPDX
# SBOMs, a checksums.txt covering all eight, a Homebrew cask naming the four archives and a host
# binary that carries its injected version.
#
# Usage:
#   ./scripts/check-release.sh <binary-name>
#
# Example:
#   ./scripts/check-release.sh cli-app
#

set -euo pipefail

main() {
  local name="${1:-}"
  local repo_root
  repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

  [[ -n "$name" ]] || fail "usage: check-release.sh <binary-name>"
  cd "$repo_root/dist" || fail "dist/ is missing; run goreleaser first"
  SCRATCH="$(mktemp -d)"
  trap 'rm -rf "$SCRATCH"' EXIT

  check_archives "$name"
  check_sboms "$name"
  check_checksums
  check_cask "$name"
  check_host_version "$name"
  echo "release snapshot OK"
}

fail() {
  echo "ERROR: $*" >&2
  exit 1
}

check_archives() {
  local name="$1" os arch
  for os in linux darwin; do
    for arch in amd64 arm64; do
      compgen -G "${name}_*_${os}_${arch}.tar.gz" >/dev/null ||
        fail "missing archive ${name}_*_${os}_${arch}.tar.gz"
    done
  done
}

check_sboms() {
  local name="$1" archive
  for archive in "${name}"_*.tar.gz; do
    [[ -f "$archive.spdx.json" ]] || fail "missing SBOM $archive.spdx.json"
    grep -q '"spdxVersion"' "$archive.spdx.json" || fail "$archive.spdx.json has no spdxVersion"
  done
}

check_checksums() {
  local lines
  [[ -f checksums.txt ]] || fail "missing checksums.txt"
  lines="$(wc -l <checksums.txt)"
  ((lines == 8)) || fail "checksums.txt has $lines lines, want 8 (four archives, four SBOMs)"
  shasum -a 256 -c checksums.txt >/dev/null || fail "checksums.txt does not match the archives"
}

check_cask() {
  local cask="homebrew/Casks/$1.rb" urls
  [[ -f "$cask" ]] || fail "missing cask $cask"
  urls="$(grep -c '\.tar\.gz"' "$cask" || true)"
  ((urls == 4)) || fail "$cask names $urls .tar.gz URLs, want 4"
}

check_host_version() {
  local name="$1" os arch archive output
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  arch="$(uname -m)"
  case "$arch" in
    x86_64) arch=amd64 ;;
    aarch64) arch=arm64 ;;
  esac

  archive="$(compgen -G "${name}_*_${os}_${arch}.tar.gz" | head -n 1)" ||
    fail "no archive for host ${os}_${arch}"
  tar -xzf "$archive" -C "$SCRATCH" "$name"
  output="$("$SCRATCH/$name" --version)"
  [[ "$output" != *"version dev"* ]] ||
    fail "host binary reports 'version dev'; ldflags did not set main.version: $output"
}

main "$@"
