#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
import os
import re
import shutil
import subprocess
import sys
import time
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[1]
REPORT_ROOT = REPO_ROOT / ".cache" / "tag-health" / "reports"
WORKTREE_ROOT = Path(
    os.environ.get("SUB2API_TAG_HEALTH_WORKTREE_ROOT", REPO_ROOT.parent / ".sub2api-tag-health-worktrees")
).resolve()
SEMVER_TAG_PATTERN = re.compile(r"^v\d+\.\d+\.\d+$")
VERIFY_SCRIPT = REPO_ROOT / "scripts" / "verify_ci.py"
PROFILES = {
    "compile": ["backend-build", "frontend-build"],
    "ci": ["backend-unit", "backend-build", "frontend-build", "frontend-test", "frontend-lint"],
    "release": [
        "backend-unit",
        "backend-integration",
        "backend-lint",
        "backend-build",
        "frontend-build",
        "frontend-test",
        "frontend-lint",
        "docker-smoke",
    ],
}


def info(message: str) -> None:
    print(f"[tag-health] {message}", flush=True)


def run(
    command: list[str],
    *,
    cwd: Path | None = None,
    env: dict[str, str] | None = None,
    capture_output: bool = False,
) -> subprocess.CompletedProcess[str]:
    info(f"$ {' '.join(command)}")
    return subprocess.run(
        command,
        cwd=str(cwd) if cwd else None,
        env=env,
        check=True,
        text=True,
        encoding="utf-8",
        errors="replace",
        capture_output=capture_output,
    )


def list_release_tags(pattern: str, max_tags: int) -> list[str]:
    result = run(
        ["git", "tag", "--list", pattern, "--sort=-version:refname"],
        cwd=REPO_ROOT,
        capture_output=True,
    )
    tags = [line.strip() for line in result.stdout.splitlines() if line.strip()]
    tags = [tag for tag in tags if SEMVER_TAG_PATTERN.fullmatch(tag)]
    if max_tags > 0:
        return tags[:max_tags]
    return tags


def matrix_payload(tags: list[str]) -> str:
    return json.dumps({"include": [{"tag": tag} for tag in tags]}, separators=(",", ":"))


def sanitize_tag(tag: str) -> str:
    return re.sub(r"[^0-9A-Za-z._-]+", "_", tag)


def report_path_for(tag: str) -> Path:
    return REPORT_ROOT / f"{sanitize_tag(tag)}.json"


def write_report(tag: str, profile: str, results: list[dict[str, object]]) -> Path:
    REPORT_ROOT.mkdir(parents=True, exist_ok=True)
    report_path = report_path_for(tag)
    report_path.write_text(
        json.dumps({"tag": tag, "profile": profile, "results": results}, indent=2, ensure_ascii=False) + "\n",
        encoding="utf-8",
    )
    return report_path


def cleanup_worktree(worktree_path: Path) -> None:
    subprocess.run(
        ["git", "worktree", "remove", "--force", str(worktree_path)],
        cwd=str(REPO_ROOT),
        check=False,
        text=True,
        encoding="utf-8",
        errors="replace",
        capture_output=True,
    )
    shutil.rmtree(worktree_path, ignore_errors=True)
    subprocess.run(
        ["git", "worktree", "prune"],
        cwd=str(REPO_ROOT),
        check=False,
        text=True,
        encoding="utf-8",
        errors="replace",
        capture_output=True,
    )


def create_worktree(tag: str) -> Path:
    WORKTREE_ROOT.mkdir(parents=True, exist_ok=True)
    worktree_path = WORKTREE_ROOT / sanitize_tag(tag)
    cleanup_worktree(worktree_path)
    run(["git", "worktree", "add", "--force", "--detach", str(worktree_path), tag], cwd=REPO_ROOT)
    return worktree_path


def verify_tag(tag: str, profile: str, keep_worktree: bool) -> int:
    modes = PROFILES[profile]
    worktree_path = create_worktree(tag)
    results: list[dict[str, object]] = []

    try:
        for mode in modes:
            started_at = time.time()
            info(f"verifying {tag} with mode {mode}")
            env = os.environ.copy()
            env["SUB2API_VERIFY_REPO_ROOT"] = str(worktree_path)

            try:
                run([sys.executable, str(VERIFY_SCRIPT), mode], cwd=REPO_ROOT, env=env)
                results.append(
                    {
                        "mode": mode,
                        "status": "success",
                        "duration_seconds": round(time.time() - started_at, 2),
                    }
                )
            except subprocess.CalledProcessError as exc:
                results.append(
                    {
                        "mode": mode,
                        "status": "failed",
                        "duration_seconds": round(time.time() - started_at, 2),
                        "returncode": exc.returncode,
                    }
                )
                report_path = write_report(tag, profile, results)
                info(f"tag {tag} failed in mode {mode}; report written to {report_path}")
                return exc.returncode

        report_path = write_report(tag, profile, results)
        info(f"tag {tag} passed profile {profile}; report written to {report_path}")
        return 0
    finally:
        if not keep_worktree:
            cleanup_worktree(worktree_path)


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Enumerate and verify release tag health")
    subparsers = parser.add_subparsers(dest="command", required=True)

    list_parser = subparsers.add_parser("list", help="List semver release tags")
    list_parser.add_argument("--pattern", default="v*")
    list_parser.add_argument("--max-tags", type=int, default=0)
    list_parser.add_argument("--format", choices=("lines", "json", "matrix", "count"), default="lines")

    verify_parser = subparsers.add_parser("verify", help="Verify one tag using current branch tooling")
    verify_parser.add_argument("--tag", required=True)
    verify_parser.add_argument("--profile", choices=sorted(PROFILES), default="compile")
    verify_parser.add_argument("--keep-worktree", action="store_true")

    return parser


def main() -> int:
    parser = build_parser()
    args = parser.parse_args()

    if args.command == "list":
        tags = list_release_tags(args.pattern, args.max_tags)
        if args.format == "count":
            print(len(tags))
        elif args.format == "json":
            print(json.dumps(tags, ensure_ascii=False))
        elif args.format == "matrix":
            print(matrix_payload(tags))
        else:
            for tag in tags:
                print(tag)
        return 0

    if args.command == "verify":
        if not SEMVER_TAG_PATTERN.fullmatch(args.tag):
            parser.error(f"tag must match vX.Y.Z, got: {args.tag}")
        return verify_tag(args.tag, args.profile, args.keep_worktree)

    parser.error(f"unsupported command: {args.command}")
    return 1


if __name__ == "__main__":
    sys.exit(main())
