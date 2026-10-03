import type { Result } from "../api/client";

// The result grid: columns + rows from any read (SQL box or table preview).
export default function Grid({ result, error }: { result: Result | null; error: string }) {
  if (error) {
    return (
      <pre className="min-h-[80px] whitespace-pre-wrap break-words rounded-md border border-[#f85149]/40 bg-[#0d1117] p-2.5 text-[#f85149]">
        {error}
      </pre>
    );
  }
  if (!result) {
    return <p className="text-xs text-[#8b949e]">no result yet</p>;
  }
  const cols = result.columns ?? [];
  const rows = result.rows ?? [];
  return (
    <div>
      {cols.length > 0 ? (
        <div className="max-h-[420px] overflow-auto rounded-md border border-[#30363d]">
          <table className="w-full border-collapse text-[13px]">
            <thead className="sticky top-0 bg-[#161b22]">
              <tr>
                {cols.map((c) => (
                  <th key={c} className="border-b border-[#30363d] px-2 py-1.5 text-left text-[#8b949e]">{c}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {rows.map((r, i) => (
                <tr key={i} className="odd:bg-[#0d1117]">
                  {r.map((cell, j) => (
                    <td key={j} className="border-b border-[#21262d] px-2 py-1 font-mono">
                      {cell === "" ? <span className="text-[#8b949e]">NULL</span> : cell}
                    </td>
                  ))}
                </tr>
              ))}
              {rows.length === 0 && (
                <tr><td colSpan={cols.length} className="px-2 py-2 text-xs text-[#8b949e]">no rows</td></tr>
              )}
            </tbody>
          </table>
        </div>
      ) : null}
      <p className="mt-1.5 text-xs text-[#8b949e]">
        {result.message ?? ""} {rows.length > 0 ? `— ${rows.length} row(s)` : ""}
        {result.affected ? ` — ${result.affected} affected` : ""}
      </p>
    </div>
  );
}