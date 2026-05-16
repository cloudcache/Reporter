import { useEffect, useState } from "react"
import { authedFetch, authedJson } from "../lib/auth"

interface ExportJob {
  id: string
  reportId: string
  projectId?: string
  exportType: string
  status: string
  filePath?: string
  errorMessage?: string
  createdBy?: string
  createdAt: string
  finishedAt?: string
}

interface ReportDefinition {
  id: string
  code?: string
  name: string
  category?: string
}

export function ExportCenterManager() {
  const [jobs, setJobs] = useState<ExportJob[]>([])
  const [reports, setReports] = useState<ReportDefinition[]>([])
  const [reportId, setReportId] = useState("")
  const [exportType, setExportType] = useState("excel")
  const [message, setMessage] = useState("")

  async function load() {
    try {
      const [nextReports, nextJobs] = await Promise.all([
        authedJson<ReportDefinition[]>("/api/v1/reports/definitions"),
        authedJson<ExportJob[]>("/api/v1/report-export-jobs"),
      ])
      setReports(nextReports)
      setJobs(nextJobs)
      if (!reportId && nextReports[0]) setReportId(nextReports[0].id)
      setMessage("")
    } catch (error) {
      setMessage(`导出中心加载失败：${error instanceof Error ? error.message : "未知错误"}`)
    }
  }

  async function createJob() {
    if (!reportId) return
    try {
      await authedJson<ExportJob>("/api/v1/reports/export", {
        method: "POST",
        body: JSON.stringify({ reportId, exportType, filters: {} }),
      })
      await load()
      setMessage("已创建导出任务")
    } catch (error) {
      setMessage(`创建导出任务失败：${error instanceof Error ? error.message : "未知错误"}`)
    }
  }

  async function download(job: ExportJob) {
    try {
      const response = await authedFetch(`/api/v1/report-export-jobs/${job.id}/download`)
      if (!response.ok) throw new Error(await response.text())
      const blob = await response.blob()
      const url = URL.createObjectURL(blob)
      const link = document.createElement("a")
      link.href = url
      link.download = fileNameFromDisposition(response.headers.get("Content-Disposition")) || `${reportName(reports, job.reportId)}.${job.exportType === "word" ? "doc" : job.exportType === "image" ? "svg" : job.exportType === "pdf" ? "pdf" : "csv"}`
      link.click()
      URL.revokeObjectURL(url)
    } catch (error) {
      setMessage(`下载失败：${error instanceof Error ? error.message : "未知错误"}`)
    }
  }

  useEffect(() => { load() }, [])

  return (
    <div className="grid gap-5">
      <section className="rounded-lg border border-line bg-surface p-4">
        <div className="flex flex-wrap items-end justify-between gap-3">
          <div>
            <h2 className="text-lg font-semibold">新建导出任务</h2>
            <p className="mt-1 text-sm text-muted">统一记录 Excel、图片、PDF、Word 导出任务和历史。</p>
          </div>
          <div className="grid flex-1 gap-3 md:max-w-3xl md:grid-cols-[minmax(220px,1fr)_160px_auto]">
            <select className="rounded-lg border border-line px-3 py-2 text-sm" value={reportId} onChange={(event) => setReportId(event.target.value)}>
              {reports.map((report) => <option key={report.id} value={report.id}>{report.name}</option>)}
            </select>
            <select className="rounded-lg border border-line px-3 py-2 text-sm" value={exportType} onChange={(event) => setExportType(event.target.value)}>
              <option value="excel">Excel</option>
              <option value="image">图片</option>
              <option value="pdf">PDF</option>
              <option value="word">Word</option>
            </select>
            <button className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white" onClick={createJob}>创建任务</button>
          </div>
        </div>
        {message && <div className="mt-3 rounded-lg bg-blue-50 px-3 py-2 text-sm text-primary">{message}</div>}
      </section>

      <section className="rounded-lg border border-line bg-surface">
        <div className="border-b border-line p-4">
          <h2 className="text-lg font-semibold">导出历史</h2>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full min-w-[880px] text-left text-sm">
            <thead className="bg-gray-50 text-muted">
              <tr>
                <th className="px-4 py-3">报表</th>
                <th className="px-4 py-3">类型</th>
                <th className="px-4 py-3">状态</th>
                <th className="px-4 py-3">创建人</th>
                <th className="px-4 py-3">创建时间</th>
                <th className="px-4 py-3">完成时间</th>
                <th className="px-4 py-3">文件</th>
                <th className="px-4 py-3">错误</th>
                <th className="px-4 py-3">操作</th>
              </tr>
            </thead>
            <tbody>
              {jobs.map((job) => (
                <tr key={job.id} className="border-t border-line">
                  <td className="px-4 py-3">{reportName(reports, job.reportId)}</td>
                  <td className="px-4 py-3">{job.exportType}</td>
                  <td className="px-4 py-3"><span className="rounded-full bg-gray-100 px-2 py-1 text-xs">{statusLabel(job.status)}</span></td>
                  <td className="px-4 py-3">{job.createdBy || "-"}</td>
                  <td className="px-4 py-3">{formatTime(job.createdAt)}</td>
                  <td className="px-4 py-3">{job.finishedAt ? formatTime(job.finishedAt) : "-"}</td>
                  <td className="px-4 py-3">{job.filePath || "-"}</td>
                  <td className="px-4 py-3">{job.errorMessage || "-"}</td>
                  <td className="px-4 py-3"><button className="rounded-md border border-line px-2 py-1 text-xs disabled:text-muted" disabled={job.status !== "success"} onClick={() => download(job)}>下载</button></td>
                </tr>
              ))}
              {jobs.length === 0 && <tr><td className="px-4 py-8 text-center text-muted" colSpan={9}>暂无导出任务</td></tr>}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  )
}

function fileNameFromDisposition(value: string | null) {
  const match = value?.match(/filename="([^"]+)"/)
  return match?.[1] || ""
}

function reportName(reports: ReportDefinition[], id: string) {
  return reports.find((report) => report.id === id || report.code === id)?.name || id
}

function statusLabel(status: string) {
  return ({ pending: "等待中", running: "生成中", success: "已完成", failed: "失败" } as Record<string, string>)[status] || status
}

function formatTime(value: string) {
  if (!value) return "-"
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}
