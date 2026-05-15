import { useEffect, useRef, useState } from "react"

type Row = Record<string, string | number>

interface ReportChartProps {
  data: Row[]
  xField?: string
  yField?: string
  title?: string
}

export function ReportChart({ data, xField = "month", yField = "submissions", title = "报表图表" }: ReportChartProps) {
  const ref = useRef<HTMLDivElement>(null)
  const [error, setError] = useState("")

  useEffect(() => {
    if (!ref.current || data.length === 0) return
    let chart: { renderSync: () => void; release: () => void } | null = null
    let disposed = false
    async function render() {
      try {
        const module = await import("@visactor/vchart")
        const VChart = module.VChart || module.default?.VChart
        if (!VChart || !ref.current || disposed) return
        ref.current.innerHTML = ""
        chart = new VChart({
          type: "bar",
          data: { values: data },
          xField,
          yField,
          seriesField: xField,
          height: 360,
          axes: [
            { orient: "bottom", type: "band" },
            { orient: "left", type: "linear" },
          ],
          title: { visible: true, text: title },
        }, { dom: ref.current })
        chart.renderSync()
        setError("")
      } catch (nextError) {
        console.warn("VChart render failed, using HTML fallback", nextError)
        setError(nextError instanceof Error ? nextError.message : "图表渲染失败")
      }
    }
    render()
    return () => {
      disposed = true
      chart?.release()
    }
  }, [data, xField, yField, title])

  if (data.length === 0) {
    return <div className="grid h-[420px] w-full place-items-center rounded-lg border border-line bg-white p-3 text-sm text-muted">暂无可绘制的数据</div>
  }
  if (error) {
    return <FallbackChart data={data} xField={xField} yField={yField} title={title} />
  }
  return <div ref={ref} className="h-[420px] min-w-[360px] rounded-lg border border-line bg-white p-3" />
}

function FallbackChart({ data, xField, yField, title }: { data: Row[]; xField: string; yField: string; title: string }) {
  const max = Math.max(...data.map((row) => Number(row[yField] || 0)), 1)
  return <div className="min-h-[420px] rounded-lg border border-line bg-white p-4">
    {title && <h2 className="mb-3 break-words text-base font-semibold">{title}</h2>}
    <div className="grid gap-2">
      {data.map((row, index) => {
        const value = Number(row[yField] || 0)
        return <div key={index} className="grid grid-cols-[120px_minmax(0,1fr)_56px] items-center gap-2 text-sm">
          <span className="truncate text-muted">{String(row[xField] ?? index + 1)}</span>
          <span className="h-3 overflow-hidden rounded-full bg-gray-100"><span className="block h-full rounded-full bg-primary" style={{ width: `${Math.max(4, value / max * 100)}%` }} /></span>
          <span className="text-right font-medium">{value}</span>
        </div>
      })}
    </div>
  </div>
}
