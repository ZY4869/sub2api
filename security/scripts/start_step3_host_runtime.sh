#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../.." && pwd)"
SERVER_BIN="$REPO_ROOT/backend/bin/server-test-linux"
ENV_FILE="$REPO_ROOT/.env.audit"
TMP_DIR="$REPO_ROOT/tmp"
REPORT_DIR="$REPO_ROOT/security/reports/manual"
DATA_DIR_DEFAULT="$REPO_ROOT/tmp/sub2api-audit-data"
SERVER_PID_FILE="$TMP_DIR/server-audit.pid"
MOCK_PID_FILE="$TMP_DIR/mock-upstream.pid"
SERVER_LOG="$REPORT_DIR/server-audit.log"
MOCK_LOG="$REPORT_DIR/mock-upstream.log"

load_env_file() {
  local file="$1"
  while IFS= read -r raw || [[ -n "$raw" ]]; do
    raw="${raw%$'\r'}"
    [[ -z "$raw" ]] && continue
    [[ "${raw:0:1}" == "#" ]] && continue
    local key="${raw%%=*}"
    local value="${raw#*=}"
    export "$key=$value"
  done < "$file"
}

stop_pid_file() {
  local pid_file="$1"
  if [[ ! -f "$pid_file" ]]; then
    return 0
  fi

  local pid
  pid="$(tr -d '\r\n' < "$pid_file")"
  if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
    kill "$pid" || true
    sleep 2
  fi
  rm -f "$pid_file"
}

ensure_dirs() {
  mkdir -p "$TMP_DIR" "$REPORT_DIR" "$DATA_DIR_DEFAULT"
}

start_mock_upstream() {
  if [[ -f "$MOCK_PID_FILE" ]]; then
    local pid
    pid="$(tr -d '\r\n' < "$MOCK_PID_FILE")"
    if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
      return 0
    fi
  fi

  nohup python3 "$REPO_ROOT/security/mock/mock_upstream.py" > "$MOCK_LOG" 2>&1 &
  echo "$!" > "$MOCK_PID_FILE"
  sleep 2
}

start_server() {
  if [[ ! -x "$SERVER_BIN" ]]; then
    echo "server binary not found or not executable: $SERVER_BIN" >&2
    exit 1
  fi

  if [[ ! -f "$ENV_FILE" ]]; then
    echo "env file not found: $ENV_FILE" >&2
    exit 1
  fi

  stop_pid_file "$SERVER_PID_FILE"
  pkill -f "$SERVER_BIN" 2>/dev/null || true
  sleep 2

  load_env_file "$ENV_FILE"
  export DATA_DIR="${DATA_DIR:-$DATA_DIR_DEFAULT}"
  mkdir -p "$DATA_DIR"

  nohup "$SERVER_BIN" > "$SERVER_LOG" 2>&1 &
  echo "$!" > "$SERVER_PID_FILE"
}

print_status() {
  sleep 10
  echo "--- listeners ---"
  ss -ltnp | grep -E ':(8080|18081|19090)' || true
  echo "--- env summary ---"
  printf 'SERVER_PORT=%s\n' "${SERVER_PORT:-}"
  printf 'DATA_DIR=%s\n' "${DATA_DIR:-}"
  printf 'DATABASE_HOST=%s\n' "${DATABASE_HOST:-}"
  printf 'DATABASE_PORT=%s\n' "${DATABASE_PORT:-}"
  printf 'REDIS_HOST=%s\n' "${REDIS_HOST:-}"
  printf 'REDIS_PORT=%s\n' "${REDIS_PORT:-}"
  echo "--- server log tail ---"
  tail -n 80 "$SERVER_LOG" || true
  echo "--- mock log tail ---"
  tail -n 20 "$MOCK_LOG" || true
}

main() {
  ensure_dirs
  start_mock_upstream
  start_server
  print_status
}

main "$@"
