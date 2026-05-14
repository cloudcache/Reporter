import { useEffect, useRef } from "react"
import VChartModule from "@visactor/vchart"

type Row = Record<string, string | number>

interface ReportChartProps {
  data: Row[]
  xField?: string
  yField?: string
  title?: string
}

export function ReportChart({ data, xField = "month", yField = "submissions", title = "报表图表" }: ReportChartProps) {
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!ref.current) return
    const { VChart } = VChartModule
    const chart = new VChart({
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
    return () => chart.release()
  }, [data, xField, yField, title])

  return <div ref={ref} className="h-[380px] w-full rounded-lg border border-line bg-surface p-3" />
}
