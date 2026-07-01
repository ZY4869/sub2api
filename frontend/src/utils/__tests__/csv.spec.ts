import { describe, expect, it } from "vitest";

import { buildCsvContent, escapeCsvCell } from "../csv";

describe("csv utilities", () => {
  it("escapes delimiters, quotes, and newlines", () => {
    expect(escapeCsvCell('a,"b"\nc')).toBe('"a,""b""\nc"');
  });

  it("prefixes formula-like values", () => {
    expect(escapeCsvCell("=SUM(A1:A2)")).toBe("'=SUM(A1:A2)");
    expect(escapeCsvCell("-10")).toBe("'-10");
  });

  it("builds CSV rows", () => {
    expect(
      buildCsvContent([
        ["Name", "Value"],
        ["demo", 12],
      ]),
    ).toBe("Name,Value\ndemo,12");
  });
});
