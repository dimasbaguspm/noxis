// Pure CSV helpers: no component knows how to serialize a grid.

function escapeCell(value: string): string {
  return /[",\n\r]/.test(value) ? `"${value.replaceAll('"', '""')}"` : value;
}

export function toCSV(columns: string[], rows: string[][]): string {
  return [columns, ...rows].map((row) => row.map(escapeCell).join(",")).join("\n");
}

export function downloadCSV(filename: string, csv: string): void {
  const url = URL.createObjectURL(new Blob([csv], { type: "text/csv;charset=utf-8" }));
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  link.click();
  URL.revokeObjectURL(url);
}
