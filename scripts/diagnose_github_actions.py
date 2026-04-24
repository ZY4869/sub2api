#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
import os
import re
import subprocess
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from datetime import datetime, timedelta, timezone
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[1]
USER_AGENT = "sub2api-github-actions-diagnose"
GITHUB_ACCEPT = "application/vnd.github+json"


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


def open_url(url: str, token: str | None, *, accept: str = GITHUB_ACCEPT):
    request = urllib.request.Request(
        url,
        headers={
            "Accept": accept,
            "User-Agent": USER_AGENT,
            **({"Authorization": f"Bearer {token}"} if token else {}),
        },
    )
    try:
        return urllib.request.urlopen(request, timeout=30)
    except urllib.error.HTTPError as exc:
        body = exc.read().decode("utf-8", errors="replace")
        raise RuntimeError(f"GitHub API request failed: {exc.code} {body}") from exc


def api_get(url: str, token: str | None) -> dict | list:
    with open_url(url, token) as response:
        return json.loads(response.read().decode("utf-8"))


def text_get(url: str, token: str | None, *, accept: str = "text/plain") -> str:
    with open_url(url, token, accept=accept) as response:
        return response.read().decode("utf-8", errors="replace")


def parse_github_time(value: str | None) -> datetime | None:
    if not value:
        return None
    try:
        return datetime.strptime(value, "%Y-%m-%dT%H:%M:%SZ").replace(tzinfo=timezone.utc)
    except ValueError:
        return None


def detect_branch_from_tag(tag: str | None, branch: str | None) -> str | None:
    if branch:
        return branch
    if not tag:
        return None
    return tag.removeprefix("refs/tags/").strip()


def list_runs(
    repo: str,
    token: str | None,
    *,
    workflow: str | None,
    branch: str | None,
    event: str | None,
    status: str | None,
    per_page: int = 30,
) -> list[dict]:
    params: dict[str, str | int] = {"per_page": per_page}
    if status:
        params["status"] = status
    if branch:
        params["branch"] = branch
    if event:
        params["event"] = event
    payload = api_get(
        f"https://api.github.com/repos/{repo}/actions/runs?{urllib.parse.urlencode(params)}",
        token,
    )
    runs = payload.get("workflow_runs", [])
    if workflow:
        runs = [run for run in runs if run.get("name") == workflow]
    return runs


def choose_run(
    repo: str,
    token: str | None,
    workflow: str | None,
    branch: str | None,
    event: str | None,
    *,
    require_failure: bool,
) -> dict:
    runs = list_runs(repo, token, workflow=workflow, branch=branch, event=event, status="completed")
    for run in runs:
        if require_failure and run.get("conclusion") in {"success", "skipped"}:
            continue
        return run
    if require_failure:
        raise RuntimeError("no failed workflow run matched the requested filters")
    raise RuntimeError("no completed workflow run matched the requested filters")


def wait_for_run(
    repo: str,
    token: str | None,
    workflow: str | None,
    branch: str | None,
    event: str | None,
    *,
    interval_seconds: int,
    timeout_seconds: int,
) -> dict:
    start_time = datetime.now(timezone.utc)
    stale_cutoff = start_time - timedelta(seconds=max(interval_seconds * 2, 30))
    deadline = time.monotonic() + timeout_seconds
    selected_run_id: int | None = None
    last_status: tuple[str | None, str | None] | None = None
    announced_wait = False

    while time.monotonic() < deadline:
        if selected_run_id is None:
            runs = list_runs(repo, token, workflow=workflow, branch=branch, event=event, status=None)
            for candidate in runs:
                created_at = parse_github_time(candidate.get("created_at"))
                if candidate.get("status") == "completed" and created_at and created_at < stale_cutoff:
                    continue
                selected_run_id = candidate.get("id")
                print(
                    f"Matched run {selected_run_id}: {candidate.get('name')} {candidate.get('html_url')}",
                    flush=True,
                )
                break
            if selected_run_id is None:
                if not announced_wait:
                    print("Waiting for a matching workflow run to appear...", flush=True)
                    announced_wait = True
                time.sleep(interval_seconds)
                continue

        run = api_get(f"https://api.github.com/repos/{repo}/actions/runs/{selected_run_id}", token)
        status = (run.get("status"), run.get("conclusion"))
        if status != last_status:
            print(
                f"Run {selected_run_id} status={run.get('status')} conclusion={run.get('conclusion')}",
                flush=True,
            )
            last_status = status
        if run.get("status") == "completed":
            return run
        time.sleep(interval_seconds)

    raise RuntimeError(f"timed out after {timeout_seconds}s waiting for workflow run")


def recommended_command(job_name: str, failed_steps: list[str]) -> str:
    context = f"{job_name} {' '.join(failed_steps)}"
    text = context.lower()
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


def fetch_failed_jobs(repo: str, token: str | None, run_id: int) -> list[dict]:
    jobs_payload = api_get(f"https://api.github.com/repos/{repo}/actions/runs/{run_id}/jobs?per_page=100", token)
    return [job for job in jobs_payload.get("jobs", []) if job.get("conclusion") not in {"success", "skipped"}]


def fetch_annotations(token: str | None, check_run_url: str | None) -> list[dict]:
    if not check_run_url:
        return []
    data = api_get(f"{check_run_url}/annotations?per_page=100", token)
    return data if isinstance(data, list) else []


def fetch_job_log_tail(repo: str, token: str | None, job_id: int, max_lines: int) -> list[str]:
    if not token or max_lines <= 0:
        return []
    log_text = text_get(f"https://api.github.com/repos/{repo}/actions/jobs/{job_id}/logs", token)
    lines = [line.rstrip() for line in log_text.splitlines() if line.strip()]
    return lines[-max_lines:]


def print_run_summary(run: dict) -> None:
    print(f"Workflow: {run.get('name')}")
    print(f"Run ID: {run.get('id')}")
    print(f"Run URL: {run.get('html_url')}")
    print(f"Branch: {run.get('head_branch')}")
    print(f"Event: {run.get('event')}")
    print(f"Status: {run.get('status')}")
    print(f"Conclusion: {run.get('conclusion')}")
    print()


def print_failed_jobs(repo: str, token: str | None, failed_jobs: list[dict], *, log_lines: int) -> None:
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

        annotations = fetch_annotations(token, job.get("check_run_url"))
        if annotations:
            print("  Annotations:")
            for annotation in annotations[:20]:
                path = annotation.get("path", "<unknown>")
                line = annotation.get("start_line") or annotation.get("line") or 0
                message = annotation.get("message", "").strip().replace("\n", " ")
                print(f"  - {path}:{line} {message}")
            if len(annotations) > 20:
                print(f"  - ... {len(annotations) - 20} more annotations omitted")

        try:
            log_tail = fetch_job_log_tail(repo, token, int(job.get("id", 0)), log_lines)
        except RuntimeError as exc:
            log_tail = []
            print(f"  Log tail unavailable: {exc}")
        if log_tail:
            print("  Log tail:")
            for line in log_tail:
                print(f"  | {line}")
        print()


def main() -> int:
    parser = argparse.ArgumentParser(description="REST-first GitHub Actions failure diagnosis")
    parser.add_argument("--workflow", help="workflow name to match, such as Release")
    parser.add_argument("--branch", help="branch name to match")
    parser.add_argument("--tag", help="tag name to match, such as v0.1.280")
    parser.add_argument("--event", help="workflow event to match, such as push")
    parser.add_argument("--run-id", type=int, help="specific workflow run id")
    parser.add_argument("--repo", help="GitHub repo in owner/name form")
    parser.add_argument("--token", help="GitHub token; defaults to GITHUB_TOKEN if present")
    parser.add_argument("--watch", action="store_true", help="wait for the matching workflow run to appear and finish")
    parser.add_argument("--interval", type=int, default=15, help="poll interval in seconds while watching")
    parser.add_argument("--timeout", type=int, default=1800, help="watch timeout in seconds")
    parser.add_argument("--log-lines", type=int, default=12, help="job log tail lines to print when GITHUB_TOKEN is set")
    args = parser.parse_args()

    repo = args.repo or detect_repo()
    token = args.token or os.environ.get("GITHUB_TOKEN")
    branch = detect_branch_from_tag(args.tag, args.branch)
    workflow = args.workflow or ("Release" if args.tag else None)
    event = args.event or ("push" if args.tag else None)

    if args.watch and args.run_id:
        raise RuntimeError("--watch and --run-id cannot be used together")

    if args.watch:
        print(
            f"Watching repo={repo} workflow={workflow or '*'} branch={branch or '*'} event={event or '*'}",
            flush=True,
        )
        run = wait_for_run(
            repo,
            token,
            workflow,
            branch,
            event,
            interval_seconds=max(5, args.interval),
            timeout_seconds=max(30, args.timeout),
        )
    elif args.run_id:
        run = api_get(f"https://api.github.com/repos/{repo}/actions/runs/{args.run_id}", token)
    else:
        explicit_selector = any([workflow, branch, event, args.tag])
        run = choose_run(
            repo,
            token,
            workflow,
            branch,
            event,
            require_failure=not explicit_selector,
        )

    print(f"Repository: {repo}")
    print_run_summary(run)

    failed_jobs = fetch_failed_jobs(repo, token, int(run["id"]))
    if not failed_jobs:
        print("No failed jobs were found in the selected run.")
        conclusion = run.get("conclusion")
        if args.watch and conclusion not in {"success", "skipped"}:
            return 1
        return 0

    print_failed_jobs(repo, token, failed_jobs, log_lines=max(0, args.log_lines))
    return 1 if args.watch else 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except KeyboardInterrupt:
        print("Interrupted.", file=sys.stderr)
        sys.exit(130)
    except RuntimeError as exc:
        print(f"error: {exc}", file=sys.stderr)
        sys.exit(1)
