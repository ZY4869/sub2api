import { describe, expect, it } from "vitest";
import {
  buildAbsoluteApiUrl,
  buildAbsoluteBackendRootUrl,
  buildApiUrl,
  buildBackendRootUrl,
} from "../url";

describe("api url helpers", () => {
  it("builds API paths from relative and absolute API bases", () => {
    expect(buildApiUrl("/admin/accounts", "/api/v1")).toBe("/api/v1/admin/accounts");
    expect(buildApiUrl("admin/accounts", "https://api.example.com/api/v1/")).toBe(
      "https://api.example.com/api/v1/admin/accounts",
    );
  });

  it("keeps backend root paths relative for same-origin deployments", () => {
    expect(buildBackendRootUrl("/v1/usage", "/api/v1")).toBe("/v1/usage");
  });

  it("uses the API origin for backend root paths when API base is absolute", () => {
    expect(buildBackendRootUrl("/health", "https://api.example.com/api/v1")).toBe(
      "https://api.example.com/health",
    );
  });

  it("builds absolute same-origin API and backend root URLs", () => {
    expect(buildAbsoluteApiUrl("/admin/models/debug/run", "/api/v1")).toBe(
      "http://localhost:3000/api/v1/admin/models/debug/run",
    );
    expect(buildAbsoluteBackendRootUrl("/setup/status", "/api/v1")).toBe(
      "http://localhost:3000/setup/status",
    );
  });
});

