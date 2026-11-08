#!/usr/bin/env bash
#
# Prove that scripts/init.sh renames the release configuration and that the result still builds.
# Copies the tracked tree to a scratch directory, initializes it as "demo-app" and asserts there.
#
# Usage:
#   ./scripts/check-init.sh
#
# Example:
#   ./scripts/check-init.sh
#

set -euo pipefail

readonly CONFIG_ID="demo-app"

main() {
  local repo_root
  repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
  SCRATCH="$(mktemp -d)"
  trap 'rm -rf "$SCRATCH"' EXIT

  copy_tracked_tree "$repo_root"
  run_init
  check_release_config
  check_example_config
  check_build
  echo "init check OK"
}

fail() {
  echo "ERROR: $*" >&2
  exit 1
}

copy_tracked_tree() {
  local repo_root="$1"
  git -C "$repo_root" ls-files -z | tar -C "$repo_root" --null -T - -cf - | tar -C "$SCRATCH" -xf -
}

run_init() {
  (cd "$SCRATCH" && ./scripts/init.sh "Demo App" "github.com/example/$CONFIG_ID" "$CONFIG_ID" >/dev/null) ||
    fail "init.sh failed in the scratch copy"
}

check_release_config() {
  local leftovers output
  cd "$SCRATCH"
  leftovers="$(grep -rn 'cli-app' .goreleaser.yaml .github/workflows || true)"
  [[ -z "$leftovers" ]] || fail "cli-app remains after init:"$'\n'"$leftovers"
  grep -Fqx "project_name: $CONFIG_ID" .goreleaser.yaml ||
    fail ".goreleaser.yaml lacks 'project_name: $CONFIG_ID'"
  grep -Eq "^[[:space:]]+main: \./cmd/$CONFIG_ID\$" .goreleaser.yaml ||
    fail ".goreleaser.yaml lacks 'main: ./cmd/$CONFIG_ID'"
  [[ -d "cmd/$CONFIG_ID" ]] || fail "cmd/$CONFIG_ID does not exist"
  # goreleaser check needs a git repository with a remote to resolve the release target
  git init -q
  git remote add origin "https://github.com/example/$CONFIG_ID.git"
  output="$(goreleaser check .goreleaser.yaml 2>&1)" ||
    fail "goreleaser check rejects the initialized .goreleaser.yaml:"$'\n'"$output"
}

check_example_config() {
  local file=".$CONFIG_ID.example.toml"
  [[ -f "$SCRATCH/$file" && ! -e "$SCRATCH/.cli-app.example.toml" ]] || fail "$file was not renamed from .cli-app.example.toml"
  ! grep -Eq 'cli-app|CLI_APP_' "$SCRATCH/$file" || fail "$file still names cli-app"
  grep -Fq 'DEMO_APP_LOG_LEVEL' "$SCRATCH/$file" || fail "$file lacks the DEMO_APP_ environment prefix"
}

check_build() {
  (cd "$SCRATCH" && go build -o /dev/null "./cmd/$CONFIG_ID") ||
    fail "go build ./cmd/$CONFIG_ID fails after init"
}

main "$@"
