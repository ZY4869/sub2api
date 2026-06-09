const assert = require("node:assert/strict");
const { spawn } = require("node:child_process");
const http = require("node:http");
const { test } = require("node:test");
const { redact } = require("./sub2api-admin.js");

const script = require.resolve("./sub2api-admin.js");

function runCLI(args, env = {}) {
  return new Promise((resolve) => {
    const child = spawn(process.execPath, [script, ...args], {
      cwd: __dirname,
      env: {
        PATH: process.env.PATH,
        SystemRoot: process.env.SystemRoot,
        COMSPEC: process.env.COMSPEC,
        PATHEXT: process.env.PATHEXT,
        TEMP: process.env.TEMP,
        TMP: process.env.TMP,
        USERPROFILE: process.env.USERPROFILE,
        SUB2API_ADMIN_BASE_URL: "",
        SUB2API_ADMIN_TOKEN: "",
        SUB2API_ADMIN_DRY_RUN: "",
        ...env,
      },
      stdio: ["ignore", "pipe", "pipe"],
    });
    let stdout = "";
    let stderr = "";
    const timer = setTimeout(() => {
      child.kill();
      resolve({ status: null, signal: "SIGTERM", stdout, stderr, error: new Error("CLI timed out") });
    }, 10000);

    child.stdout.setEncoding("utf8");
    child.stderr.setEncoding("utf8");
    child.stdout.on("data", (chunk) => {
      stdout += chunk;
    });
    child.stderr.on("data", (chunk) => {
      stderr += chunk;
    });
    child.on("close", (status, signal) => {
      clearTimeout(timer);
      resolve({ status, signal, stdout, stderr });
    });
  });
}

function startServer(handler) {
  const requests = [];
  const server = http.createServer((req, res) => {
    res.setHeader("connection", "close");
    requests.push({
      method: req.method,
      url: req.url,
      authorization: req.headers.authorization,
    });
    handler(req, res);
  });
  return new Promise((resolve, reject) => {
    server.once("error", reject);
    server.listen(0, "127.0.0.1", () => {
      const { port } = server.address();
      resolve({
        baseURL: `http://127.0.0.1:${port}`,
        requests,
        close: () => new Promise((done) => server.close(done)),
      });
    });
  });
}

test("restore-original-proxy defaults to dry-run and does not send writes", async () => {
  const server = await startServer((_req, res) => {
    res.writeHead(500, { "content-type": "application/json" });
    res.end(JSON.stringify({ error: "unexpected write" }));
  });
  try {
    const result = await runCLI(["restore-original-proxy", "--account-id", "42"], {
      SUB2API_ADMIN_BASE_URL: server.baseURL,
      SUB2API_ADMIN_TOKEN: "admin-token",
    });

    assert.equal(result.status, 0, result.stderr);
    assert.deepEqual(server.requests, []);
    assert.match(result.stdout, /"dry_run": true/);
    assert.match(result.stdout, /"path": "\/api\/v1\/admin\/accounts\/42\/restore-original-proxy"/);
  } finally {
    await server.close();
  }
});

test("restore-original-proxy executes write when dry-run is disabled", async () => {
  const server = await startServer((req, res) => {
    assert.equal(req.method, "POST");
    assert.equal(req.url, "/api/v1/admin/accounts/42/restore-original-proxy");
    res.writeHead(200, { "content-type": "application/json" });
    res.end(JSON.stringify({ data: { token: "secret-token", ok: true } }));
  });
  try {
    const result = await runCLI(["restore-original-proxy", "--account-id", "42"], {
      SUB2API_ADMIN_BASE_URL: server.baseURL,
      SUB2API_ADMIN_TOKEN: "admin-token",
      SUB2API_ADMIN_DRY_RUN: "0",
    });

    assert.equal(result.status, 0, result.stderr);
    assert.equal(server.requests.length, 1);
    assert.equal(server.requests[0].authorization, "Bearer admin-token");
    assert.match(result.stdout, /\[redacted\]/);
    assert.doesNotMatch(result.stdout, /secret-token/);
  } finally {
    await server.close();
  }
});

test("configuration errors are explicit and do not print token values", async () => {
  const missingBase = await runCLI(["health"], { SUB2API_ADMIN_TOKEN: "admin-token" });
  assert.notEqual(missingBase.status, 0);
  assert.match(missingBase.stderr, /SUB2API_ADMIN_BASE_URL is required/);
  assert.doesNotMatch(missingBase.stderr, /admin-token/);

  const missingToken = await runCLI(["health"], { SUB2API_ADMIN_BASE_URL: "http://127.0.0.1:1" });
  assert.notEqual(missingToken.status, 0);
  assert.match(missingToken.stderr, /SUB2API_ADMIN_TOKEN is required/);
});

test("non-2xx errors are summarized with sensitive fields redacted", async () => {
  const server = await startServer((_req, res) => {
    res.writeHead(403, { "content-type": "application/json" });
    res.end(JSON.stringify({
      error: {
        message: "denied",
        token: "server-token",
        password: "server-password",
        secret: "server-secret",
        authorization: "Bearer leaked",
      },
    }));
  });
  try {
    const result = await runCLI(["health"], {
      SUB2API_ADMIN_BASE_URL: server.baseURL,
      SUB2API_ADMIN_TOKEN: "admin-token",
    });

    assert.notEqual(result.status, 0);
    assert.match(result.stderr, /admin API returned 403/);
    assert.match(result.stderr, /\[redacted\]/);
    assert.doesNotMatch(result.stderr, /server-token|server-password|server-secret|Bearer leaked|admin-token/);
  } finally {
    await server.close();
  }
});

test("redact recursively masks sensitive keys", () => {
  const out = redact({
    token: "top",
    nested: {
      password: "pw",
      secret: "sec",
      value: "visible",
    },
    list: [{ authorization: "Bearer x" }],
  });

  assert.deepEqual(out, {
    token: "[redacted]",
    nested: {
      password: "[redacted]",
      secret: "[redacted]",
      value: "visible",
    },
    list: [{ authorization: "[redacted]" }],
  });
});
