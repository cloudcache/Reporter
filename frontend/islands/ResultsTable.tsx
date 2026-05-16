import { useEffect, useRef, useState } from "react"

type Row = Record<string, string | number>

interface ResultsTableProps {
  rows: Row[]
  onRowClick?: (row: Row) => void
}

export function ResultsTable({ rows, onRowClick }: ResultsTableProps) {
  const ref = useRef<HTMLDivElement>(null)
  const [error, setError] = useState("")

  useEffect(() => {
    if (!ref.current || rows.length === 0) return
    let table: { release: () => void } | null = null
    let disposed = false
    async function render() {
      try {
        const module = await import("@visactor/vtable")
        const VTable = module
        if (!ref.current || disposed) return
        const fields = Object.keys(rows[0] || { id: "" })
        ref.current.innerHTML = ""
        table = new VTable.ListTable(ref.current, {
          records: rows,
          columns: fields.map((field) => ({ field, title: field, width: 150 })),
          widthMode: "standard",
          theme: VTable.themes.ARCO,
        })
        setError("")
      } catch (nextError) {
        console.warn("VTable render failed, using HTML fallback", nextError)
        setError(nextError instanceof Error ? nextError.message : "表格渲染失败")
      }
    }
    render()
    return () => {
      disposed = true
      table?.release()
    }
  }, [rows])

  if (rows.length === 0) {
    return <div className="grid h-[460px] w-full place-items-center rounded-lg border border-line bg-white text-sm text-muted">暂无明细数据</div>
  }
  if (error || onRowClick) return <FallbackTable rows={rows} onRowClick={onRowClick} />
  return <div className="w-full overflow-x-auto">
    <div ref={ref} className="h-[460px] min-w-[640px] overflow-hidden rounded-lg border border-line bg-white" />
  </div>
}

function FallbackTable({ rows, onRowClick }: { rows: Row[]; onRowClick?: (row: Row) => void }) {
  const fields = Object.keys(rows[0] || {})
  return <div className="rounded-lg border border-line bg-white">
    <div className="max-h-[460px] overflow-auto">
      <table className="w-full min-w-[640px] text-sm">
        <thead className="sticky top-0 bg-gray-50 text-xs uppercase text-muted">
          <tr>{fields.map((field) => <th key={field} className="border-b border-line px-3 py-2 text-left font-semibold">{field}</th>)}</tr>
        </thead>
        <tbody>
          {rows.map((row, index) => (
            <tr key={index} className={`border-b border-line last:border-0 hover:bg-gray-50 ${onRowClick ? "cursor-pointer" : ""}`} onClick={() => onRowClick?.(row)}>
              {fields.map((field) => <td key={field} className="px-3 py-2">{String(row[field] ?? "")}</td>)}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  </div>
}
