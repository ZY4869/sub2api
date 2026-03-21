import { describe, expect, it } from "vitest";

import { formatTokenCount } from "../format";

describe("formatTokenCount", () => {
  it("formats full mode with thousands separators", () => {
    expect(formatTokenCount(999, { mode: "full" })).toBe("999");
    expect(formatTokenCount(15_735, { mode: "full" })).toBe("15,735");
    expect(formatTokenCount(171_600, { mode: "full" })).toBe("171,600");
  });

  it("formats compact mode with K, M, and B", () => {
    expect(formatTokenCount(999, { mode: "compact" })).toBe("999");
    expect(formatTokenCount(15_735, { mode: "compact" })).toBe("15.7K");
    expect(formatTokenCount(171_600, { mode: "compact" })).toBe("171.6K");
    expect(formatTokenCount(1_250_000, { mode: "compact" })).toBe("1.3M");
    expect(formatTokenCount(2_500_000_000, { mode: "compact" })).toBe(
      "2.5B",
    );
  });
});
