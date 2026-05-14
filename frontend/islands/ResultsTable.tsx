import { useEffect, useRef } from "react"
import * as VTable from "@visactor/vtable"

type Row = Record<string, string | number>

interface ResultsTableProps {
  rows: Row[]
}

export function ResultsTable({ rows }: ResultsTableProps) {
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!ref.current) return
    const fields = Object.keys(rows[0] || { id: "" })
    const table = new VTable.ListTable(ref.current, {
      records: rows,
      columns: fields.map((field) => ({ field, title: field, width: 150 })),
      widthMode: "standard",
      theme: VTable.themes.ARCO,
    })
    return () => table.release()
  }, [rows])

  return <div ref={ref} className="h-[420px] w-full overflow-hidden rounded-lg border border-line bg-surface" />
}
