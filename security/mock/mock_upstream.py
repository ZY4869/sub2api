from http.server import BaseHTTPRequestHandler, HTTPServer
import json
import time


def _selected_headers(headers):
    keys = (
        "authorization",
        "x-api-key",
        "cookie",
        "anthropic-version",
        "anthropic-beta",
        "x-claude-code-session-id",
        "x-client-request-id",
        "content-type",
        "user-agent",
    )
    return {
        key: headers.get(key)
        for key in keys
        if headers.get(key) is not None
    }


class Handler(BaseHTTPRequestHandler):
    server_version = "Sub2APIMockUpstream/1.0"

    def _write_json(self, code: int, payload: dict) -> None:
        body = json.dumps(payload).encode("utf-8")
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def _write_redirect(self, code: int, location: str) -> None:
        self.send_response(code)
        self.send_header("Location", location)
        self.end_headers()

    def _log_request(self, body: bytes) -> None:
        entry = {
            "ts": time.strftime("%Y-%m-%dT%H:%M:%S%z"),
            "method": self.command,
            "path": self.path,
            "body_len": len(body),
            "headers": _selected_headers(self.headers),
        }
        print(json.dumps(entry, ensure_ascii=True), flush=True)

    def log_message(self, format, *args):
        return

    def do_GET(self):
        self._log_request(b"")
        self._write_json(
            200,
            {
                "ok": True,
                "method": "GET",
                "path": self.path,
                "mock": "upstream",
            },
        )

    def do_POST(self):
        length = int(self.headers.get("Content-Length", "0"))
        body = self.rfile.read(length) if length else b""
        self._log_request(body)

        if self.path.startswith("/redirect-local/"):
            self._write_redirect(
                302,
                "http://127.0.0.1:19090/final/v1/messages?beta=true&redirected=1",
            )
            return

        if self.path.startswith("/status/500/"):
            self._write_json(
                500,
                {
                    "error": {
                        "type": "mock_upstream_error",
                        "message": "mock upstream failure sentinel",
                    },
                    "path": self.path,
                },
            )
            return

        if self.path.startswith("/echo-headers/"):
            self._write_json(
                200,
                {
                    "ok": True,
                    "path": self.path,
                    "headers": _selected_headers(self.headers),
                    "received_bytes": len(body),
                },
            )
            return

        if self.path.endswith("/stream"):
            self.send_response(200)
            self.send_header("Content-Type", "text/event-stream")
            self.end_headers()
            for i in range(3):
                self.wfile.write(f"data: {{\"delta\":\"chunk-{i}\"}}\n\n".encode("utf-8"))
                self.wfile.flush()
                time.sleep(0.1)
            return

        self._write_json(
            200,
            {
                "id": "mock-response",
                "path": self.path,
                "headers": _selected_headers(self.headers),
                "received_bytes": len(body),
                "usage": {
                    "prompt_tokens": 10,
                    "completion_tokens": 5,
                    "total_tokens": 15,
                },
                "choices": [
                    {
                        "message": {
                            "content": "mock response",
                        }
                    }
                ],
            },
        )


if __name__ == "__main__":
    HTTPServer(("0.0.0.0", 19090), Handler).serve_forever()
