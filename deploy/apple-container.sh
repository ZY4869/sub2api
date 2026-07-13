#!/usr/bin/env sh
set -eu

DEFAULT_GITHUB_REPO="ZY4869/sub2api"
DEFAULT_SUB2API_IMAGE="ghcr.io/zy4869/sub2api:latest"

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
ACTION="${1:-up}"
CONTAINER_CLI="${CONTAINER_CLI:-container}"
CONTAINER_NAME="${SUB2API_CONTAINER_NAME:-sub2api}"
SUB2API_IMAGE="${SUB2API_IMAGE:-$DEFAULT_SUB2API_IMAGE}"
BIND_HOST="${BIND_HOST:-127.0.0.1}"
SERVER_PORT="${SERVER_PORT:-8080}"
DATA_DIR="${SUB2API_DATA_DIR:-$SCRIPT_DIR/apple-container-data}"

DATABASE_HOST="${DATABASE_HOST:-host.docker.internal}"
DATABASE_PORT="${DATABASE_PORT:-5432}"
DATABASE_USER="${DATABASE_USER:-${POSTGRES_USER:-sub2api}}"
DATABASE_PASSWORD="${DATABASE_PASSWORD:-${POSTGRES_PASSWORD:-}}"
DATABASE_DBNAME="${DATABASE_DBNAME:-${POSTGRES_DB:-sub2api}}"
DATABASE_SSLMODE="${DATABASE_SSLMODE:-disable}"

REDIS_HOST="${REDIS_HOST:-host.docker.internal}"
REDIS_PORT="${REDIS_PORT:-6379}"
REDIS_PASSWORD="${REDIS_PASSWORD:-}"
REDIS_DB="${REDIS_DB:-0}"

print_info() {
  printf '%s\n' "[apple-container] $*"
}

print_warning() {
  printf '%s\n' "[apple-container] warning: $*" >&2
}

require_container_cli() {
  if ! command -v "$CONTAINER_CLI" >/dev/null 2>&1; then
    print_warning "Apple Container CLI '$CONTAINER_CLI' was not found in PATH."
    print_warning "Install it first, or set CONTAINER_CLI to the compatible command."
    exit 127
  fi
}

start_system() {
  "$CONTAINER_CLI" system start >/dev/null 2>&1 || true
}

stop_container() {
  "$CONTAINER_CLI" stop "$CONTAINER_NAME" >/dev/null 2>&1 || true
}

remove_container() {
  "$CONTAINER_CLI" rm "$CONTAINER_NAME" >/dev/null 2>&1 || true
}

run_container() {
  mkdir -p "$DATA_DIR"

  if [ -z "$DATABASE_PASSWORD" ]; then
    print_warning "DATABASE_PASSWORD/POSTGRES_PASSWORD is empty; ensure your PostgreSQL accepts this connection."
  fi

  print_info "Reference repo: $DEFAULT_GITHUB_REPO"
  print_info "Image: $SUB2API_IMAGE"
  print_info "Name: $CONTAINER_NAME"
  print_info "URL: http://$BIND_HOST:$SERVER_PORT"
  print_info "Data: $DATA_DIR"

  start_system
  "$CONTAINER_CLI" image pull "$SUB2API_IMAGE"
  stop_container
  remove_container

  "$CONTAINER_CLI" run \
    --detach \
    --name "$CONTAINER_NAME" \
    --publish "$BIND_HOST:$SERVER_PORT:8080" \
    --volume "$DATA_DIR:/app/data" \
    --env AUTO_SETUP="${AUTO_SETUP:-false}" \
    --env SERVER_HOST="0.0.0.0" \
    --env SERVER_PORT="8080" \
    --env SERVER_MODE="${SERVER_MODE:-release}" \
    --env RUN_MODE="${RUN_MODE:-standard}" \
    --env DATABASE_HOST="$DATABASE_HOST" \
    --env DATABASE_PORT="$DATABASE_PORT" \
    --env DATABASE_USER="$DATABASE_USER" \
    --env DATABASE_PASSWORD="$DATABASE_PASSWORD" \
    --env DATABASE_DBNAME="$DATABASE_DBNAME" \
    --env DATABASE_SSLMODE="$DATABASE_SSLMODE" \
    --env DATABASE_MAX_OPEN_CONNS="${DATABASE_MAX_OPEN_CONNS:-50}" \
    --env DATABASE_MAX_IDLE_CONNS="${DATABASE_MAX_IDLE_CONNS:-10}" \
    --env REDIS_HOST="$REDIS_HOST" \
    --env REDIS_PORT="$REDIS_PORT" \
    --env REDIS_PASSWORD="$REDIS_PASSWORD" \
    --env REDIS_DB="$REDIS_DB" \
    --env REDIS_ENABLE_TLS="${REDIS_ENABLE_TLS:-false}" \
    --env ADMIN_EMAIL="${ADMIN_EMAIL:-admin@sub2api.local}" \
    --env ADMIN_PASSWORD="${ADMIN_PASSWORD:-}" \
    --env JWT_SECRET="${JWT_SECRET:-}" \
    --env TOTP_ENCRYPTION_KEY="${TOTP_ENCRYPTION_KEY:-}" \
    --env TZ="${TZ:-Asia/Shanghai}" \
    --env GEMINI_OAUTH_CLIENT_ID="${GEMINI_OAUTH_CLIENT_ID:-}" \
    --env GEMINI_OAUTH_CLIENT_SECRET="${GEMINI_OAUTH_CLIENT_SECRET:-}" \
    --env GEMINI_OAUTH_SCOPES="${GEMINI_OAUTH_SCOPES:-}" \
    --env GEMINI_CLI_OAUTH_CLIENT_SECRET="${GEMINI_CLI_OAUTH_CLIENT_SECRET:-}" \
    --env ANTIGRAVITY_OAUTH_CLIENT_SECRET="${ANTIGRAVITY_OAUTH_CLIENT_SECRET:-}" \
    --env SECURITY_URL_ALLOWLIST_ENABLED="${SECURITY_URL_ALLOWLIST_ENABLED:-false}" \
    --env SECURITY_URL_ALLOWLIST_ALLOW_INSECURE_HTTP="${SECURITY_URL_ALLOWLIST_ALLOW_INSECURE_HTTP:-false}" \
    --env SECURITY_URL_ALLOWLIST_ALLOW_PRIVATE_HOSTS="${SECURITY_URL_ALLOWLIST_ALLOW_PRIVATE_HOSTS:-false}" \
    "$SUB2API_IMAGE" \
    /app/sub2api
}

case "$ACTION" in
  up|start)
    require_container_cli
    run_container
    ;;
  stop)
    require_container_cli
    stop_container
    ;;
  restart)
    require_container_cli
    stop_container
    run_container
    ;;
  rm|remove)
    require_container_cli
    stop_container
    remove_container
    ;;
  logs)
    require_container_cli
    "$CONTAINER_CLI" logs --follow "$CONTAINER_NAME"
    ;;
  status|ps)
    require_container_cli
    "$CONTAINER_CLI" list --all | grep "$CONTAINER_NAME" || true
    ;;
  *)
    cat <<EOF
Usage: $0 [up|start|stop|restart|rm|logs|status]

Environment:
  SUB2API_IMAGE=$DEFAULT_SUB2API_IMAGE
  SUB2API_CONTAINER_NAME=$CONTAINER_NAME
  SERVER_PORT=$SERVER_PORT
  DATABASE_HOST=$DATABASE_HOST
  REDIS_HOST=$REDIS_HOST
EOF
    exit 2
    ;;
esac
