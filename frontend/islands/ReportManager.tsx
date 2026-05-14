import { useEffect, useMemo, useState } from "react"
import { ReportChart } from "./ReportChart"
import { ResultsTable } from "./ResultsTable"
import { apiBase, requireSession } from "../lib/auth"

type Row = Record<string, string | number>

interface ReportWidget {
  id: string
  reportId: string
  type: "bar" | "table" | string
  title: string
  dataSource?: string
}

interface Report {
  id: string
  name: string
  description: string
  widgets?: ReportWidget[]
}

interface QueryResult {
  dimensions: string[]
  measures: string[]
  rows: Row[]
}

const emptyReport: Report = { id: "", name: "", description: "", widgets: [] }

export function ReportManager() {
  const [reports, setReports] = useState<Report[]>([])
  const [selectedId, setSelectedId] = useState("")
  const [draft, setDraft] = useState<Report>(emptyReport)
  const [query, setQuery] = useState<QueryResult>({ dimensions: [], measures: [], rows: [] })
  const [message, setMessage] = useState("正在连接报表 API...")
  const selected = useMemo(() => reports.find((report) => report.id === selectedId), [reports, selectedId])

  async function api<T>(path: string, init?: RequestInit): Promise<T> {
    requireSession()
    const response = await fetch(`${apiBase}${path}`, {
      ...init,
      credentials: "include",
      headers: {
        "Content-Type": "application/json",
        ...(init?.headers || {}),
      },
    })
    if (!response.ok) throw new Error(await response.text())
    return response.json()
  }

  async function loginAndLoad() {
    try {
      requireSession()
      const list = await fetch(`${apiBase}/api/v1/reports`, {
        credentials: "include",
      })
      if (!list.ok) throw new Error(await list.text())
      const reports = await list.json() as Report[]
      setReports(reports)
      if (reports[0]) {
        setSelectedId(reports[0].id)
        setDraft(reports[0])
        await loadQuery(reports[0].id)
      }
      setMessage("")
    } catch (error) {
      setMessage(`报表 API 未连接：${error instanceof Error ? error.message : "未知错误"}`)
    }
  }

  async function loadQuery(reportId: string) {
    requireSession()
    const response = await fetch(`${apiBase}/api/v1/reports/${reportId}/query`, {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: "{}",
    })
    if (!response.ok) throw new Error(await response.text())
    setQuery(await response.json())
  }

  async function selectReport(report: Report) {
    setSelectedId(report.id)
    setDraft(report)
    setMessage("")
    try {
      await loadQuery(report.id)
    } catch (error) {
      setMessage(`查询失败：${error instanceof Error ? error.message : "未知错误"}`)
    }
  }

  async function save() {
    try {
      const method = selectedId ? "PUT" : "POST"
      const path = selectedId ? `/api/v1/reports/${selectedId}` : "/api/v1/reports"
      const report = await api<Report>(path, { method, body: JSON.stringify(draft) })
      if (selectedId) {
        setReports(reports.map((item) => item.id === selectedId ? report : item))
      } else {
        setReports([...reports, report])
      }
      setDraft(report)
      setSelectedId(report.id)
      setMessage("已保存报表")
    } catch (error) {
      setMessage(`保存失败：${error instanceof Error ? error.message : "未知错误"}`)
    }
  }

  async function addWidget(type: "bar" | "table") {
    if (!selectedId) return
    try {
      const widget = await api<ReportWidget>(`/api/v1/reports/${selectedId}/widgets`, {
        method: "POST",
        body: JSON.stringify({ type, title: type === "bar" ? "新图表" : "新明细表", dataSource: "survey-dict" }),
      })
      const next = { ...draft, widgets: [...(draft.widgets || []), widget] }
      setDraft(next)
      setReports(reports.map((item) => item.id === selectedId ? next : item))
      setMessage("已添加组件")
    } catch (error) {
      setMessage(`添加组件失败：${error instanceof Error ? error.message : "未知错误"}`)
    }
  }

  useEffect(() => {
    loginAndLoad()
  }, [])

  const xField = query.dimensions[0] || "month"
  const yField = query.measures[0] || "submissions"

  return (
    <div className="grid gap-5 xl:grid-cols-[320px_minmax(0,1fr)]">
      <aside className="rounded-lg border border-line bg-surface p-4">
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-sm font-semibold">报表列表</h2>
          <button className="rounded-lg border border-line px-3 py-1.5 text-xs hover:border-primary" onClick={() => { setSelectedId(""); setDraft(emptyReport); setQuery({ dimensions: [], measures: [], rows: [] }) }}>
            新建
          </button>
        </div>
        <div className="grid gap-2">
          {reports.map((report) => (
            <button key={report.id} className={`rounded-lg border px-3 py-3 text-left ${report.id === selectedId ? "border-primary bg-blue-50" : "border-line"}`} onClick={() => selectReport(report)}>
              <span className="block text-sm font-medium">{report.name}</span>
              <span className="mt-1 block text-xs text-muted">{report.description}</span>
            </button>
          ))}
        </div>
      </aside>

      <section className="grid gap-5">
        <div className="rounded-lg border border-line bg-surface p-4">
          <div className="grid gap-3 md:grid-cols-[1fr_1.5fr_auto]">
            <input className="rounded-lg border border-line px-3 py-2 text-sm" placeholder="报表名称" value={draft.name} onChange={(event) => setDraft({ ...draft, name: event.target.value })} />
            <input className="rounded-lg border border-line px-3 py-2 text-sm" placeholder="描述" value={draft.description} onChange={(event) => setDraft({ ...draft, description: event.target.value })} />
            <button className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white" onClick={save}>保存</button>
          </div>
          {message && <div className="mt-3 rounded-lg bg-blue-50 px-3 py-2 text-sm text-primary">{message}</div>}
        </div>

        <div className="flex flex-wrap gap-2">
          <button className="rounded-lg border border-line px-3 py-2 text-sm hover:border-primary" onClick={() => addWidget("bar")}>添加图表</button>
          <button className="rounded-lg border border-line px-3 py-2 text-sm hover:border-primary" onClick={() => addWidget("table")}>添加明细表</button>
          <button className="rounded-lg border border-line px-3 py-2 text-sm hover:border-primary" disabled={!selectedId} onClick={() => selectedId && loadQuery(selectedId)}>刷新数据</button>
        </div>

        <div className="grid gap-5 xl:grid-cols-[1fr_1.15fr]">
          <ReportChart data={query.rows} xField={xField} yField={yField} title={draft.widgets?.find((item) => item.type !== "table")?.title || draft.name || "报表图表"} />
          <div>
            <h2 className="mb-3 text-base font-semibold">报表明细</h2>
            <ResultsTable rows={query.rows} />
          </div>
        </div>
      </section>
    </div>
  )
}
