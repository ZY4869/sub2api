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

    expect(state.tokenDisplayMode.value).toBe("full");

    state.setTokenDisplayMode("compact");

    expect(storage.get("token-display-mode")).toBe("compact");
    expect(state.tokenDisplayMode.value).toBe("compact");

    const secondState = tokenModule.useTokenDisplayMode();
    expect(secondState.tokenDisplayMode.value).toBe("compact");
  });

  it("restores persisted mode for new imports", async () => {
    storage.set("token-display-mode", "compact");

    const tokenModule = await import("../useTokenDisplayMode");
    const state = tokenModule.useTokenDisplayMode();

    expect(tokenModule.getPersistedTokenDisplayMode()).toBe("compact");
    expect(state.tokenDisplayMode.value).toBe("compact");
    expect(state.formatTokenDisplay(171_600)).toBe("171.6K");
  });
});
