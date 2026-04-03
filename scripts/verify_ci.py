#!/usr/bin/env python3
from __future__ import annotations

import argparse
import contextlib
import hashlib
import json
import os
import re
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
DEFAULT_NPM_AUDIT_REGISTRY = "https://registry.npmjs.org"

frontend_deps_installed = False


def info(message: str) -> None:
    print(f"[verify-ci] {message}", flush=True)


def emit_console_line(line: str) -> None:
    try:
        print(line, end="")
    except UnicodeEncodeError:
        encoding = sys.stdout.encoding or "utf-8"
        if hasattr(sys.stdout, "buffer"):
            sys.stdout.buffer.write(line.encode(encoding, errors="replace"))
            sys.stdout.buffer.flush()
        else:
            sys.stdout.write(line.encode(encoding, errors="replace").decode(encoding))
            sys.stdout.flush()


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
        encoding="utf-8",
        errors="replace",
        capture_output=capture_output,
    )


def run_streaming(
    command: list[str],
    *,
    cwd: Path | None = None,
    env: dict[str, str] | None = None,
    log_path: Path | None = None,
    check: bool = True,
) -> subprocess.CompletedProcess[str]:
    info(f"$ {' '.join(command)}")
    if log_path is not None:
        log_path.parent.mkdir(parents=True, exist_ok=True)
        info(f"streaming command output to {log_path}")

    process = subprocess.Popen(
        command,
        cwd=str(cwd) if cwd else None,
        env=env,
        text=True,
        encoding="utf-8",
        errors="replace",
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
    )

    output_parts: list[str] = []
    with (log_path.open("w", encoding="utf-8") if log_path else contextlib.nullcontext()) as log_file:
        assert process.stdout is not None
        for line in process.stdout:
            emit_console_line(line)
            output_parts.append(line)
            if log_file is not None:
                log_file.write(line)

    return_code = process.wait()
    stdout = "".join(output_parts)
    completed = subprocess.CompletedProcess(command, return_code, stdout=stdout, stderr="")
    if check and return_code != 0:
        raise subprocess.CalledProcessError(return_code, command, output=stdout)
    return completed


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


def version_probe_command(binary_name: str, binary_path: str) -> list[str]:
    if binary_name == "govulncheck":
        return [binary_path, "-version"]
    return [binary_path, "version"]


def github_actions_escape(value: str) -> str:
    return value.replace("%", "%25").replace("\r", "%0D").replace("\n", "%0A")


def emit_github_annotation(level: str, title: str, message: str) -> None:
    escaped_title = github_actions_escape(title)
    escaped_message = github_actions_escape(message)
    print(f"::{level} title={escaped_title}::{escaped_message}", flush=True)


def append_step_summary(lines: list[str]) -> None:
    summary_path = os.environ.get("GITHUB_STEP_SUMMARY")
    if not summary_path:
        return
    with Path(summary_path).open("a", encoding="utf-8") as handle:
        handle.write("\n".join(lines) + "\n")


def summarize_go_test_failure(log_text: str) -> dict[str, list[str]]:
    failed_tests: list[str] = []
    failed_packages: list[str] = []
    diagnostic_lines: list[str] = []
    failure_blocks: list[dict[str, list[str] | str]] = []
    lines = log_text.splitlines()

    for line in lines:
        if match := re.match(r"^--- FAIL: (\S+)", line):
            failed_tests.append(match.group(1))
        if match := re.match(r"^FAIL\s+(\S+)", line):
            package_name = match.group(1)
            if package_name != "FAIL":
                failed_packages.append(package_name)
        if line.startswith("panic:") or line.startswith("# ") or "[build failed]" in line:
            diagnostic_lines.append(line)

    index = 0
    while index < len(lines):
        match = re.match(r"^--- FAIL: (\S+)", lines[index])
        if not match:
            index += 1
            continue

        test_name = match.group(1)
        details: list[str] = []
        block_index = index + 1
        while block_index < len(lines):
            current = lines[block_index]
            if re.match(r"^(--- (PASS|FAIL|SKIP):|=== RUN|FAIL\s+\S+|ok\s+\S+|\?\s+\S+)", current):
                break

            stripped = current.strip()
            if stripped:
                details.append(stripped)
            block_index += 1

        normalized_details = list(dict.fromkeys(details))
        interesting_details = [
            line
            for line in normalized_details
            if any(
                marker in line
                for marker in (
                    "--- FAIL:",
                    "panic:",
                    "Error:",
                    "Messages:",
                    "Received unexpected error:",
                    "Not equal:",
                    "Should be",
                    "Trace:",
                    "[build failed]",
                )
            )
        ]
        detail_lines = interesting_details or normalized_details
        if detail_lines:
            failure_blocks.append(
                {
                    "test": test_name,
                    "details": detail_lines[:8],
                }
            )
        index = block_index

    return {
        "tests": list(dict.fromkeys(failed_tests)),
        "packages": list(dict.fromkeys(failed_packages)),
        "diagnostics": list(dict.fromkeys(diagnostic_lines[-20:])),
        "failure_blocks": failure_blocks[:10],
        "tail": log_text.splitlines()[-40:],
    }


def emit_backend_integration_failure_summary(log_text: str, log_path: Path) -> None:
    summary = summarize_go_test_failure(log_text)
    failed_tests = summary["tests"]
    failed_packages = summary["packages"]
    diagnostics = summary["diagnostics"]
    failure_blocks = summary["failure_blocks"]
    tail_lines = summary["tail"]

    summary_lines = ["backend-integration failed"]
    if failed_packages:
        summary_lines.append(f"packages: {', '.join(failed_packages[:10])}")
    if failed_tests:
        summary_lines.append(f"tests: {', '.join(failed_tests[:10])}")
    if diagnostics:
        summary_lines.append(f"diagnostics: {' | '.join(diagnostics[:5])}")
    if len(summary_lines) == 1 and tail_lines:
        summary_lines.append(f"tail: {' | '.join(tail_lines[-5:])}")
    summary_lines.append(f"log path: {log_path}")

    for line in summary_lines:
        info(line)

    emit_github_annotation("error", "backend-integration", " | ".join(summary_lines[:4]))

    for package_name in failed_packages[:10]:
        emit_github_annotation("error", "backend-integration package", package_name)
    for test_name in failed_tests[:10]:
        emit_github_annotation("error", "backend-integration test", test_name)
    for diagnostic in diagnostics[:10]:
        emit_github_annotation("error", "backend-integration diagnostic", diagnostic)
    for failure_block in failure_blocks[:10]:
        test_name = str(failure_block["test"])
        details = [str(line) for line in failure_block["details"]]
        emit_github_annotation(
            "error",
            f"backend-integration detail {test_name}",
            " | ".join(details[:6]),
        )

    step_summary = [
        "### backend-integration failed",
        "",
        f"- Log: `{log_path}`",
    ]
    if failed_packages:
        step_summary.append(f"- Packages: `{', '.join(failed_packages[:10])}`")
    if failed_tests:
        step_summary.append(f"- Tests: `{', '.join(failed_tests[:10])}`")
    if diagnostics:
        step_summary.append(f"- Diagnostics: `{ ' | '.join(diagnostics[:5]) }`")
    if failure_blocks:
        step_summary.append("")
        step_summary.append("#### Failure Details")
        for failure_block in failure_blocks[:5]:
            test_name = str(failure_block["test"])
            details = [str(line) for line in failure_block["details"]]
            step_summary.append(f"- `{test_name}`: {' | '.join(details[:6])}")
    if tail_lines:
        step_summary.append("")
        step_summary.append("```text")
        step_summary.extend(tail_lines[-20:])
        step_summary.append("```")
    append_step_summary(step_summary)


def ensure_go_tool(binary_name: str, module_path: str, requested_version: str) -> Path:
    CACHE_BIN_DIR.mkdir(parents=True, exist_ok=True)
    expected_version = normalize_go_tool_version(requested_version).lstrip("v")

    path_binary = shutil.which(binary_name)
    if path_binary and expected_version in tool_output(version_probe_command(binary_name, path_binary)):
        return Path(path_binary)

    cached_binary = CACHE_BIN_DIR / current_platform_exe(binary_name)
    if cached_binary.exists() and expected_version in tool_output(version_probe_command(binary_name, str(cached_binary))):
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


def extract_json_document(*streams: str) -> dict | list | None:
    decoder = json.JSONDecoder()
    for stream in streams:
        text = (stream or "").strip()
        if not text:
            continue
        try:
            return json.loads(text)
        except json.JSONDecodeError:
            pass
        for line in reversed([candidate.strip() for candidate in text.splitlines() if candidate.strip()]):
            try:
                return json.loads(line)
            except json.JSONDecodeError:
                continue
        for index, char in enumerate(text):
            if char not in "{[":
                continue
            try:
                document, end = decoder.raw_decode(text[index:])
            except json.JSONDecodeError:
                continue
            if text[index + end :].strip():
                continue
            return document
    return None


def hash_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as file:
        for chunk in iter(lambda: file.read(65536), b""):
            digest.update(chunk)
    return digest.hexdigest()


def frontend_lockfile_hash() -> str:
    return hash_file(FRONTEND_DIR / "pnpm-lock.yaml")


def frontend_bin_path(name: str) -> Path:
    suffix = ".cmd" if os.name == "nt" else ""
    return FRONTEND_DIR / "node_modules" / ".bin" / f"{name}{suffix}"


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
        frontend_bin_path("vite"),
        frontend_bin_path("vitest"),
        frontend_bin_path("eslint"),
        frontend_bin_path("vue-tsc"),
        frontend_bin_path("tsc"),
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

    run_pnpm(["install", "--frozen-lockfile", "--config.confirmModulesPurge=false", "--reporter=append-only"])
    write_frontend_deps_state(lockfile_hash)
    frontend_deps_installed = True


def backend_unit() -> None:
    run(["go", "test", "-tags=unit", "./..."], cwd=BACKEND_DIR)


def backend_integration() -> None:
    log_path = REPO_ROOT / ".cache" / "verify-ci" / "logs" / "backend-integration.log"
    try:
        run_streaming(
            ["go", "test", "-count=1", "-v", "-tags=integration", "./..."],
            cwd=BACKEND_DIR,
            log_path=log_path,
        )
    except subprocess.CalledProcessError as exc:
        emit_backend_integration_failure_summary(exc.output or "", log_path)
        raise


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
        pnpm_base_command()
        + [
            "audit",
            f"--registry={DEFAULT_NPM_AUDIT_REGISTRY}",
            "--prod",
            "--audit-level=high",
            "--json",
        ],
        cwd=FRONTEND_DIR,
        capture_output=True,
        check=False,
    )
    audit_document = extract_json_document(audit_result.stdout, audit_result.stderr)
    if audit_document is None:
        raise RuntimeError(
            "pnpm audit did not return valid JSON output.\n"
            f"stdout:\n{audit_result.stdout}\n"
            f"stderr:\n{audit_result.stderr}"
        )
    audit_path.write_text(json.dumps(audit_document, indent=2, sort_keys=True) + "\n", encoding="utf-8")
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


def wait_for_container_http(
    docker: str,
    container_name: str,
    *,
    path: str = "/health",
    timeout_seconds: int = 45,
) -> None:
    deadline = time.time() + timeout_seconds
    last_error = "request not attempted"
    while time.time() < deadline:
        result = run(
            [
                docker,
                "exec",
                container_name,
                "curl",
                "-fsS",
                f"http://127.0.0.1:8080{path}",
            ],
            cwd=REPO_ROOT,
            capture_output=True,
            check=False,
        )
        if result.returncode == 0:
            return
        combined = "\n".join(part for part in (result.stdout.strip(), result.stderr.strip()) if part).strip()
        if combined:
            last_error = combined
        time.sleep(1)
    raise RuntimeError(f"timeout waiting for {container_name}{path}: {last_error}")


def wait_for_docker_port(docker: str, container_name: str, container_port: str, *, timeout_seconds: int = 15) -> str:
    deadline = time.time() + timeout_seconds
    last_output = "docker port returned no output"
    while time.time() < deadline:
        result = run([docker, "port", container_name, container_port], cwd=REPO_ROOT, capture_output=True, check=False)
        port_output = result.stdout.strip()
        if port_output:
            return port_output.splitlines()[0].rsplit(":", 1)[-1].strip()
        combined = "\n".join(part for part in (result.stdout.strip(), result.stderr.strip()) if part).strip()
        if combined:
            last_output = combined
        time.sleep(1)
    raise RuntimeError(f"docker did not expose port {container_port} within {timeout_seconds}s: {last_output}")


def docker_smoke() -> None:
    docker = shutil.which("docker")
    if not docker:
        raise RuntimeError("docker is not available on PATH")

    frontend_build()

    smoke_root = REPO_ROOT / ".cache" / "verify-ci" / "docker-smoke"
    context_dir = smoke_root / "context"
    logs_dir = REPO_ROOT / ".cache" / "verify-ci" / "logs"
    if smoke_root.exists():
        shutil.rmtree(smoke_root)
    (context_dir / "deploy").mkdir(parents=True, exist_ok=True)
    logs_dir.mkdir(parents=True, exist_ok=True)

    docker_build_log = logs_dir / "docker-smoke-build.log"
    docker_container_log = logs_dir / "docker-smoke-container.log"
    for path in (docker_build_log, docker_container_log):
        if path.exists():
            path.unlink()

    linux_binary = context_dir / "sub2api"
    build_env = os.environ.copy()
    build_env.update({"GOOS": "linux", "GOARCH": "amd64", "CGO_ENABLED": "0"})
    run(["go", "build", "-tags=embed", "-o", str(linux_binary), "./cmd/server"], cwd=BACKEND_DIR, env=build_env)

    shutil.copy2(REPO_ROOT / "Dockerfile.goreleaser", context_dir / "Dockerfile.goreleaser")
    shutil.copy2(REPO_ROOT / "deploy" / "docker-entrypoint.sh", context_dir / "deploy" / "docker-entrypoint.sh")

    tag = f"sub2api-ci-smoke:{int(time.time())}"
    container_name = f"sub2api-ci-smoke-{int(time.time())}"
    run_streaming(
        [docker, "build", "-f", str(context_dir / "Dockerfile.goreleaser"), "-t", tag, str(context_dir)],
        cwd=REPO_ROOT,
        log_path=docker_build_log,
    )

    container_started = False
    try:
        run([docker, "run", "-d", "-P", "--name", container_name, tag], cwd=REPO_ROOT)
        container_started = True
        host_port = wait_for_docker_port(docker, container_name, "8080/tcp")
        info(f"docker exposed 8080/tcp on host port {host_port}")
        wait_for_container_http(docker, container_name)
    except Exception:
        if container_started:
            logs = run([docker, "logs", container_name], cwd=REPO_ROOT, capture_output=True, check=False)
            if logs.stdout.strip():
                docker_container_log.write_text(logs.stdout, encoding="utf-8")
                info("docker smoke container stdout/stderr:")
                for line in logs.stdout.splitlines()[-80:]:
                    info(f"[container] {line}")
        raise
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
    info("release preflight runs release-specific smoke checks; standard CI covers backend/frontend/security")
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
