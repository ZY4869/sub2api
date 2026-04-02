#!/usr/bin/env python3
from __future__ import annotations

import argparse
import hashlib
import json
import os
import shutil
import subprocess
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[1]
BACKEND_DIR = REPO_ROOT / "backend"
FRONTEND_DIR = REPO_ROOT / "frontend"
CACHE_BIN_DIR = REPO_ROOT / ".cache" / "verify-ci" / "bin"
FRONTEND_CACHE_DIR = REPO_ROOT / ".cache" / "verify-ci"
FRONTEND_DEPS_STATE_FILE = FRONTEND_CACHE_DIR / "frontend-deps.json"
DEFAULT_GOLANGCI_LINT_VERSION = "v2.9"
DEFAULT_GOVULNCHECK_VERSION = "v1.1.4"

frontend_deps_installed = False


def info(message: str) -> None:
    print(f"[verify-ci] {message}", flush=True)


def run(
    command: list[str],
    *,
    cwd: Path | None = None,
    env: dict[str, str] | None = None,
    capture_output: bool = False,
    check: bool = True,
) -> subprocess.CompletedProcess[str]:
    info(f"$ {' '.join(command)}")
    return subprocess.run(
        command,
        cwd=str(cwd) if cwd else None,
        env=env,
        check=check,
        text=True,
        capture_output=capture_output,
    )


def current_platform_exe(name: str) -> str:
    return f"{name}.exe" if os.name == "nt" else name


def normalize_go_tool_version(version: str) -> str:
    return version if version.count(".") >= 2 else f"{version}.0"


def tool_output(command: list[str]) -> str:
    try:
        result = run(command, capture_output=True, check=False)
    except FileNotFoundError:
        return ""
    return f"{result.stdout}\n{result.stderr}"


def ensure_go_tool(binary_name: str, module_path: str, requested_version: str) -> Path:
    CACHE_BIN_DIR.mkdir(parents=True, exist_ok=True)
    expected_version = normalize_go_tool_version(requested_version).lstrip("v")

    path_binary = shutil.which(binary_name)
    if path_binary and expected_version in tool_output([path_binary, "version"]):
        return Path(path_binary)

    cached_binary = CACHE_BIN_DIR / current_platform_exe(binary_name)
    if cached_binary.exists() and expected_version in tool_output([str(cached_binary), "version"]):
        return cached_binary

    env = os.environ.copy()
    env["GOBIN"] = str(CACHE_BIN_DIR)
    run(["go", "install", f"{module_path}@{normalize_go_tool_version(requested_version)}"], cwd=BACKEND_DIR, env=env)
    return cached_binary


def pnpm_base_command() -> list[str]:
    if os.name == "nt":
        pnpm_cmd = shutil.which("pnpm.cmd")
        if pnpm_cmd:
            return [pnpm_cmd]

    pnpm = shutil.which("pnpm")
    if pnpm and not (os.name == "nt" and pnpm.lower().endswith(".ps1")):
        return [pnpm]

    corepack = shutil.which("corepack")
    if corepack:
        return [corepack, "pnpm"]
    raise RuntimeError("pnpm is not available on PATH and corepack is not installed")


def run_pnpm(args: list[str]) -> None:
    run(pnpm_base_command() + args, cwd=FRONTEND_DIR)


def hash_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as file:
        for chunk in iter(lambda: file.read(65536), b""):
            digest.update(chunk)
    return digest.hexdigest()


def frontend_lockfile_hash() -> str:
    return hash_file(FRONTEND_DIR / "pnpm-lock.yaml")


def read_frontend_deps_state() -> dict[str, str]:
    if not FRONTEND_DEPS_STATE_FILE.exists():
        return {}
    try:
        return json.loads(FRONTEND_DEPS_STATE_FILE.read_text(encoding="utf-8"))
    except json.JSONDecodeError:
        return {}


def write_frontend_deps_state(lockfile_hash: str) -> None:
    FRONTEND_CACHE_DIR.mkdir(parents=True, exist_ok=True)
    FRONTEND_DEPS_STATE_FILE.write_text(json.dumps({"lockfile_hash": lockfile_hash}), encoding="utf-8")


def frontend_deps_ready() -> bool:
    required_paths = [
        FRONTEND_DIR / "node_modules",
        FRONTEND_DIR / "node_modules" / "vite" / "bin" / "vite.js",
        FRONTEND_DIR / "node_modules" / "vitest" / "vitest.mjs",
        FRONTEND_DIR / "node_modules" / "eslint" / "bin" / "eslint.js",
        FRONTEND_DIR / "node_modules" / "vue-tsc" / "bin" / "vue-tsc.js",
    ]
    return all(path.exists() for path in required_paths)


def ensure_frontend_deps() -> None:
    global frontend_deps_installed
    if frontend_deps_installed:
        return
    lockfile_hash = frontend_lockfile_hash()
    state = read_frontend_deps_state()

    if frontend_deps_ready():
        if state.get("lockfile_hash") == lockfile_hash:
            info("reusing existing frontend dependencies (lockfile unchanged)")
            frontend_deps_installed = True
            return
        if not os.environ.get("CI"):
            info("reusing existing frontend dependencies in local workspace (lockfile state bootstrap)")
            write_frontend_deps_state(lockfile_hash)
            frontend_deps_installed = True
            return

    run_pnpm(["install", "--frozen-lockfile"])
    write_frontend_deps_state(lockfile_hash)
    frontend_deps_installed = True


def backend_unit() -> None:
    run(["go", "test", "-tags=unit", "./..."], cwd=BACKEND_DIR)


def backend_integration() -> None:
    run(["go", "test", "-tags=integration", "./..."], cwd=BACKEND_DIR)


def backend_lint() -> None:
    requested = os.environ.get("GOLANGCI_LINT_VERSION", DEFAULT_GOLANGCI_LINT_VERSION)
    binary = ensure_go_tool("golangci-lint", "github.com/golangci/golangci-lint/v2/cmd/golangci-lint", requested)
    run([str(binary), "run", "--timeout=30m"], cwd=BACKEND_DIR)


def backend_build() -> None:
    run(["go", "build", "./cmd/server"], cwd=BACKEND_DIR)


def frontend_build() -> None:
    ensure_frontend_deps()
    run_pnpm(["run", "build"])


def frontend_test() -> None:
    ensure_frontend_deps()
    run_pnpm(["run", "test:run"])


def frontend_lint() -> None:
    ensure_frontend_deps()
    run_pnpm(["run", "lint:check"])


def security() -> None:
    requested = os.environ.get("GOVULNCHECK_VERSION", DEFAULT_GOVULNCHECK_VERSION)
    binary = ensure_go_tool("govulncheck", "golang.org/x/vuln/cmd/govulncheck", requested)
    run([str(binary), "./..."], cwd=BACKEND_DIR)

    ensure_frontend_deps()
    audit_path = FRONTEND_DIR / "audit.json"
    audit_result = run(
        pnpm_base_command() + ["audit", "--prod", "--audit-level=high", "--json"],
        cwd=FRONTEND_DIR,
        capture_output=True,
        check=False,
    )
    audit_path.write_text(audit_result.stdout, encoding="utf-8")
    run(
        [
            sys.executable,
            str(REPO_ROOT / "tools" / "check_pnpm_audit_exceptions.py"),
            "--audit",
            str(audit_path),
            "--exceptions",
            str(REPO_ROOT / ".github" / "audit-exceptions.yml"),
        ],
        cwd=REPO_ROOT,
    )


def wait_for_http(url: str, *, expect_content_type: str | None = None, timeout_seconds: int = 45) -> None:
    deadline = time.time() + timeout_seconds
    last_error = "request not attempted"
    while time.time() < deadline:
        try:
            with urllib.request.urlopen(url, timeout=5) as response:
                content_type = response.headers.get("Content-Type", "")
                if response.status == 200 and (
                    expect_content_type is None or expect_content_type in content_type
                ):
                    return
                last_error = f"unexpected response status={response.status} content_type={content_type}"
        except urllib.error.URLError as exc:
            last_error = str(exc)
        time.sleep(1)
    raise RuntimeError(f"timeout waiting for {url}: {last_error}")


def docker_smoke() -> None:
    docker = shutil.which("docker")
    if not docker:
        raise RuntimeError("docker is not available on PATH")

    frontend_build()

    smoke_root = REPO_ROOT / ".cache" / "verify-ci" / "docker-smoke"
    context_dir = smoke_root / "context"
    if smoke_root.exists():
        shutil.rmtree(smoke_root)
    (context_dir / "deploy").mkdir(parents=True, exist_ok=True)

    linux_binary = context_dir / "sub2api"
    build_env = os.environ.copy()
    build_env.update({"GOOS": "linux", "GOARCH": "amd64", "CGO_ENABLED": "0"})
    run(["go", "build", "-tags=embed", "-o", str(linux_binary), "./cmd/server"], cwd=BACKEND_DIR, env=build_env)

    shutil.copy2(REPO_ROOT / "Dockerfile.goreleaser", context_dir / "Dockerfile.goreleaser")
    shutil.copy2(REPO_ROOT / "deploy" / "docker-entrypoint.sh", context_dir / "deploy" / "docker-entrypoint.sh")

    tag = f"sub2api-ci-smoke:{int(time.time())}"
    container_name = f"sub2api-ci-smoke-{int(time.time())}"
    run([docker, "build", "-f", str(context_dir / "Dockerfile.goreleaser"), "-t", tag, str(context_dir)], cwd=REPO_ROOT)

    container_started = False
    try:
        run([docker, "run", "-d", "-P", "--name", container_name, tag], cwd=REPO_ROOT)
        container_started = True
        port_output = run([docker, "port", container_name, "8080/tcp"], cwd=REPO_ROOT, capture_output=True).stdout.strip()
        if not port_output:
            raise RuntimeError("docker did not expose port 8080")
        host_port = port_output.splitlines()[0].rsplit(":", 1)[-1].strip()
        wait_for_http(f"http://127.0.0.1:{host_port}/health")
        wait_for_http(f"http://127.0.0.1:{host_port}/", expect_content_type="text/html")
    finally:
        if container_started:
            run([docker, "rm", "-f", container_name], cwd=REPO_ROOT, check=False)
        run([docker, "image", "rm", "-f", tag], cwd=REPO_ROOT, check=False)


def full() -> None:
    backend_unit()
    backend_integration()
    backend_lint()
    backend_build()
    frontend_build()
    frontend_test()
    frontend_lint()
    security()


def release_preflight() -> None:
    full()
    docker_smoke()


MODES: dict[str, callable] = {
    "backend-unit": backend_unit,
    "backend-integration": backend_integration,
    "backend-lint": backend_lint,
    "backend-build": backend_build,
    "frontend-build": frontend_build,
    "frontend-test": frontend_test,
    "frontend-lint": frontend_lint,
    "security": security,
    "docker-smoke": docker_smoke,
    "full": full,
    "release-preflight": release_preflight,
}


def main() -> int:
    parser = argparse.ArgumentParser(description="Repository-standard CI verification entrypoint")
    parser.add_argument("mode", choices=sorted(MODES))
    args = parser.parse_args()
    MODES[args.mode]()
    return 0


if __name__ == "__main__":
    sys.exit(main())
