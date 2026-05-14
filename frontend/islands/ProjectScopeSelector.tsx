import { useEffect, useState } from "react"
import { authedJson } from "../lib/auth"

interface Project { id: string; name: string; targetType?: string; status?: string }

const targetLabels: Record<string, string> = {
  outpatient: "门诊",
  emergency: "急诊",
  inpatient: "住院",
  discharge: "出院",
  physical: "体检",
  staff: "员工",
}

export function ProjectScopeSelector() {
  const [projects, setProjects] = useState<Project[]>([])
  const [projectId, setProjectId] = useState("")
  const [message, setMessage] = useState("")

  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    const current = params.get("projectId") || ""
    setProjectId(current)
    loadProjects()
      .then(setProjects)
      .catch((error) => setMessage(error instanceof Error ? error.message : "项目列表加载失败"))
  }, [])

  function changeScope(nextProjectId: string) {
    setProjectId(nextProjectId)
    const url = new URL(window.location.href)
    if (nextProjectId) url.searchParams.set("projectId", nextProjectId)
    else url.searchParams.delete("projectId")
    window.history.replaceState({}, "", url)
    window.dispatchEvent(new CustomEvent("project-scope-change", { detail: { projectId: nextProjectId } }))
  }

  const current = projects.find((item) => item.id === projectId)

  return <section className="rounded-lg border border-line bg-surface p-4">
    <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
      <div>
        <h2 className="text-base font-semibold text-ink">项目范围</h2>
        <p className="mt-1 text-sm text-muted">
          数据中心默认按当前项目查看；有权限的用户可切换为全部项目进行横向分析。
        </p>
      </div>
      <label className="grid gap-1 text-sm lg:min-w-[360px]">
        <span className="font-medium text-muted">查看范围</span>
        <select className="h-11 rounded-lg border border-line bg-white px-3 text-ink outline-none focus:border-primary focus:ring-2 focus:ring-blue-100" value={projectId} onChange={(event) => changeScope(event.target.value)}>
          <option value="">全部项目</option>
          {projects.map((project) => <option key={project.id} value={project.id}>{project.name}</option>)}
        </select>
      </label>
    </div>
    <div className="mt-3 flex flex-wrap gap-2 text-xs text-muted">
      <span className="rounded-full bg-gray-100 px-2.5 py-1">当前：{current?.name || "全部项目"}</span>
      {current?.targetType && <span className="rounded-full bg-blue-50 px-2.5 py-1 text-primary">{targetLabels[current.targetType] || current.targetType}</span>}
      {current?.status && <span className="rounded-full bg-gray-100 px-2.5 py-1">{current.status}</span>}
      {message && <span className="rounded-full bg-red-50 px-2.5 py-1 text-red-600">{message}</span>}
    </div>
  </section>
}

function loadProjects() {
  return authedJson<Project[]>("/api/v1/projects").catch(() => authedJson<Project[]>("/api/v1/satisfaction/projects"))
}
