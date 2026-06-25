import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

describe("useTokenDisplayMode", () => {
  let storage = new Map<string, string>();

  beforeEach(() => {
    storage = new Map<string, string>();
    vi.resetModules();
    vi.stubGlobal("localStorage", {
      getItem: vi.fn((key: string) => storage.get(key) ?? null),
      setItem: vi.fn((key: string, value: string) => {
        storage.set(key, String(value));
      }),
      removeItem: vi.fn((key: string) => {
        storage.delete(key);
      }),
    });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("reads and writes the shared localStorage key", async () => {
    const tokenModule = await import("../useTokenDisplayMode");
    const state = tokenModule.useTokenDisplayMode();

    expect(state.tokenDisplayMode.value).toBe("m");

    state.setTokenDisplayMode("k");

    expect(storage.get("token-display-mode")).toBe("k");
    expect(state.tokenDisplayMode.value).toBe("k");

    const secondState = tokenModule.useTokenDisplayMode();
    expect(secondState.tokenDisplayMode.value).toBe("k");
  });

  it("restores persisted mode for new imports", async () => {
    storage.set("token-display-mode", "compact");

    const tokenModule = await import("../useTokenDisplayMode");
    const state = tokenModule.useTokenDisplayMode();

    expect(tokenModule.getPersistedTokenDisplayMode()).toBe("m");
    expect(state.tokenDisplayMode.value).toBe("m");
    expect(state.formatTokenDisplay(1_663_471)).toBe("1.7M");
  });
});
