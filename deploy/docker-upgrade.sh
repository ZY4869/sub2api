#!/bin/bash
# =============================================================================
# Sub2API Docker Upgrade Script
# =============================================================================
# This script upgrades an existing Docker deployment to the latest GHCR image
# for the configured repository. It creates a tar.gz backup before pulling the
# new image and recreating the application container.
# =============================================================================

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

DEFAULT_GITHUB_REPO="ZY4869/sub2api"
DEFAULT_GITHUB_REF="main"
GITHUB_REPO="${SUB2API_GITHUB_REPO:-$DEFAULT_GITHUB_REPO}"
GITHUB_REF="${SUB2API_GITHUB_REF:-$DEFAULT_GITHUB_REF}"
GITHUB_REPO_LOWER=$(echo "${GITHUB_REPO}" | tr '[:upper:]' '[:lower:]')
TARGET_IMAGE="${SUB2API_TARGET_IMAGE:-ghcr.io/${GITHUB_REPO_LOWER}:latest}"
DEPLOY_DIR="${SUB2API_DEPLOY_DIR:-$(pwd)}"
BACKUP_DIR="${SUB2API_BACKUP_DIR:-${DEPLOY_DIR}/backups}"
COMPOSE_FILE="${DEPLOY_DIR}/docker-compose.yml"
ENV_FILE="${DEPLOY_DIR}/.env"
DEFAULT_IMAGE_EXPR='${SUB2API_IMAGE:-ghcr.io/zy4869/sub2api:latest}'

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

command_exists() {
    command -v "$1" >/dev/null 2>&1
}

ensure_file() {
    local path="$1"
    if [ ! -f "$path" ]; then
        print_error "Required file not found: $path"
        exit 1
    fi
}

upsert_env_var() {
    local key="$1"
    local value="$2"

    if grep -q "^${key}=" "$ENV_FILE"; then
        sed -i "s#^${key}=.*#${key}=${value}#" "$ENV_FILE"
    else
        printf '\n%s=%s\n' "$key" "$value" >> "$ENV_FILE"
    fi
}

migrate_compose_image() {
    if grep -q 'SUB2API_IMAGE' "$COMPOSE_FILE"; then
        return
    fi

    sed -i "0,/^[[:space:]]*image:[[:space:]]*/s#^[[:space:]]*image:[[:space:]].*#    image: ${DEFAULT_IMAGE_EXPR}#" "$COMPOSE_FILE"
}

main() {
    print_info "Using deployment directory: ${DEPLOY_DIR}"
    print_info "Using target image: ${TARGET_IMAGE}"
    print_info "Using source branch for docs/scripts: ${GITHUB_REF}"

    if ! command_exists docker; then
        print_error "docker is not installed."
        exit 1
    fi

    ensure_file "$COMPOSE_FILE"
    ensure_file "$ENV_FILE"

    cd "$DEPLOY_DIR"
    mkdir -p "$BACKUP_DIR"

    local timestamp
    timestamp=$(date +%F-%H%M%S)
    local backup_file="${BACKUP_DIR}/sub2api-${timestamp}.tar.gz"
    local -a backup_targets=(".env" "docker-compose.yml")

    if [ -d "${DEPLOY_DIR}/data" ]; then
        backup_targets+=("data")
    fi
    if [ -d "${DEPLOY_DIR}/postgres_data" ]; then
        backup_targets+=("postgres_data")
    else
        print_warning "postgres_data directory not found; Docker named volume data will not be included in the tar backup."
    fi
    if [ -d "${DEPLOY_DIR}/redis_data" ]; then
        backup_targets+=("redis_data")
    else
        print_warning "redis_data directory not found; Docker named volume data will not be included in the tar backup."
    fi

    print_warning "Stopping containers for a consistent backup..."
    docker compose down

    print_info "Creating backup: ${backup_file}"
    tar czf "$backup_file" "${backup_targets[@]}"
    print_success "Backup created: ${backup_file}"

    print_info "Switching deployment to image: ${TARGET_IMAGE}"
    upsert_env_var "SUB2API_IMAGE" "$TARGET_IMAGE"
    migrate_compose_image

    print_info "Pulling latest application image..."
    docker compose pull sub2api

    print_info "Starting services..."
    docker compose up -d

    local running_image=""
    if docker inspect sub2api >/dev/null 2>&1; then
        running_image=$(docker inspect sub2api --format '{{.Config.Image}}')
    fi

    docker compose ps
    print_success "Upgrade complete."
    if [ -n "$running_image" ]; then
        print_success "Running image: ${running_image}"
    fi
    print_success "Backup saved to: ${backup_file}"
}

main "$@"
