#!/usr/bin/env bash

# codex-check.sh
#
# 受控检查 Codex 账号/配额，并分析一次 Responses WebSocket 请求中的：
#   requested model -> response.completed.response.model -> websocket_timing.engine_ids
#
# 默认模式会调用本机 codex CLI。--mock 模式完全离线，只生成脱敏测试事件，
# 用来验证解析和 Sol/Luna 不一致的判定逻辑。脚本只在进程内读取认证，绝不打印或保存访问令牌。

set -u
set -o pipefail

MODEL="gpt-5.6-sol"
DO_QUOTA=1
DO_ROUTE=1
MOCK=0
MOCK_ENGINE="luna"
KEEP_LOGS=0
SHOW_EMAIL=0
WS_MODE=0
WS_ENDPOINT="/v1/responses"
ROUNDS=1
SAME_CONNECTION_ROUNDS=1
PYTHON_BIN=""
EVENTS_FILE=""

usage() {
  cat <<'USAGE'
Usage: codex-check.sh [options]

Options:
  --quota-only          Only show account/quota; makes no model request
  --route-only          Only run the routing test
  --mock                Run an offline parser test; never calls codex or OpenAI
  --mock-engine FAMILY  Mock engine family: sol, luna, mixed, other (default: luna)
  --events FILE         Analyze an existing JSONL/SSE event capture; no network request
  --model MODEL         Model to request (default: gpt-5.6-sol)
  --ws                  Use a direct WebSocket client instead of codex exec
  --endpoint PATH       WebSocket endpoint (/v1/responses, /responses, or /backend-api/codex/responses)
  --rounds N            Number of independent connections (default: 1)
  --same-connection-rounds N  response.create messages per connection (default: 1)
  --show-email          Show the full account email instead of masking it
  --keep-logs           Keep only redacted JSONL/summary logs and print their path
  -h, --help            Show this help

Default behavior: account/quota + one tiny low-reasoning routing request.

The real route test requires a logged-in `codex` CLI. The offline command is:
  ./scripts/codex-check.sh --mock --route-only

To analyze a capture exported by a WebSocket client without sending another request:
  ./scripts/codex-check.sh --events ./events.jsonl --model gpt-5.6-sol
USAGE
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --quota-only)
      DO_QUOTA=1
      DO_ROUTE=0
      ;;
    --route-only)
      DO_QUOTA=0
      DO_ROUTE=1
      ;;
    --mock)
      MOCK=1
      ;;
    --mock-engine)
      shift
      if [ "$#" -eq 0 ]; then
        echo "Error: --mock-engine requires a value." >&2
        exit 2
      fi
      MOCK_ENGINE="$1"
      ;;
    --events)
      shift
      if [ "$#" -eq 0 ]; then
        echo "Error: --events requires a file path." >&2
        exit 2
      fi
      EVENTS_FILE="$1"
      DO_QUOTA=0
      DO_ROUTE=1
      ;;
    --model)
      shift
      if [ "$#" -eq 0 ]; then
        echo "Error: --model requires a value." >&2
        exit 2
      fi
      MODEL="$1"
      ;;
    --ws)
      WS_MODE=1
      DO_ROUTE=1
      ;;
    --endpoint)
      shift
      [ "$#" -gt 0 ] || { echo "Error: --endpoint requires a path." >&2; exit 2; }
      WS_ENDPOINT="$1"
      ;;
    --rounds)
      shift
      [ "$#" -gt 0 ] && [[ "$1" =~ ^[1-9][0-9]*$ ]] || { echo "Error: --rounds must be a positive integer." >&2; exit 2; }
      ROUNDS="$1"
      ;;
    --same-connection-rounds)
      shift
      [ "$#" -gt 0 ] && [[ "$1" =~ ^[1-9][0-9]*$ ]] || { echo "Error: --same-connection-rounds must be a positive integer." >&2; exit 2; }
      SAME_CONNECTION_ROUNDS="$1"
      ;;
    --show-email)
      SHOW_EMAIL=1
      ;;
    --keep-logs)
      KEEP_LOGS=1
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
  shift
done

case "$WS_ENDPOINT" in
  /v1/responses|/responses|/backend-api/codex/responses)
    ;;
  *)
    echo "Error: --endpoint must be /v1/responses, /responses, or /backend-api/codex/responses." >&2
    exit 2
    ;;
esac

find_python() {
  if command -v python3 >/dev/null 2>&1; then
    command -v python3
    return 0
  fi
  if command -v python >/dev/null 2>&1; then
    command -v python
    return 0
  fi
  return 1
}

if [ "$DO_QUOTA" -eq 1 ] || [ "$DO_ROUTE" -eq 1 ]; then
  PYTHON_BIN="$(find_python || true)"
fi

if [ "$DO_QUOTA" -eq 1 ] && [ -z "$PYTHON_BIN" ]; then
  echo "Error: python3 or python is required for account/quota inspection." >&2
  exit 1
fi

if [ "$WS_MODE" -eq 0 ] && [ "$MOCK" -eq 0 ] && [ "$DO_ROUTE" -eq 1 ] && ! command -v codex >/dev/null 2>&1; then
  if [ -z "$EVENTS_FILE" ]; then
    echo "Error: codex was not found in PATH. Use --mock or --events for offline analysis." >&2
    exit 1
  fi
fi

if [ -n "$EVENTS_FILE" ] && [ ! -r "$EVENTS_FILE" ]; then
  echo "Error: event capture is not readable: $EVENTS_FILE" >&2
  exit 1
fi

printf 'Requested model: %s\n' "$MODEL"
if [ "$WS_MODE" -eq 1 ]; then
  printf 'WebSocket endpoint: %s (connections=%s, rounds/connection=%s)\n' "$WS_ENDPOINT" "$ROUNDS" "$SAME_CONNECTION_ROUNDS"
fi
if [ "$MOCK" -eq 1 ]; then
  printf 'Mode: offline mock (no account, token, or network request)\n'
elif [ -n "$EVENTS_FILE" ]; then
  printf 'Mode: offline event analysis (no account, token, or network request)\n'
elif command -v codex >/dev/null 2>&1; then
  printf 'Codex: %s\n' "$(codex --version 2>/dev/null || echo unknown)"
fi

query_account_and_quota() {
  SHOW_EMAIL="$SHOW_EMAIL" "$PYTHON_BIN" - <<'PY'
import datetime as dt
import json
import os
import queue
import subprocess
import sys
import threading
import time

SHOW_EMAIL = os.environ.get("SHOW_EMAIL") == "1"


def fail(msg):
    print(f"Account/quota: unable to read ({msg})")
    sys.exit(0)


def mask_email(email):
    if SHOW_EMAIL or not email or "@" not in email:
        return email or "-"
    name, domain = email.split("@", 1)
    visible = name[:1] if len(name) <= 2 else name[:2]
    return f"{visible}***@{domain}"


def local_reset(ts):
    if ts is None:
        return "unknown"
    try:
        value = float(ts)
        return dt.datetime.fromtimestamp(value, tz=dt.timezone.utc).astimezone().strftime("%Y-%m-%d %H:%M:%S %Z")
    except Exception:
        return str(ts)


def duration_label(minutes):
    if minutes is None:
        return "window"
    try:
        m = int(minutes)
    except Exception:
        return f"{minutes} min"
    labels = {300: "5h", 1440: "daily", 10080: "weekly"}
    if m in labels:
        return labels[m]
    if 40320 <= m <= 44640:
        return "monthly"
    if m % 1440 == 0:
        return f"{m // 1440}d"
    if m % 60 == 0:
        return f"{m // 60}h"
    return f"{m}m"


def send(proc, obj):
    proc.stdin.write(json.dumps(obj, separators=(",", ":")) + "\n")
    proc.stdin.flush()


def start_reader(proc):
    """Read the app-server pipe in a thread; this works on POSIX and Windows."""
    messages = queue.Queue()

    def read_lines():
        try:
            for line in proc.stdout:
                line = line.strip()
                if not line:
                    continue
                try:
                    messages.put(json.loads(line))
                except json.JSONDecodeError:
                    continue
        finally:
            messages.put(None)

    threading.Thread(target=read_lines, daemon=True).start()
    return messages


def read_id(messages, wanted_id, timeout=15.0):
    deadline = time.monotonic() + timeout
    while True:
        remaining = deadline - time.monotonic()
        if remaining <= 0:
            raise TimeoutError(f"timed out waiting for response id {wanted_id}")
        try:
            msg = messages.get(timeout=remaining)
        except queue.Empty:
            raise TimeoutError(f"timed out waiting for response id {wanted_id}")
        if msg is None:
            raise RuntimeError("app-server closed stdout")
        if msg.get("id") != wanted_id:
            continue
        if "error" in msg:
            raise RuntimeError(f"JSON-RPC error: {msg['error']}")
        return msg.get("result") or {}


proc = None
try:
    proc = subprocess.Popen(
        ["codex", "app-server", "--stdio"],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.DEVNULL,
        text=True,
        bufsize=1,
    )
    messages = start_reader(proc)
    send(proc, {"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {
        "clientInfo": {"name": "codex-check", "version": "2.0"},
        "capabilities": {"experimentalApi": True},
    }})
    read_id(messages, 1)
    send(proc, {"jsonrpc": "2.0", "method": "notifications/initialized"})
    send(proc, {"jsonrpc": "2.0", "id": 2, "method": "account/read", "params": {"refreshToken": False}})
    account_result = read_id(messages, 2)
    send(proc, {"jsonrpc": "2.0", "id": 3, "method": "account/rateLimits/read"})
    limits_result = read_id(messages, 3)
except FileNotFoundError:
    fail("codex app-server is unavailable")
except Exception as exc:
    fail(str(exc))
finally:
    if proc is not None:
        try:
            proc.terminate()
            proc.wait(timeout=2)
        except Exception:
            try:
                proc.kill()
            except Exception:
                pass

account = account_result.get("account")
print("\n[Account]")
if not account:
    print("  Login: not detected")
else:
    kind = account.get("type", "unknown")
    print(f"  Auth: {kind}")
    if kind == "chatgpt":
        print(f"  Email: {mask_email(account.get('email'))}")
        print(f"  Plan: {account.get('planType') or '-'}")
print(f"  Requires OpenAI auth: {account_result.get('requiresOpenaiAuth')}")

by_id = limits_result.get("rateLimitsByLimitId") or {}
if not by_id:
    fallback = limits_result.get("rateLimits")
    if fallback:
        key = fallback.get("limitId") or "default"
        by_id = {key: fallback}

print("\n[Quota]")
if not by_id:
    print("  No rate-limit snapshot returned.")
else:
    keys = sorted(by_id.keys(), key=lambda k: (k != "codex", str(k)))
    for key in keys:
        snap = by_id.get(key) or {}
        name = snap.get("limitName") or key
        suffix = []
        if snap.get("planType"):
            suffix.append(f"plan={snap['planType']}")
        if snap.get("normalModelSlug"):
            suffix.append(f"model={snap['normalModelSlug']}")
        print(f"  {name}" + (f" ({', '.join(suffix)})" if suffix else ""))
        windows = [(slot, snap.get(slot)) for slot in ("primary", "secondary") if snap.get(slot)]
        if not windows:
            print("    window: unavailable")
        for slot, window in windows:
            used = window.get("usedPercent")
            duration = duration_label(window.get("windowDurationMins"))
            try:
                remaining = max(0.0, min(100.0, 100.0 - float(used)))
                used_text = f"{float(used):g}% used / {remaining:g}% left"
            except Exception:
                used_text = f"used={used}"
            print(f"    {duration:>7}: {used_text}; resets {local_reset(window.get('resetsAt'))}")
        if snap.get("rateLimitReachedType"):
            print(f"    reached: {snap['rateLimitReachedType']}")
        if snap.get("spendControlReached") is True:
            print("    spend control: reached")
        credits = snap.get("credits")
        if credits:
            if credits.get("unlimited") is True:
                print("    credits: unlimited")
            elif credits.get("balance") is not None:
                print(f"    credits balance: {credits['balance']}")
PY
}

make_mock_events() {
  local path="$1"
  case "$MODEL" in
    ""|*[!A-Za-z0-9._:-]*)
      echo "Error: --model contains characters unsafe for mock JSON output." >&2
      return 2
      ;;
  esac
  case "$MOCK_ENGINE" in
    sol)
      engine='gpt56sol-codex-abc123'
      ;;
    luna)
      engine='gpt56lun-codex-def456'
      ;;
    mixed)
      engine='gpt56sol-codex-abc123 gpt56lun-codex-def456'
      ;;
    other)
      engine='gpt5x-codex-unknown'
      ;;
    *)
      echo "Error: --mock-engine must be sol, luna, mixed, or other." >&2
      return 2
      ;;
  esac
  {
    printf '%s\n' "{\"type\":\"response.created\",\"response\":{\"id\":\"resp_mock_123\",\"model\":\"$MODEL\"}}"
    if [ "$MOCK_ENGINE" = "mixed" ]; then
      printf '%s\n' "{\"type\":\"responsesapi.websocket_timing\",\"timing_metrics\":{\"engine_ids\":[\"gpt56sol-codex-abc123\",\"gpt56lun-codex-def456\"]}}"
    else
      printf '%s\n' "{\"type\":\"responsesapi.websocket_timing\",\"timing_metrics\":{\"engine_ids\":[\"$engine\"]}}"
    fi
    printf '%s\n' "{\"type\":\"response.completed\",\"response\":{\"id\":\"resp_mock_123\",\"model\":\"$MODEL\",\"output\":[{\"type\":\"message\",\"content\":[{\"type\":\"output_text\",\"text\":\"OK\"}]}]}}"
  } >"$path"
}

analyze_events() {
  local events_file="$1"
  local summary_file="$2"
  local redacted_events_file="$3"
  "$PYTHON_BIN" - "$events_file" "$summary_file" "$redacted_events_file" "$MODEL" <<'PY'
import hashlib
import json
import re
import sys
from datetime import datetime, timezone
from pathlib import Path

events_path, summary_path, redacted_events_path, requested_model = sys.argv[1:]
text = Path(events_path).read_text(encoding="utf-8", errors="replace")
engine_pattern = re.compile(r"gpt56(?:sol|lun)-codex-[A-Za-z0-9._-]+", re.I)
completed_model_pattern = re.compile(r"response(?:\.completed)?\.response\.model\s*[:=]\s*[\"']?([A-Za-z0-9._-]+)", re.I)
families = {}
redacted_engine_ids = set()
engine_matches_seen = set()
records = []
json_objects = []
textual_completed_models = []
textual_timing_count = 0


def add_engine(value):
    values = value if isinstance(value, list) else [value]
    for item in values:
        if not isinstance(item, str):
            continue
        for match in engine_pattern.findall(item):
            normalized = match.lower()
            if normalized in engine_matches_seen:
                continue
            engine_matches_seen.add(normalized)
            family = "sol" if "gpt56sol-codex-" in match.lower() else "luna"
            families[family] = families.get(family, 0) + 1
            prefix = "gpt56sol" if family == "sol" else "gpt56lun"
            redacted_engine_ids.add(f"{prefix}-codex-<redacted>")


def walk(value):
    if isinstance(value, dict):
        event_type = value.get("type")
        if isinstance(event_type, str):
            event_type = event_type.strip()
        response = value.get("response")
        response_model = response.get("model") if isinstance(response, dict) else None
        response_id = response.get("id") if isinstance(response, dict) else value.get("id")
        if event_type:
            record = {
                "sequence": len(records) + 1,
                "received_at": datetime.now(timezone.utc).isoformat(timespec="milliseconds"),
                "type": event_type,
            }
            if isinstance(response_id, str) and response_id:
                record["response_id"] = hashlib.sha256(response_id.encode()).hexdigest()[:12]
            if isinstance(value.get("model"), str):
                record["model"] = value["model"]
            if isinstance(response_model, str):
                record["response_model"] = response_model
            timing = value.get("timing_metrics")
            if isinstance(timing, dict):
                add_engine(timing.get("engine_ids"))
                if redacted_engine_ids:
                    record["engine_ids"] = sorted(redacted_engine_ids)
            records.append(record)


for line in text.splitlines():
    stripped = line.strip()
    if not stripped:
        continue
    if "responsesapi.websocket_timing" in stripped:
        textual_timing_count += 1
    textual_model = completed_model_pattern.search(stripped)
    if textual_model:
        textual_completed_models.append(textual_model.group(1))
    # Codex/Responses 输出有时使用 SSE 风格的 `data: {...}` 行；只解析
    # data 负载，不把 event 名或原始行写入保留日志。
    candidate = stripped[5:].strip() if stripped.startswith("data:") else stripped
    try:
        obj = json.loads(candidate)
        json_objects.append(obj)
        walk(obj)
    except json.JSONDecodeError:
        for match in engine_pattern.findall(line):
            normalized = match.lower()
            if normalized in engine_matches_seen:
                continue
            engine_matches_seen.add(normalized)
            family = "sol" if "gpt56sol-codex-" in match.lower() else "luna"
            families[family] = families.get(family, 0) + 1
            prefix = "gpt56sol" if family == "sol" else "gpt56lun"
            redacted_engine_ids.add(f"{prefix}-codex-<redacted>")

completed_models = []
for record in records:
    if record.get("type") in ("response.completed", "response.done") and record.get("response_model"):
        completed_models.append(record["response_model"])
completed_models.extend(textual_completed_models)

unique_models = sorted(set(completed_models))
engine_families = sorted(families)
if engine_families == ["sol"]:
    result = "SOL"
elif engine_families == ["luna"]:
    result = "LUNA"
elif set(engine_families) == {"sol", "luna"}:
    result = "MIXED"
else:
    result = "OTHER/UNKNOWN"

summary = {
    "requested_model": requested_model,
    "completed_response_models": unique_models,
    "engine_families": engine_families,
    "engine_ids": sorted(redacted_engine_ids),
    "engine_id_count": sum(families.values()),
    "timing_event_count": max(
        sum(1 for record in records if record.get("type") == "responsesapi.websocket_timing"),
        textual_timing_count,
    ),
    "event_count": len(records),
    "result": result,
    "model_name_matches": bool(unique_models) and unique_models == [requested_model],
    "model_engine_mismatch": bool(unique_models) and unique_models == [requested_model] and result in ("LUNA", "MIXED"),
    "events": records,
}
# Replace the cumulative family list with the final set for stable records.
for record in records:
    if record.get("type") == "responsesapi.websocket_timing":
        record["engine_families"] = engine_families
        record["engine_ids"] = sorted(redacted_engine_ids)

# This is the only file intended to be retained. It deliberately contains no
# raw event, prompt, authorization header, token, or complete upstream ID.
with Path(redacted_events_path).open("w", encoding="utf-8") as output:
    for record in records:
        output.write(json.dumps(record, ensure_ascii=False, separators=(",", ":")) + "\n")
Path(summary_path).write_text(json.dumps(summary, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
print(json.dumps(summary, ensure_ascii=False, separators=(",", ":")))
PY
}

run_route_test() {
  local workdir
  workdir="$(mktemp -d "${TMPDIR:-/tmp}/codex-check.XXXXXX")" || return 1
  local events_log="$workdir/events.jsonl"
  local summary_log="$workdir/summary.json"
  local redacted_events_log="$workdir/events.redacted.jsonl"
  local exec_log="$workdir/exec.jsonl"
  local trace_log="$workdir/trace.log"

  cleanup() {
    if [ "$KEEP_LOGS" -eq 1 ]; then
      printf '\nRedacted logs kept at: %s\n' "$workdir"
    else
      rm -rf "$workdir"
    fi
  }
  trap cleanup EXIT HUP INT TERM

  printf '\n[Routing test]\n'
  local route_rc=0
  if [ -n "$EVENTS_FILE" ]; then
    # Read-only input mode. The source capture is never copied to the output
    # directory; only analyze_events' redacted artifacts may be retained.
    events_log="$EVENTS_FILE"
  elif [ "$MOCK" -eq 1 ]; then
    make_mock_events "$events_log"
    local mock_rc=$?
    if [ "$mock_rc" -ne 0 ]; then
      trap - EXIT HUP INT TERM
      cleanup
      return "$mock_rc"
    fi
    cp "$events_log" "$exec_log"
  elif [ "$WS_MODE" -eq 1 ]; then
    # Ensure downstream redaction still has a valid input when the optional
    # websocket client cannot import, authenticate, or dial the endpoint.
    : > "$events_log"
    run_websocket_test "$events_log"
    route_rc=$?
  else
    printf '  Sending one low-reasoning request...\n'
    RUST_LOG='codex_api::responses_websocket_timing=trace' \
    codex -a never exec \
        --ephemeral \
        --json \
        --model "$MODEL" \
        --sandbox read-only \
        --skip-git-repo-check \
        -C "$workdir" \
        -c model_reasoning_effort=low \
        'Reply with exactly OK. Do not call tools, inspect files, or modify anything.' \
        >"$exec_log" 2>"$trace_log"
    route_rc=$?
    # stdout/stderr are retained only for local parsing; --keep-logs does not expose them.
    cat "$exec_log" "$trace_log" >"$events_log"
    if [ "$route_rc" -ne 0 ]; then
      printf '  codex exec exit code: %s\n' "$route_rc"
    fi
  fi

  local summary_line
  summary_line="$(analyze_events "$events_log" "$summary_log" "$redacted_events_log")"
  "$PYTHON_BIN" - "$summary_line" <<'PY'
import json
import sys

summary = json.loads(sys.argv[1])
print(f"  Completed response.model: {', '.join(summary['completed_response_models']) or '-'}")
print(f"  websocket_timing events: {summary['timing_event_count']}")
print(f"  Engine families: {', '.join(summary['engine_families']) or '-'}")
print(f"  Result: {summary['result']}")
if summary["model_engine_mismatch"]:
    print("  Warning: response.model matches the request, but telemetry reports a Luna-family engine.")
elif not summary["completed_response_models"]:
    print("  Result detail: no response.completed response.model was captured.")
elif not summary["engine_families"]:
    print("  Result detail: no websocket_timing engine family was captured (HTTPS fallback or trace format change is possible).")
elif not summary["model_name_matches"]:
    print("  Result detail: response.model did not match the requested model.")
PY

  if [ "$KEEP_LOGS" -eq 1 ]; then
    # Deliberately expose only the structured, redacted summary and JSONL event records.
    cp "$summary_log" "$workdir/summary.redacted.json"
    if [ "$EVENTS_FILE" != "$events_log" ]; then
      rm -f "$events_log"
    fi
    rm -f "$exec_log" "$trace_log" "$summary_log"
  else
    if [ "$EVENTS_FILE" != "$events_log" ]; then
      rm -f "$events_log"
    fi
    rm -f "$exec_log" "$trace_log" "$summary_log" "$redacted_events_log"
  fi

  trap - EXIT HUP INT TERM
  cleanup
  return "$route_rc"
}

run_websocket_test() {
  local events_log="$1"
  if [ -z "$PYTHON_BIN" ]; then
    echo "Error: Python is required for --ws." >&2
    return 1
  fi
  "$PYTHON_BIN" - "$events_log" "$WS_ENDPOINT" "$MODEL" "$ROUNDS" "$SAME_CONNECTION_ROUNDS" <<'PY'
import json
import os
import re
import sys
import time
from pathlib import Path

events_path, endpoint, model, rounds, same_rounds = sys.argv[1:]
rounds = int(rounds)
same_rounds = int(same_rounds)

def read_text(path):
    try:
        return Path(path).read_text(encoding="utf-8", errors="ignore")
    except Exception:
        return ""

def configured_base_url():
    for key in ("SUB2API_BASE_URL", "OPENAI_BASE_URL", "CODEX_BASE_URL"):
        value = os.environ.get(key, "").strip()
        if value:
            return value.rstrip("/")
    home = os.environ.get("USERPROFILE") or os.environ.get("HOME") or ""
    config = read_text(os.path.join(home, ".codex", "config.toml"))
    for pattern in (r"(?m)^\s*(?:base_url|openai_base_url)\s*=\s*['\"]([^'\"]+)",
                    r"(?m)^\s*baseURL\s*=\s*['\"]([^'\"]+)"):
        match = re.search(pattern, config)
        if match:
            return match.group(1).rstrip("/")
    return "http://127.0.0.1:8000"

def configured_token():
    for key in ("SUB2API_API_KEY", "OPENAI_API_KEY", "CODEX_API_KEY"):
        value = os.environ.get(key, "").strip()
        if value:
            return value
    home = os.environ.get("USERPROFILE") or os.environ.get("HOME") or ""
    auth = read_text(os.path.join(home, ".codex", "auth.json"))
    try:
        data = json.loads(auth)
    except Exception:
        data = {}
    for key in ("access_token", "api_key", "token"):
        value = data.get(key)
        if isinstance(value, str) and value.strip():
            return value.strip()
    account = data.get("tokens")
    if isinstance(account, dict):
        for key in ("access_token", "api_key", "token"):
            value = account.get(key)
            if isinstance(value, str) and value.strip():
                return value.strip()
    return ""

def ws_url(base):
    if base.startswith("https://"):
        base = "wss://" + base[8:]
    elif base.startswith("http://"):
        base = "ws://" + base[7:]
    return base.rstrip("/") + "/" + endpoint.lstrip("/")

def append_event(out, value):
    if isinstance(value, dict):
        out.write(json.dumps(value, ensure_ascii=False, separators=(",", ":")) + "\n")

try:
    import websocket  # websocket-client; optional dependency for this diagnostic tool
except Exception:
    print("WebSocket test: unavailable (install Python package websocket-client or use --events/--mock)", file=sys.stderr)
    raise SystemExit(3)

token = configured_token()
if not token:
    print("WebSocket test: no local API token found; set SUB2API_API_KEY or configure ~/.codex/auth.json", file=sys.stderr)
    raise SystemExit(2)

url = ws_url(configured_base_url())
timeout = float(os.environ.get("CODEX_CHECK_WS_TIMEOUT", "30"))
handshakes = 0
completed = 0
failures = 0
close_codes = []
with Path(events_path).open("w", encoding="utf-8") as out:
    for connection_no in range(1, rounds + 1):
        ws = None
        try:
            ws = websocket.create_connection(
                url,
                timeout=timeout,
                header=["Authorization: Bearer " + token, "User-Agent: codex-check-ws/1.0"],
                suppress_origin=True,
            )
            handshakes += 1
            for turn in range(1, same_rounds + 1):
                payload = {
                    "type": "response.create",
                    "model": model,
                    "reasoning": {"effort": "high"},
                    "input": "Reply with exactly OK. Do not call tools.",
                }
                ws.send(json.dumps(payload, separators=(",", ":")))
                deadline = time.monotonic() + timeout
                saw_terminal = False
                while time.monotonic() < deadline:
                    raw = ws.recv()
                    if raw is None:
                        break
                    if isinstance(raw, bytes):
                        raw = raw.decode("utf-8", "replace")
                    try:
                        event = json.loads(raw)
                    except Exception:
                        continue
                    append_event(out, event)
                    kind = event.get("type") if isinstance(event, dict) else ""
                    if kind in ("response.completed", "response.done"):
                        completed += 1
                        saw_terminal = True
                        break
                    if kind in ("error", "response.failed"):
                        failures += 1
                        saw_terminal = True
                        break
                if not saw_terminal:
                    failures += 1
                    append_event(out, {"type": "local.ws_timeout", "connection": connection_no, "turn": turn})
        except Exception as exc:
            failures += 1
            # Never persist exception text because some websocket libraries can
            # include request headers in diagnostic errors.
            append_event(out, {"type": "local.ws_error", "connection": connection_no})
            print(f"WebSocket connection {connection_no}: failed ({type(exc).__name__})", file=sys.stderr)
        finally:
            if ws is not None:
                try:
                    ws.close()
                except Exception:
                    pass

print(json.dumps({
    "handshake_successes": handshakes,
    "connections": rounds,
    "completed_events": completed,
    "failures": failures,
}, separators=(",", ":")))
raise SystemExit(0 if handshakes and failures == 0 else 1)
PY
}

if [ "$DO_QUOTA" -eq 1 ]; then
  if [ "$MOCK" -eq 1 ]; then
    printf '\n[Account]\n  Offline mock: skipped (no credentials read)\n'
    printf '\n[Quota]\n  Offline mock: skipped\n'
  else
    query_account_and_quota
  fi
fi

if [ "$DO_ROUTE" -eq 1 ]; then
  run_route_test
fi
