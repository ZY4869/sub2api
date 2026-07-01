export type CsvCell = string | number | boolean | null | undefined;

export function escapeCsvCell(value: CsvCell): string {
  if (value == null) {
    return "";
  }

  const raw = String(value);
  const escaped = raw.replace(/"/g, '""');
  const injectionSafe = /^[=+\-@\t\r]/.test(raw) ? `'${escaped}` : escaped;

  return /[,"\n\r]/.test(injectionSafe) ? `"${injectionSafe}"` : injectionSafe;
}

export function buildCsvContent(rows: CsvCell[][]): string {
  return rows.map((row) => row.map(escapeCsvCell).join(",")).join("\n");
}
