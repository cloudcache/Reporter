import { useEffect, useMemo, useState } from "react"
import { ReportChart } from "./ReportChart"
import { ResultsTable } from "./ResultsTable"
import { authedJson } from "../lib/auth"

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
  type?: string
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
const reportTypeLabels: Record<string, string> = { satisfaction: "满意度分析", complaint: "评价投诉分析", followup: "随访分析", custom: "自定义报表" }

export function ReportManager() {
  const [reports, setReports] = useState<Report[]>([])
  const [selectedId, setSelectedId] = useState("")
  const [draft, setDraft] = useState<Report>(emptyReport)
  const [query, setQuery] = useState<QueryResult>({ dimensions: [], measures: [], rows: [] })
  const [message, setMessage] = useState("正在连接报表 API...")
  const selected = useMemo(() => reports.find((report) => report.id === selectedId), [reports, selectedId])

  async function api<T>(path: string, init?: RequestInit): Promise<T> {
    return authedJson<T>(path, init)
  }

  async function loginAndLoad() {
    try {
      const nextReports = await authedJson<Report[]>("/api/v1/reports")
      setReports(nextReports)
      const preferred = nextReports.find((item) => item.type === "satisfaction") || nextReports[0]
      if (preferred) {
        setSelectedId(preferred.id)
        setDraft(preferred)
        await loadQuery(preferred.id)
      }
      setMessage("")
    } catch (error) {
      setMessage(`报表 API 未连接：${error instanceof Error ? error.message : "未知错误"}`)
    }
  }

  async function loadQuery(reportId: string) {
    setQuery(await authedJson<QueryResult>(`/api/v1/reports/${reportId}/query`, {
      method: "POST",
      body: "{}",
    }))
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
        body: JSON.stringify({ type, title: type === "bar" ? "新图表" : "新明细表", dataSource: draft.type === "complaint" ? "evaluation_complaints" : draft.type === "satisfaction" ? "survey_submissions" : "followup_records" }),
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
          <h2 className="text-sm font-semibold">分析报表</h2>
          <button className="rounded-lg border border-line px-3 py-1.5 text-xs hover:border-primary" onClick={() => { setSelectedId(""); setDraft(emptyReport); setQuery({ dimensions: [], measures: [], rows: [] }) }}>
            新建
          </button>
        </div>
        <div className="mb-3 rounded-lg bg-gray-50 p-3 text-xs leading-5 text-muted">
          满意度分析、评价投诉分析和随访分析在这里并列管理，数据从各业务表实时聚合。
        </div>
        <div className="grid gap-2">
          {reports.map((report) => (
            <button key={report.id} className={`rounded-lg border px-3 py-3 text-left ${report.id === selectedId ? "border-primary bg-blue-50" : "border-line"}`} onClick={() => selectReport(report)}>
              <span className="block text-sm font-medium">{report.name}</span>
              <span className="mt-1 inline-flex rounded-full bg-gray-100 px-2 py-0.5 text-xs text-muted">{reportTypeLabels[report.type || "custom"] || report.type}</span>
              <span className="mt-1 block text-xs text-muted">{report.description}</span>
            </button>
          ))}
        </div>
      </aside>

      <section className="grid gap-5">
        <div className="rounded-lg border border-line bg-surface p-4">
          <div className="grid gap-3 md:grid-cols-[180px_1fr_1.5fr_auto]">
            <select className="rounded-lg border border-line px-3 py-2 text-sm" value={draft.type || "custom"} onChange={(event) => setDraft({ ...draft, type: event.target.value })}>
              {Object.entries(reportTypeLabels).map(([id, label]) => <option key={id} value={id}>{label}</option>)}
            </select>
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
