#!/usr/bin/env bash
# Shared functions for wsl-local-deploy.sh.

BLUE='\033[0;34m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

DEPLOY_DIR="${SUB2API_DEPLOY_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)}"
COMPOSE_FILE="${SUB2API_COMPOSE_FILE:-docker-compose.local.yml}"
ENV_FILE="${SUB2API_ENV_FILE:-.env}"
SERVICE="${SUB2API_SERVICE:-sub2api}"
DOCKER_TIMEOUT="${SUB2API_DOCKER_TIMEOUT:-150}"
HEALTH_TIMEOUT="${SUB2API_HEALTH_TIMEOUT:-150}"
PULL_IMAGE=true
MODE=deploy

print_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
print_ok() { echo -e "${GREEN}[OK]${NC} $1"; }
print_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
print_err() { echo -e "${RED}[ERROR]${NC} $1"; }
command_exists() { command -v "$1" >/dev/null 2>&1; }
is_wsl() { grep -qiE 'microsoft|wsl' /proc/version 2>/dev/null; }
docker_ready() { timeout 12s docker info >/dev/null 2>&1; }

parse_args() {
    for arg in "$@"; do
        case "$arg" in
            --diagnose) MODE=diagnose ;;
            --no-pull) PULL_IMAGE=false ;;
            -h|--help) usage; exit 0 ;;
            *) print_err "Unknown option: $arg"; usage; exit 2 ;;
        esac
    done
}

run_root() {
    if [ "$(id -u)" -eq 0 ]; then
        "$@"
    elif command_exists sudo && sudo -n true >/dev/null 2>&1; then
        sudo "$@"
    else
        return 1
    fi
}

docker_ping() {
    if command_exists curl && [ -S /var/run/docker.sock ]; then
        timeout 8s curl --unix-socket /var/run/docker.sock -fsS http://localhost/_ping >/dev/null 2>&1
    else
        docker_ready
    fi
}

safe_root_shell() {
    local cmd="$1"
    timeout 12s bash -lc "$cmd" 2>&1 || run_root timeout 12s bash -lc "$cmd" 2>&1 || true
}

print_diagnostics() {
    print_warn "WSL/Docker diagnostics follow. Secrets are not printed."
    echo "Deploy dir: ${DEPLOY_DIR}"
    echo "Compose file: ${COMPOSE_FILE}"
    echo "Env file: ${ENV_FILE}"
    echo "WSL: $(is_wsl && echo yes || echo no)"
    echo "systemd state:"
    safe_root_shell "systemctl is-active containerd docker docker.socket || true"
    echo "systemd jobs:"
    safe_root_shell "systemctl list-jobs --no-pager || true"
    echo "Docker-related processes:"
    safe_root_shell "ps -eo pid,ppid,stat,wchan:32,comm,args | grep -E 'dockerd|containerd|containerd-shim|docker-proxy' | grep -v grep || true"
    echo "Recent kernel messages:"
    safe_root_shell "dmesg -T | tail -80 || true"
}

ensure_docker() {
    if ! command_exists docker; then
        print_err "docker command is not installed in WSL."
        exit 1
    fi
    if docker_ready; then
        print_ok "Docker daemon is ready."
        return
    fi
    print_warn "Docker daemon is not ready. Trying a conservative systemd start..."
    run_root timeout 90s systemctl enable --now containerd docker.socket docker || \
        print_warn "Could not start Docker via systemd without interactive sudo/root."
    wait_docker_ready
}

wait_docker_ready() {
    local start_time elapsed
    start_time=$(date +%s)
    while true; do
        if docker_ready; then
            print_ok "Docker daemon is ready."
            return
        fi
        elapsed=$(( $(date +%s) - start_time ))
        if [ "$elapsed" -ge "$DOCKER_TIMEOUT" ]; then
            print_err "Docker daemon did not become ready within ${DOCKER_TIMEOUT}s."
            print_diagnostics
            exit 1
        fi
        docker_ping || print_info "Waiting for Docker daemon... ${elapsed}s"
        sleep 5
    done
}

compose_cmd() {
    if docker compose version >/dev/null 2>&1; then
        COMPOSE=(docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE")
    elif command_exists docker-compose && docker-compose version >/dev/null 2>&1; then
        COMPOSE=(docker-compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE")
    else
        print_err "Docker Compose v2 plugin or docker-compose is required."
        exit 1
    fi
}

env_value() {
    local key="$1" line value
    line=$(grep -E "^[[:space:]]*${key}=" "$ENV_FILE" | tail -n 1 || true)
    value="${line#*=}"
    value="${value%%#*}"
    value="${value%$'\r'}"
    printf '%s' "$value" | sed -e 's/^[[:space:]"'\'']*//' -e 's/[[:space:]"'\'']*$//'
}

local_url() {
    local host port
    host="$(env_value BIND_HOST)"
    port="$(env_value SERVER_PORT)"
    host="${host:-127.0.0.1}"
    port="${port:-8080}"
    if [ "$host" = "0.0.0.0" ] || [ "$host" = "::" ]; then
        host="127.0.0.1"
    fi
    printf 'http://%s:%s' "$host" "$port"
}

has_existing_local_data() {
    [ -d data ] || [ -d postgres_data ] || [ -d redis_data ]
}

deploy_stack() {
    if has_existing_local_data; then
        print_info "Existing local deployment detected."
        if [ "$PULL_IMAGE" = true ]; then
            print_info "Pulling latest application image..."
            "${COMPOSE[@]}" pull "$SERVICE"
        else
            print_warn "Skipping image pull because --no-pull was provided."
        fi
        print_info "Recreating application container..."
        "${COMPOSE[@]}" up -d --force-recreate "$SERVICE"
        return
    fi
    print_info "No local data directories found. Starting as first-time deployment..."
    mkdir -p data postgres_data redis_data
    "${COMPOSE[@]}" up -d
}

wait_health() {
    local url health_url start_time elapsed
    url="$(local_url)"
    health_url="${url}/health"
    start_time=$(date +%s)
    print_info "Waiting for health check: ${health_url}"
    while true; do
        if command_exists curl && curl -fsS "$health_url" >/dev/null 2>&1; then
            print_ok "Sub2API is healthy: ${url}"
            return
        fi
        elapsed=$(( $(date +%s) - start_time ))
        if [ "$elapsed" -ge "$HEALTH_TIMEOUT" ]; then
            print_err "Sub2API did not pass health check within ${HEALTH_TIMEOUT}s."
            "${COMPOSE[@]}" ps || true
            "${COMPOSE[@]}" logs --tail=80 "$SERVICE" || true
            exit 1
        fi
        sleep 5
    done
}
