import { useEffect, useRef, useState } from "react"
import { ReportChart } from "./ReportChart"
import { ResultsTable } from "./ResultsTable"
import { authedFetch, authedJson } from "../lib/auth"

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
  code?: string
  type?: string
  category?: string
  subjectType?: string
  defaultDimension?: string
  name: string
  description: string
  widgets?: ReportWidget[]
}

interface QueryResult {
  dimensions: string[]
  measures: string[]
  rows: Row[]
}
interface ReportInsights { sentiment: string; themes: string[]; rootCauses: string[]; suggestions: string[] }
interface ReportFilters { dateFrom: string; dateTo: string; department: string; doctor: string; visitType: string; channel: string; questionId: string }

const emptyReport: Report = { id: "", name: "", description: "", widgets: [] }
const reportTypeLabels: Record<string, string> = { satisfaction: "满意度分析", complaint: "评价投诉分析", followup: "随访分析", custom: "自定义报表" }
const defaultFilters: ReportFilters = { dateFrom: "", dateTo: "", department: "", doctor: "", visitType: "", channel: "", questionId: "" }

export function ReportManager() {
  const [reports, setReports] = useState<Report[]>([])
  const [selectedId, setSelectedId] = useState("")
  const [draft, setDraft] = useState<Report>(emptyReport)
  const [query, setQuery] = useState<QueryResult>({ dimensions: [], measures: [], rows: [] })
  const [insights, setInsights] = useState<ReportInsights | null>(null)
  const [message, setMessage] = useState("正在连接报表 API...")
  const [projectId, setProjectId] = useState("")
  const [filters, setFilters] = useState<ReportFilters>(defaultFilters)
  const [viewMode, setViewMode] = useState<"chart" | "table">("table")
  const [drillRows, setDrillRows] = useState<Row[]>([])
  const [drillTitle, setDrillTitle] = useState("")
  const selectedIdRef = useRef("")

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
        selectedIdRef.current = preferred.id
        setDraft(preferred)
        await loadQuery(preferred.id, currentProjectId(), filters)
      }
      setMessage("")
    } catch (error) {
      setMessage(`报表 API 未连接：${error instanceof Error ? error.message : "未知错误"}`)
    }
  }

  async function loadQuery(reportId: string, scopedProjectId = projectId, nextFilters = filters) {
    const [nextQuery, nextInsights] = await Promise.all([
      authedJson<QueryResult>(`/api/v1/reports/${reportId}/query`, {
      method: "POST",
      body: JSON.stringify({ projectId: scopedProjectId || undefined, filters: nextFilters }),
      }),
      authedJson<ReportInsights>(`/api/v1/reports/${reportId}/insights${queryString(scopedProjectId, nextFilters)}`).catch(() => null),
    ])
    setQuery(nextQuery)
    setInsights(nextInsights)
    setDrillRows([])
    setDrillTitle("")
  }

  async function selectReport(report: Report) {
    setSelectedId(report.id)
    selectedIdRef.current = report.id
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
      selectedIdRef.current = report.id
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

  async function exportReport(format: "word" | "pdf") {
    if (!selectedId) return
    const response = await authedFetch(`/api/v1/reports/${selectedId}/export?format=${format}${queryString(projectId, filters, true)}`)
    if (!response.ok) throw new Error(await response.text())
    const blob = await response.blob()
    const url = URL.createObjectURL(blob)
    const link = document.createElement("a")
    link.href = url
    link.download = `${draft.name || "report"}.${format === "word" ? "doc" : "pdf"}`
    link.click()
    URL.revokeObjectURL(url)
  }

  function exportExcel() {
    if (query.rows.length === 0) return
    const fields = Object.keys(query.rows[0])
    const csv = [fields.join(","), ...query.rows.map((row) => fields.map((field) => `"${String(row[field] ?? "").replace(/"/g, '""')}"`).join(","))].join("\n")
    const blob = new Blob([`\uFEFF${csv}`], { type: "text/csv;charset=utf-8" })
    const url = URL.createObjectURL(blob)
    const link = document.createElement("a")
    link.href = url
    link.download = `${draft.name || "report"}.csv`
    link.click()
    URL.revokeObjectURL(url)
  }

  async function drilldown(row: Row) {
    const nextFilters: ReportFilters = {
      ...filters,
      department: stringValue(row.department) || filters.department,
      doctor: stringValue(row.doctor) || stringValue(row.staff) || filters.doctor,
      questionId: stringValue(row.questionId) || filters.questionId,
    }
    try {
      const result = await authedJson<QueryResult>("/api/v1/reports/drilldown/submissions", {
        method: "POST",
        body: JSON.stringify({ projectId, filters: nextFilters }),
      })
      setDrillRows(result.rows || [])
      setDrillTitle([stringValue(row.department), stringValue(row.question), stringValue(row.reason), stringValue(row.month)].filter(Boolean).join(" · ") || "答卷明细")
    } catch (error) {
      setMessage(`下钻失败：${error instanceof Error ? error.message : "未知错误"}`)
    }
  }

  useEffect(() => {
    setProjectId(currentProjectId())
    loginAndLoad()
    const onScopeChange = (event: Event) => {
      const detail = (event as CustomEvent<{ projectId?: string }>).detail
      const nextProjectId = detail?.projectId || ""
      setProjectId(nextProjectId)
      if (selectedIdRef.current) loadQuery(selectedIdRef.current, nextProjectId, filters)
    }
    window.addEventListener("project-scope-change", onScopeChange)
    return () => window.removeEventListener("project-scope-change", onScopeChange)
  }, [])

  const xField = query.dimensions[0] || "month"
  const yField = query.measures[0] || "submissions"
  const groupedReports = groupReports(reports)
  const summary = summarizeRows(query.rows, query.measures)

  return (
    <div className="grid gap-5 xl:grid-cols-[320px_minmax(0,1fr)]">
      <aside className="rounded-lg border border-line bg-surface p-4 xl:sticky xl:top-24 xl:self-start">
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-sm font-semibold">专题报表目录</h2>
          <button className="rounded-lg border border-line px-3 py-1.5 text-xs hover:border-primary" onClick={() => { setSelectedId(""); selectedIdRef.current = ""; setDraft(emptyReport); setQuery({ dimensions: [], measures: [], rows: [] }) }}>
            新建
          </button>
        </div>
        <div className="mb-3 rounded-lg bg-gray-50 p-3 text-xs leading-5 text-muted">
          按传统行风系统固定报表口径组织，支持项目、周期、科室和渠道筛选。
        </div>
        <div className="grid max-h-[calc(100vh-260px)] gap-4 overflow-y-auto pr-1">
          {groupedReports.map((group) => (
            <section key={group.category} className="grid gap-2">
              <div className="text-xs font-semibold text-muted">{group.category}</div>
              {group.items.map((report) => (
                <button key={report.id} className={`rounded-lg border px-3 py-3 text-left ${report.id === selectedId ? "border-primary bg-blue-50" : "border-line"}`} onClick={() => selectReport(report)}>
                  <span className="block text-sm font-medium">{report.name}</span>
                  <span className="mt-1 inline-flex rounded-full bg-gray-100 px-2 py-0.5 text-xs text-muted">{reportTypeLabels[report.type || "custom"] || report.type}</span>
                  <span className="mt-1 block text-xs leading-5 text-muted">{report.description}</span>
                </button>
              ))}
            </section>
          ))}
        </div>
      </aside>

      <section className="grid min-w-0 gap-5">
        <div className="rounded-lg border border-line bg-surface p-4">
          <div className="mb-4 flex flex-wrap items-start justify-between gap-3">
            <div>
              <h2 className="text-xl font-semibold">{draft.name || "专题报表"}</h2>
              <p className="mt-1 max-w-3xl text-sm leading-6 text-muted">{draft.description || "选择左侧报表后查看统计结果。"}</p>
            </div>
            <div className="flex flex-wrap gap-2">
              <button className={`rounded-lg px-3 py-2 text-sm ${viewMode === "chart" ? "bg-primary text-white" : "border border-line bg-white"}`} onClick={() => setViewMode("chart")}>图表显示</button>
              <button className={`rounded-lg px-3 py-2 text-sm ${viewMode === "table" ? "bg-primary text-white" : "border border-line bg-white"}`} onClick={() => setViewMode("table")}>数据显示</button>
            </div>
          </div>
          <div className="grid gap-3 lg:grid-cols-[160px_minmax(160px,1fr)_minmax(220px,1.4fr)_auto]">
            <select className="rounded-lg border border-line px-3 py-2 text-sm" value={draft.type || "custom"} onChange={(event) => setDraft({ ...draft, type: event.target.value })}>
              {Object.entries(reportTypeLabels).map(([id, label]) => <option key={id} value={id}>{label}</option>)}
            </select>
            <input className="rounded-lg border border-line px-3 py-2 text-sm" placeholder="报表名称" value={draft.name} onChange={(event) => setDraft({ ...draft, name: event.target.value })} />
            <input className="rounded-lg border border-line px-3 py-2 text-sm" placeholder="描述" value={draft.description} onChange={(event) => setDraft({ ...draft, description: event.target.value })} />
            <button className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white" onClick={save}>保存</button>
          </div>
          {message && <div className="mt-3 rounded-lg bg-blue-50 px-3 py-2 text-sm text-primary">{message}</div>}
        </div>

        <div className="rounded-lg border border-line bg-surface p-4">
          <div className="mb-3 text-sm font-semibold">报表和查询条件</div>
          <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
            <label className="grid gap-1 text-sm"><span className="text-muted">开始日期</span><input type="date" className="rounded-lg border border-line px-3 py-2" value={filters.dateFrom} onChange={(event) => setFilters({ ...filters, dateFrom: event.target.value })} /></label>
            <label className="grid gap-1 text-sm"><span className="text-muted">结束日期</span><input type="date" className="rounded-lg border border-line px-3 py-2" value={filters.dateTo} onChange={(event) => setFilters({ ...filters, dateTo: event.target.value })} /></label>
            <label className="grid gap-1 text-sm"><span className="text-muted">科室</span><input className="rounded-lg border border-line px-3 py-2" placeholder="输入科室名称" value={filters.department} onChange={(event) => setFilters({ ...filters, department: event.target.value })} /></label>
            <label className="grid gap-1 text-sm"><span className="text-muted">渠道</span><select className="rounded-lg border border-line px-3 py-2" value={filters.channel} onChange={(event) => setFilters({ ...filters, channel: event.target.value })}><option value="">全部渠道</option><option value="web">Web</option><option value="wechat">微信</option><option value="sms">短信</option><option value="phone">电话</option><option value="tablet">平板</option><option value="qrcode">二维码</option></select></label>
            <label className="grid gap-1 text-sm"><span className="text-muted">医生/人员</span><input className="rounded-lg border border-line px-3 py-2" placeholder="后续用于岗位维度" value={filters.doctor} onChange={(event) => setFilters({ ...filters, doctor: event.target.value })} /></label>
            <label className="grid gap-1 text-sm"><span className="text-muted">就诊类型</span><select className="rounded-lg border border-line px-3 py-2" value={filters.visitType} onChange={(event) => setFilters({ ...filters, visitType: event.target.value })}><option value="">全部</option><option value="outpatient">门诊</option><option value="emergency">急诊</option><option value="inpatient">住院</option><option value="discharge">出院</option><option value="physical">体检</option></select></label>
            <label className="grid gap-1 text-sm"><span className="text-muted">题目 ID</span><input className="rounded-lg border border-line px-3 py-2" placeholder="question_id" value={filters.questionId} onChange={(event) => setFilters({ ...filters, questionId: event.target.value })} /></label>
            <div className="flex items-end gap-2">
              <button className="h-10 rounded-lg bg-primary px-4 text-sm font-medium text-white" disabled={!selectedId} onClick={() => selectedId && loadQuery(selectedId, projectId, filters)}>查询</button>
              <button className="h-10 rounded-lg border border-line px-4 text-sm" onClick={() => setFilters(defaultFilters)}>重置</button>
            </div>
          </div>
        </div>

        <div className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-line bg-surface p-3">
          <div className="flex flex-wrap gap-2">
          <button className="rounded-lg border border-line px-3 py-2 text-sm hover:border-primary" onClick={() => addWidget("bar")}>添加图表</button>
          <button className="rounded-lg border border-line px-3 py-2 text-sm hover:border-primary" onClick={() => addWidget("table")}>添加明细表</button>
          <button className="rounded-lg border border-line px-3 py-2 text-sm hover:border-primary" disabled={!selectedId} onClick={() => selectedId && loadQuery(selectedId, projectId, filters)}>刷新数据</button>
          </div>
          <div className="flex flex-wrap gap-2">
          <button className="rounded-lg border border-line px-3 py-2 text-sm hover:border-primary disabled:text-muted" disabled={!selectedId || query.rows.length === 0} onClick={exportExcel}>导出 Excel</button>
          <button className="rounded-lg border border-line px-3 py-2 text-sm hover:border-primary disabled:text-muted" disabled={!selectedId} onClick={() => exportReport("word")}>导出 Word</button>
          <button className="rounded-lg border border-line px-3 py-2 text-sm hover:border-primary disabled:text-muted" disabled={!selectedId} onClick={() => exportReport("pdf")}>导出 PDF</button>
          </div>
        </div>

        <div className="grid gap-3 md:grid-cols-3">
          <MetricCard label="记录数" value={query.rows.length} />
          <MetricCard label={query.measures[0] || "主指标"} value={summary.primary} />
          <MetricCard label="报表口径" value={draft.defaultDimension || draft.code || "-"} />
        </div>

        {insights && <div className="grid gap-3 rounded-lg border border-line bg-surface p-4">
          <h2 className="text-base font-semibold">AI 洞察</h2>
          <div className="flex flex-wrap gap-2">{insights.themes.map((item) => <span key={item} className="rounded-full bg-blue-50 px-3 py-1 text-xs text-primary">{item}</span>)}</div>
          <div className="grid gap-2 text-sm leading-6 text-muted">{[...insights.rootCauses, ...insights.suggestions].map((item) => <p key={item}>{item}</p>)}</div>
        </div>}

        <div className={viewMode === "chart" ? "grid min-w-0 gap-5" : "grid min-w-0 gap-5"}>
          {viewMode === "chart" && (
          <section className="min-w-0 rounded-lg border border-line bg-surface p-4">
            <div className="mb-3 flex items-center justify-between gap-3">
              <h2 className="truncate text-base font-semibold">{draft.widgets?.find((item) => item.type !== "table")?.title || draft.name || "报表图表"}</h2>
              <span className="shrink-0 rounded-full bg-gray-100 px-2 py-0.5 text-xs text-muted">{query.rows.length} 条</span>
            </div>
            <ReportChart data={query.rows} xField={xField} yField={yField} title="" />
          </section>
          )}
          {viewMode === "table" && (
          <section className="min-w-0 rounded-lg border border-line bg-surface p-4">
            <div className="mb-3 flex items-center justify-between gap-3">
              <h2 className="text-base font-semibold">报表明细</h2>
              <span className="shrink-0 rounded-full bg-gray-100 px-2 py-0.5 text-xs text-muted">可横向滚动</span>
            </div>
            <ResultsTable rows={query.rows} onRowClick={drilldown} />
          </section>
          )}
        </div>
        {drillRows.length > 0 && (
          <section className="min-w-0 rounded-lg border border-line bg-surface p-4">
            <div className="mb-3 flex items-center justify-between gap-3">
              <div>
                <h2 className="text-base font-semibold">答卷明细下钻</h2>
                <p className="mt-1 text-sm text-muted">{drillTitle} · {drillRows.length} 条</p>
              </div>
              <button className="rounded-lg border border-line px-3 py-2 text-sm" onClick={() => setDrillRows([])}>关闭</button>
            </div>
            <ResultsTable rows={drillRows} />
          </section>
        )}
      </section>
    </div>
  )
}

function currentProjectId() {
  return new URLSearchParams(window.location.search).get("projectId") || ""
}

function groupReports(reports: Report[]) {
  const groups = new Map<string, Report[]>()
  reports.forEach((report) => {
    const category = report.category || reportTypeLabels[report.type || "custom"] || "其他报表"
    groups.set(category, [...(groups.get(category) || []), report])
  })
  return Array.from(groups.entries()).map(([category, items]) => ({ category, items }))
}

function queryString(projectId: string, filters: ReportFilters, prefixWithAmp = false) {
  const params = new URLSearchParams()
  if (projectId) params.set("projectId", projectId)
  Object.entries(filters).forEach(([key, value]) => { if (value) params.set(key, value) })
  const text = params.toString()
  if (!text) return ""
  return `${prefixWithAmp ? "&" : "?"}${text}`
}

function summarizeRows(rows: Row[], measures: string[]) {
  const primaryMeasure = measures.find((item) => rows.some((row) => typeof row[item] === "number")) || measures[0]
  if (!primaryMeasure) return { primary: "-" }
  const values = rows.map((row) => Number(row[primaryMeasure])).filter((value) => Number.isFinite(value))
  if (values.length === 0) return { primary: "-" }
  const avg = values.reduce((sum, value) => sum + value, 0) / values.length
  return { primary: Number(avg.toFixed(2)) }
}

function stringValue(value: unknown) {
  if (value == null) return ""
  const text = String(value).trim()
  return text === "<nil>" ? "" : text
}

function MetricCard({ label, value }: { label: string; value: string | number }) {
  return <div className="rounded-lg border border-line bg-surface p-4">
    <div className="text-sm text-muted">{label}</div>
    <div className="mt-2 text-2xl font-semibold">{value}</div>
  </div>
}
