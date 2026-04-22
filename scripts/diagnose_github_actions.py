#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
import os
import re
import subprocess
import sys
import urllib.error
import urllib.request
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[1]
USER_AGENT = "sub2api-github-actions-diagnose"


def detect_repo() -> str:
    if os.environ.get("GITHUB_REPOSITORY"):
        return os.environ["GITHUB_REPOSITORY"]

    remote = subprocess.run(
        ["git", "remote", "get-url", "origin"],
        cwd=str(REPO_ROOT),
        check=False,
        text=True,
        capture_output=True,
    ).stdout.strip()
    match = re.search(r"github\.com[:/](?P<owner>[^/]+)/(?P<repo>[^/.]+)", remote)
    if not match:
        raise RuntimeError("unable to detect GitHub repository from origin remote")
    return f"{match.group('owner')}/{match.group('repo')}"


def api_get(url: str, token: str | None) -> dict | list:
    request = urllib.request.Request(
        url,
        headers={
            "Accept": "application/vnd.github+json",
            "User-Agent": USER_AGENT,
            **({"Authorization": f"Bearer {token}"} if token else {}),
        },
    )
    try:
        with urllib.request.urlopen(request, timeout=30) as response:
            return json.loads(response.read().decode("utf-8"))
    except urllib.error.HTTPError as exc:
        body = exc.read().decode("utf-8", errors="replace")
        raise RuntimeError(f"GitHub API request failed: {exc.code} {body}") from exc


def choose_run(repo: str, token: str | None, workflow: str | None, branch: str | None) -> dict:
    params = ["per_page=30", "status=completed"]
    if branch:
        params.append(f"branch={branch}")
    payload = api_get(f"https://api.github.com/repos/{repo}/actions/runs?{'&'.join(params)}", token)
    runs = payload.get("workflow_runs", [])
    for run in runs:
        if workflow and run.get("name") != workflow:
            continue
        if run.get("conclusion") in {"success", "skipped"}:
            continue
        return run
    raise RuntimeError("no failed workflow run matched the requested filters")


def recommended_command(job_name: str, failed_steps: list[str]) -> str:
    context = f"{job_name} {' '.join(failed_steps)}"
    text = context.lower()
    tag_match = re.search(r"\bv\d+\.\d+\.\d+\b", context)
    if tag_match and ("verify" in text or "tag health" in text):
        return f"python scripts/tag_health.py verify --tag {tag_match.group(0)} --profile compile"
    if "validate-release-source" in text:
        return "go mod tidy -C backend && git diff --exit-code -- backend/cmd/server/VERSION frontend/package.json backend/go.mod backend/go.sum"
    if "release-preflight" in text or "release" in text:
        return "scripts/verify-ci.sh release-gate"
    if "integration" in text:
        return "scripts/verify-ci.sh backend-integration"
    if "unit" in text:
        return "scripts/verify-ci.sh backend-unit"
    if "lint" in text and "frontend" in text:
        return "scripts/verify-ci.sh frontend-lint"
    if "lint" in text:
        return "scripts/verify-ci.sh backend-lint"
    if "frontend" in text and "build" in text:
        return "scripts/verify-ci.sh frontend-build"
    if "frontend" in text and "test" in text:
        return "scripts/verify-ci.sh frontend-test"
    if "security" in text:
        return "scripts/verify-ci.sh security"
    if "docker" in text:
        return "scripts/verify-ci.sh docker-smoke"
    if "build" in text:
        return "scripts/verify-ci.sh backend-build"
    return "scripts/verify-ci.sh full"


def main() -> int:
    parser = argparse.ArgumentParser(description="REST-first GitHub Actions failure diagnosis")
    parser.add_argument("--workflow", help="workflow name to match, such as CI")
    parser.add_argument("--branch", help="branch name to match")
    parser.add_argument("--run-id", type=int, help="specific workflow run id")
    parser.add_argument("--repo", help="GitHub repo in owner/name form")
    parser.add_argument("--token", help="GitHub token; defaults to GITHUB_TOKEN if present")
    args = parser.parse_args()

    repo = args.repo or detect_repo()
    token = args.token or os.environ.get("GITHUB_TOKEN")
    run = (
        api_get(f"https://api.github.com/repos/{repo}/actions/runs/{args.run_id}", token)
        if args.run_id
        else choose_run(repo, token, args.workflow, args.branch)
    )

    jobs_payload = api_get(f"https://api.github.com/repos/{repo}/actions/runs/{run['id']}/jobs?per_page=100", token)
    failed_jobs = [job for job in jobs_payload.get("jobs", []) if job.get("conclusion") not in {"success", "skipped"}]

    print(f"Repository: {repo}")
    print(f"Workflow: {run.get('name')}")
    print(f"Run ID: {run.get('id')}")
    print(f"Run URL: {run.get('html_url')}")
    print(f"Branch: {run.get('head_branch')}")
    print(f"Conclusion: {run.get('conclusion')}")
    print()

    if not failed_jobs:
        print("No failed jobs were found in the selected run.")
        return 0

    print("Failed Jobs:")
    for job in failed_jobs:
        failed_steps = [
            step.get("name", "<unnamed step>")
            for step in job.get("steps", [])
            if step.get("conclusion") not in {None, "success", "skipped"}
        ]
        print(f"- {job.get('name')}: {job.get('conclusion')}")
        if failed_steps:
            for step in failed_steps:
                print(f"  Failed step: {step}")
        print(f"  Recommended local reproduce command: {recommended_command(job.get('name', ''), failed_steps)}")

        check_run_url = job.get("check_run_url")
        if not check_run_url:
            continue
        annotations = api_get(f"{check_run_url}/annotations?per_page=100", token)
        if not annotations:
            continue
        print("  Annotations:")
        for annotation in annotations[:20]:
            path = annotation.get("path", "<unknown>")
            line = annotation.get("start_line") or annotation.get("line") or 0
            message = annotation.get("message", "").strip().replace("\n", " ")
            print(f"  - {path}:{line} {message}")
        if len(annotations) > 20:
            print(f"  - ... {len(annotations) - 20} more annotations omitted")
        print()

    return 0


if __name__ == "__main__":
    sys.exit(main())
