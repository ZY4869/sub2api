#!/usr/bin/env bash
# Main flow for the WSL local deployment helper.

usage() {
    cat <<EOF
Usage: ./wsl-local-deploy.sh [options]

Options:
  --diagnose       Only print WSL/Docker/Compose diagnostics.
  --no-pull        Recreate/start with the currently available image.
  -h, --help       Show this help message.

Environment:
  SUB2API_DEPLOY_DIR       Deployment directory. Default: this script directory.
  SUB2API_COMPOSE_FILE     Compose file name. Default: docker-compose.local.yml
  SUB2API_ENV_FILE         Env file name. Default: .env
  SUB2API_DOCKER_TIMEOUT   Docker readiness timeout in seconds. Default: 150
  SUB2API_HEALTH_TIMEOUT   App health timeout in seconds. Default: 150
EOF
}

ensure_files() {
    cd "$DEPLOY_DIR"
    if [ ! -f "$COMPOSE_FILE" ]; then
        print_err "Compose file not found: ${DEPLOY_DIR}/${COMPOSE_FILE}"
        exit 1
    fi
    if [ ! -f "$ENV_FILE" ]; then
        print_err "Env file not found: ${DEPLOY_DIR}/${ENV_FILE}"
        print_info "Create it from .env.example or run deploy/docker-deploy.sh first."
        exit 1
    fi
}

run_diagnose() {
    cd "$DEPLOY_DIR" 2>/dev/null || true
    print_diagnostics
    if command_exists docker && docker_ready; then
        compose_cmd
        "${COMPOSE[@]}" ps || true
    else
        print_warn "Docker is not ready; skipped compose ps."
    fi
}

run_deploy() {
    ensure_files
    ensure_docker
    compose_cmd
    deploy_stack
    "${COMPOSE[@]}" ps
    wait_health
}

main() {
    parse_args "$@"
    print_info "Using deployment directory: ${DEPLOY_DIR}"
    is_wsl || print_warn "This helper is optimized for WSL, but the current kernel does not look like WSL."
    if [ "$MODE" = diagnose ]; then
        run_diagnose
        exit 0
    fi
    run_deploy
}
