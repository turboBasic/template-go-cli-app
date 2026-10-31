#!/usr/bin/env bash
# Initialize a generated project with its name, module path, and config identifier.
#
# Usage:
#   ./scripts/init.sh [PROJECT_NAME [MODULE_PATH [CONFIG_ID]]]
#
# Example:
#   ./scripts/init.sh "Example App" github.com/example/example-app example-app

set -euo pipefail
IFS=$'\n\t'

main() {
  local repo_root temporary_dir
  local project_name module_path config_id
  local -a staged_paths target_paths
  repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

  parse_inputs "$@"
  validate_inputs
  validate_markers "$repo_root"
  temporary_dir="$(mktemp -d "$repo_root/.init-tmp.XXXXXX")"
  trap "rm -rf -- $(printf '%q' "$temporary_dir")" EXIT
  stage_changes "$repo_root" "$temporary_dir"
  stage_go_sources "$repo_root" "$temporary_dir"
  install_changes
  rename_command_directory "$repo_root"
  printf 'Initialized %s (%s) with config identifier %s.\n' "$project_name" "$module_path" "$config_id"
}

parse_inputs() {
  if (( $# > 3 )); then
    printf 'Usage: %s [PROJECT_NAME [MODULE_PATH [CONFIG_ID]]]\n' "$0" >&2
    exit 2
  fi

  project_name="${1:-}"
  module_path="${2:-}"
  config_id="${3:-}"
  prompt_for_value 'Project name' project_name
  prompt_for_value 'Go module path' module_path
  prompt_for_value 'Config identifier' config_id
}

validate_inputs() {
  local project_name_pattern='^[[:alnum:]][[:alnum:]. _-]{0,63}$'
  local module_path_pattern='^[[:alnum:]._-]+(/[[:alnum:]._-]+)+$'
  local config_id_pattern='^[a-z0-9]+(-[a-z0-9]+)*$'
  local segment
  local -a module_segments

  [[ "$project_name" =~ $project_name_pattern ]] || {
    printf 'Project name must start with a letter or number and use at most 64 letters, numbers, spaces, dots, underscores, or hyphens.\n' >&2
    exit 2
  }
  [[ "$module_path" =~ $module_path_pattern ]] || {
    printf 'Go module path must use slash-separated path segments with no spaces.\n' >&2
    exit 2
  }
  IFS='/' read -r -a module_segments <<< "$module_path"
  for segment in "${module_segments[@]}"; do
    [[ "$segment" != '.' && "$segment" != '..' ]] || {
      printf 'Go module path cannot contain dot or dot-dot path segments.\n' >&2
      exit 2
    }
  done
  [[ "$config_id" =~ $config_id_pattern ]] || {
    printf 'Config identifier must be lowercase kebab-case.\n' >&2
    exit 2
  }
}

validate_markers() {
  local repo_root="$1"
  local path

  require_marker_or_value "$repo_root/go.mod" 'github.com/turboBasic/template-go-cli-app' "$module_path"
  require_readme_title "$repo_root/README.md" "$project_name"
  for path in "$repo_root/README.md" "$repo_root/SECURITY.md" "$repo_root/justfile" "$repo_root/docs/architecture-specification.md"; do
    require_marker_or_value "$path" 'cli-app' "$config_id"
  done

  if [[ -e "$repo_root/cmd/cli-app" && -e "$repo_root/cmd/$config_id" && "$config_id" != 'cli-app' ]]; then
    printf 'Both cmd/cli-app and cmd/%s exist; refusing to overwrite either directory.\n' "$config_id" >&2
    exit 1
  fi
  if [[ ! -d "$repo_root/cmd/cli-app" && ! -d "$repo_root/cmd/$config_id" ]]; then
    printf 'Neither cmd/cli-app nor cmd/%s exists.\n' "$config_id" >&2
    exit 1
  fi
}

stage_changes() {
  local repo_root="$1"
  local temporary_dir="$2"
  local index=0
  local path

  for path in "$repo_root/go.mod" "$repo_root/README.md" "$repo_root/SECURITY.md" "$repo_root/justfile" "$repo_root/docs/architecture-specification.md"; do
    staged_paths+=("$temporary_dir/$index")
    target_paths+=("$path")
    cp -p "$path" "$temporary_dir/$index"
    if [[ "$path" == "$repo_root/go.mod" ]]; then
      stage_module_path_replacement "$path" "$temporary_dir/$index"
    elif [[ "$path" == "$repo_root/README.md" ]]; then
      sed -e 's|https://github.com/turboBasic/template-go-cli-app|@TEMPLATE_REPO_URL@|g' \
        -e "1s|^# Go CLI App Template.*$|# $project_name|" \
        -e "s|cli-app|$config_id|g" \
        -e 's|@TEMPLATE_REPO_URL@|https://github.com/turboBasic/template-go-cli-app|g' \
        "$path" > "$temporary_dir/$index"
    elif [[ "$path" == "$repo_root/SECURITY.md" ]]; then
      sed -e 's|https://github.com/turboBasic/template-go-cli-app|@TEMPLATE_REPO_URL@|g' \
        -e "s|cli-app|$config_id|g" \
        -e 's|@TEMPLATE_REPO_URL@|https://github.com/turboBasic/template-go-cli-app|g' \
        "$path" > "$temporary_dir/$index"
    else
      sed "s|cli-app|$config_id|g" "$path" > "$temporary_dir/$index"
    fi
    ((index += 1))
  done
}

stage_go_sources() {
  local repo_root="$1"
  local temporary_dir="$2"
  local path index

  while IFS= read -r -d '' path; do
    if grep -Fq -- 'github.com/turboBasic/template-go-cli-app' "$path"; then
      index="${#target_paths[@]}"
      staged_paths+=("$temporary_dir/$index")
      target_paths+=("$path")
      stage_module_path_replacement "$path" "$temporary_dir/$index"
    fi
  done < <(find "$repo_root" -type d \( -name .git -o -name vendor -o -name tmp -o -name dist \) -prune -o -type f -name '*.go' -print0)
}

stage_module_path_replacement() {
  local source="$1"
  local destination="$2"

  sed "s|github\.com/turboBasic/template-go-cli-app|$module_path|g" "$source" > "$destination"
}

install_changes() {
  local index
  for index in "${!target_paths[@]}"; do
    mv "${staged_paths[$index]}" "${target_paths[$index]}"
  done
}

rename_command_directory() {
  local repo_root="$1"
  if [[ -d "$repo_root/cmd/cli-app" && "$config_id" != 'cli-app' ]]; then
    mv "$repo_root/cmd/cli-app" "$repo_root/cmd/$config_id"
  fi
}

prompt_for_value() {
  local prompt="$1"
  local variable_name="$2"
  local value="${!2}"

  if [[ -z "$value" ]]; then
    [[ -t 0 ]] || {
      printf '%s is required when input is not interactive.\n' "$prompt" >&2
      exit 2
    }
    read -r -p "$prompt: " value || {
      printf 'Could not read %s.\n' "$prompt" >&2
      exit 2
    }
    printf -v "$variable_name" '%s' "$value"
  fi
}

require_marker_or_value() {
  local path="$1"
  local marker="$2"
  local value="$3"

  if [[ ! -f "$path" ]]; then
    printf 'Required file not found: %s\n' "$path" >&2
    exit 1
  fi
  if ! grep -Fq -- "$marker" "$path" && ! grep -Fq -- "$value" "$path"; then
    printf 'Neither initialization marker nor supplied value found in %s.\n' "$path" >&2
    exit 1
  fi
}

require_readme_title() {
  local path="$1"
  local project_name="$2"
  local title

  title="$(sed -n '1p' "$path")"
  if [[ "$title" != '# Go CLI App Template'* && "$title" != "# $project_name" ]]; then
    printf 'README title is not the template title or supplied project name.\n' >&2
    exit 1
  fi
}

main "$@"
