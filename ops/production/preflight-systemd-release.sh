#!/usr/bin/env bash
set -euo pipefail

release_root="${RELEASE_ROOT:-/opt/sub2api/releases}"
current_link="${CURRENT_LINK:-/opt/sub2api/current}"
binary_relative="${BINARY_RELATIVE:-sub2api}"
service_name="${SERVICE_NAME:-sub2api.service}"
service_user="${SERVICE_USER:-sub2api}"
production_env_file="${PRODUCTION_ENV_FILE:-/etc/sub2api/sub2api.env}"
preflight_env_file="${PREFLIGHT_ENV_FILE:-/etc/sub2api/preflight.env}"
preflight_port="${PREFLIGHT_PORT:-18081}"
preflight_health_url="${PREFLIGHT_HEALTH_URL:-http://127.0.0.1:${preflight_port}/health}"
production_health_url="${PRODUCTION_HEALTH_URL:-http://127.0.0.1:8080/health}"
health_timeout="${HEALTH_TIMEOUT_SECONDS:-30}"

log() { printf '[release] %s\n' "$*"; }
die() { printf '[release] ERROR: %s\n' "$*" >&2; exit 1; }

[[ $# -eq 1 ]] || die "usage: $0 <candidate-release-directory>"
[[ "$health_timeout" =~ ^[1-9][0-9]*$ ]] || die "HEALTH_TIMEOUT_SECONDS must be a positive integer"
[[ "$preflight_port" =~ ^[1-9][0-9]{0,4}$ ]] || die "PREFLIGHT_PORT must be a valid port"
preflight_port_number=$((10#$preflight_port))
(( preflight_port_number <= 65535 )) || die "PREFLIGHT_PORT must be at most 65535"

for command_name in realpath runuser curl systemctl ln mv readlink chown; do
  command -v "$command_name" >/dev/null 2>&1 || die "required command not found: $command_name"
done

release_root="$(realpath -m "$release_root")"
candidate_dir="$(realpath -m "$1")"
# Resolve only the parent: resolving the link itself would follow it to the old
# release and make the later atomic replacement target the wrong path.
current_link_parent="$(realpath -m "$(dirname "$current_link")")"
current_link="$current_link_parent/$(basename "$current_link")"

path_must_be_below_release_root() {
  local path="$1"
  case "$path/" in
    "$release_root"/*/) return 0 ;;
    *) die "path escapes release root: $path" ;;
  esac
}

path_must_be_below_release_root "$candidate_dir"
[[ -d "$candidate_dir" ]] || die "candidate release directory not found: $candidate_dir"

candidate_binary="$(realpath -m "$candidate_dir/$binary_relative")"
path_must_be_below_release_root "$candidate_binary"
[[ -f "$candidate_binary" ]] || die "candidate binary not found: $candidate_binary"
[[ -x "$candidate_binary" ]] || die "candidate binary is not executable: $candidate_binary"
[[ -f "$production_env_file" ]] || die "production environment file not found: $production_env_file"
[[ -f "$preflight_env_file" ]] || die "isolated preflight environment file not found: $preflight_env_file"
[[ "$(realpath -m "$production_env_file")" != "$(realpath -m "$preflight_env_file")" ]] || \
  die "preflight must use an isolated environment file, not the production environment"

run_as_service() {
  runuser -u "$service_user" -- "$@"
}

run_as_service test -r "$production_env_file" || die "production environment is unreadable by $service_user"
run_as_service test -r "$preflight_env_file" || die "preflight environment is unreadable by $service_user"
run_as_service test -x "$candidate_binary" || die "candidate binary is not executable by $service_user"

version_output="$(run_as_service "$candidate_binary" -version 2>&1)" || die "candidate version check failed"
[[ -n "$version_output" ]] || die "candidate version output is empty"
log "candidate version: $version_output"

candidate_state_dir="$(mktemp -d "${TMPDIR:-/tmp}/sub2api-preflight.XXXXXX")"
candidate_log="$candidate_state_dir/candidate.log"
candidate_pid_file="$candidate_state_dir/candidate.pid"
chown "$service_user" "$candidate_state_dir"
candidate_pid=""
candidate_wrapper_pid=""
cleanup_candidate() {
  if [[ -n "$candidate_pid" ]] && kill -0 "$candidate_pid" 2>/dev/null; then
    kill -TERM "$candidate_pid" 2>/dev/null || true
    for _ in {1..50}; do
      kill -0 "$candidate_pid" 2>/dev/null || break
      sleep 0.1
    done
    if kill -0 "$candidate_pid" 2>/dev/null; then
      kill -KILL "$candidate_pid" 2>/dev/null || true
    fi
  fi
  if [[ -n "$candidate_wrapper_pid" ]]; then
    wait "$candidate_wrapper_pid" 2>/dev/null || true
  fi
  rm -rf -- "$candidate_state_dir"
}
trap cleanup_candidate EXIT

runuser -u "$service_user" -- /bin/bash -c '
  set -euo pipefail
  set -a
  . "$1"
  set +a
  cd "$2"
  export GIN_MODE=release
  export SERVER_HOST=127.0.0.1
  export SERVER_PORT="$3"
  export BACKGROUND_JOBS_ENABLED=false
  printf "%s\n" "$$" > "$5"
  exec "$4"
' preflight "$preflight_env_file" "$candidate_dir" "$preflight_port" "$candidate_binary" "$candidate_pid_file" >"$candidate_log" 2>&1 &
candidate_wrapper_pid=$!

for _ in {1..50}; do
  [[ -s "$candidate_pid_file" ]] && break
  kill -0 "$candidate_wrapper_pid" 2>/dev/null || break
  sleep 0.1
done
[[ -s "$candidate_pid_file" ]] || {
  tail -n 80 "$candidate_log" >&2 || true
  die "candidate exited before recording its process id"
}
candidate_pid="$(cat "$candidate_pid_file")"
[[ "$candidate_pid" =~ ^[1-9][0-9]*$ ]] || die "candidate process id is invalid"
kill -0 "$candidate_pid" 2>/dev/null || die "candidate process is not running"

probe_health() {
  local url="$1"
  local pid="${2:-}"
  local deadline=$((SECONDS + health_timeout))
  while (( SECONDS < deadline )); do
    if curl --fail --silent --show-error --max-time 2 "$url" >/dev/null; then
      return 0
    fi
    if [[ -n "$pid" ]] && ! kill -0 "$pid" 2>/dev/null; then
      return 1
    fi
    sleep 1
  done
  return 1
}

if ! probe_health "$preflight_health_url" "$candidate_pid"; then
  tail -n 80 "$candidate_log" >&2 || true
  die "candidate failed isolated startup health check"
fi
log "candidate isolated health check passed"
cleanup_candidate
candidate_pid=""
candidate_wrapper_pid=""

[[ -L "$current_link" ]] || die "current release link is missing or is not a symlink: $current_link"
previous_release="$(readlink -f "$current_link")"
path_must_be_below_release_root "$previous_release"
[[ -d "$previous_release" ]] || die "previous release target is missing: $previous_release"

atomic_link() {
  local target="$1"
  local temp_link="${current_link}.next.$$"
  path_must_be_below_release_root "$target"
  rm -f -- "$temp_link"
  ln -s "$target" "$temp_link"
  mv -Tf "$temp_link" "$current_link"
}

rollback() {
  log "rolling back to $previous_release"
  atomic_link "$previous_release"
  systemctl restart "$service_name" || die "rollback restart failed"
  probe_health "$production_health_url" || die "rollback health check failed"
  log "rollback health check passed"
}

atomic_link "$candidate_dir"
if ! systemctl restart "$service_name"; then
  rollback
  die "candidate service restart failed; previous release restored"
fi
if ! probe_health "$production_health_url"; then
  rollback
  die "candidate production health check failed; previous release restored"
fi

log "release activated: $candidate_dir"
