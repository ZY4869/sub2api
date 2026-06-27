#!/usr/bin/env node

const sensitiveKeys = new Set([
  "authorization",
  "token",
  "api_key",
  "access_token",
  "refresh_token",
  "password",
  "secret",
  "credentials",
]);

function usage() {
  return [
    "Usage: node sub2api-admin.js <command> [options]",
    "",
    "Commands:",
    "  health",
    "  list-proxies",
    "  list-accounts",
    "  restore-original-proxy --account-id <id>",
    "",
    "Environment:",
    "  SUB2API_ADMIN_BASE_URL  Required admin API base URL",
    "  SUB2API_ADMIN_TOKEN     Admin bearer token; takes priority over SUB2API_JWT",
    "  SUB2API_JWT             Fallback admin bearer token",
    "  SUB2API_ADMIN_DRY_RUN   Defaults to dry-run; set 0/false/no to execute writes",
  ].join("\n");
}

function parseArgs(argv) {
  const [command, ...rest] = argv;
  const options = {};
  for (let i = 0; i < rest.length; i += 1) {
    const arg = rest[i];
    if (!arg.startsWith("--")) {
      throw new Error(`unexpected argument: ${arg}`);
    }
    const key = arg.slice(2).replace(/-([a-z])/g, (_, ch) => ch.toUpperCase());
    const value = rest[i + 1];
    if (!value || value.startsWith("--")) {
      throw new Error(`missing value for ${arg}`);
    }
    options[key] = value;
    i += 1;
  }
  return { command, options };
}

function isDryRun() {
  const raw = String(process.env.SUB2API_ADMIN_DRY_RUN || "").trim().toLowerCase();
  return !["0", "false", "no"].includes(raw);
}

function requireConfig() {
  const baseURL = String(process.env.SUB2API_ADMIN_BASE_URL || "").trim().replace(/\/+$/, "");
  const token = String(process.env.SUB2API_ADMIN_TOKEN || process.env.SUB2API_JWT || "").trim();
  if (!baseURL) {
    throw new Error("SUB2API_ADMIN_BASE_URL is required");
  }
  if (!token) {
    throw new Error("SUB2API_ADMIN_TOKEN or SUB2API_JWT is required");
  }
  return { baseURL, token };
}

function redact(value) {
  if (Array.isArray(value)) {
    return value.slice(0, 20).map(redact);
  }
  if (value && typeof value === "object") {
    const out = {};
    for (const [key, child] of Object.entries(value)) {
      if (sensitiveKeys.has(key.toLowerCase())) {
        out[key] = "[redacted]";
      } else {
        out[key] = redact(child);
      }
    }
    return out;
  }
  return value;
}

async function request(config, method, path) {
  const response = await fetch(`${config.baseURL}${path}`, {
    method,
    headers: {
      Authorization: `Bearer ${config.token}`,
      Accept: "application/json",
      "Content-Type": "application/json",
      Connection: "close",
    },
  });
  const text = await response.text();
  let body = text;
  try {
    body = text ? JSON.parse(text) : null;
  } catch {
    body = text.slice(0, 500);
  }
  if (!response.ok) {
    const summary = JSON.stringify(redact(body)).slice(0, 1000);
    throw new Error(`admin API returned ${response.status}: ${summary}`);
  }
  return body;
}

function summarizeList(body) {
  const items = Array.isArray(body) ? body : Array.isArray(body?.data) ? body.data : Array.isArray(body?.items) ? body.items : [];
  return {
    count: items.length,
    items: items.slice(0, 50).map((item) => ({
      id: item?.id,
      name: item?.name,
      status: item?.status,
      platform: item?.platform,
      type: item?.type,
    })),
  };
}

async function main() {
  const { command, options } = parseArgs(process.argv.slice(2));
  if (!command || command === "help" || command === "--help") {
    console.log(usage());
    return;
  }

  const config = requireConfig();
  switch (command) {
    case "health": {
      const body = await request(config, "GET", "/api/v1/admin/health");
      console.log(JSON.stringify(redact(body), null, 2));
      return;
    }
    case "list-proxies": {
      const body = await request(config, "GET", "/api/v1/admin/proxies");
      console.log(JSON.stringify(summarizeList(redact(body)), null, 2));
      return;
    }
    case "list-accounts": {
      const body = await request(config, "GET", "/api/v1/admin/accounts");
      console.log(JSON.stringify(summarizeList(redact(body)), null, 2));
      return;
    }
    case "restore-original-proxy": {
      const accountID = String(options.accountId || "").trim();
      if (!/^\d+$/.test(accountID)) {
        throw new Error("--account-id must be a positive integer");
      }
      const path = `/api/v1/admin/accounts/${accountID}/restore-original-proxy`;
      if (isDryRun()) {
        console.log(JSON.stringify({ dry_run: true, method: "POST", path, account_id: Number(accountID) }, null, 2));
        return;
      }
      const body = await request(config, "POST", path);
      console.log(JSON.stringify(redact(body), null, 2));
      return;
    }
    default:
      throw new Error(`unknown command: ${command}`);
  }
}

if (require.main === module) {
  main().catch((err) => {
    console.error(err instanceof Error ? err.message : String(err));
    process.exitCode = 1;
  });
}

module.exports = {
  parseArgs,
  isDryRun,
  requireConfig,
  redact,
  request,
  summarizeList,
  main,
};
