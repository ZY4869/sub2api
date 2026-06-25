import { describe, expect, it } from "vitest";

import { formatTokenCount } from "../format";

describe("formatTokenCount", () => {
  it("formats natural mode with thousands separators", () => {
    expect(formatTokenCount(999, { mode: "natural" })).toBe("999");
    expect(formatTokenCount(15_735, { mode: "natural" })).toBe("15,735");
    expect(formatTokenCount(171_600, { mode: "natural" })).toBe("171,600");
  });

  it("formats fixed K mode", () => {
    expect(formatTokenCount(999, { mode: "k" })).toBe("1K");
    expect(formatTokenCount(15_735, { mode: "k" })).toBe("15.7K");
    expect(formatTokenCount(171_600, { mode: "k" })).toBe("171.6K");
    expect(formatTokenCount(1_250_000, { mode: "k" })).toBe("1250K");
  });

  it("formats fixed M mode", () => {
    expect(formatTokenCount(999, { mode: "m" })).toBe("999");
    expect(formatTokenCount(15_735, { mode: "m" })).toBe("15,735");
    expect(formatTokenCount(171_600, { mode: "m" })).toBe("0.2M");
    expect(formatTokenCount(1_250_000, { mode: "m" })).toBe("1.3M");
  });
});
