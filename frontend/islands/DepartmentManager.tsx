import { useEffect, useMemo, useState } from "react"
import { authedJson } from "../lib/auth"

interface Department {
  id: string
  code: string
  name: string
  kind: string
  status: string
  createdAt?: string
  updatedAt?: string
}

const empty: Department = { id: "", code: "", name: "", kind: "clinical", status: "active" }
const kindLabels: Record<string, string> = { clinical: "临床科室", nursing: "护理单元", medical_tech: "医技科室", admin: "职能部门", logistics: "后勤部门" }
const statusLabels: Record<string, string> = { active: "启用", inactive: "停用" }

export function DepartmentManager() {
  const [items, setItems] = useState<Department[]>([])
  const [draft, setDraft] = useState<Department>(empty)
  const [query, setQuery] = useState("")
  const [view, setView] = useState<"list" | "form">("list")
  const [message, setMessage] = useState("正在加载科室...")

  const filtered = useMemo(() => {
    const keyword = query.trim().toLowerCase()
    if (!keyword) return items
    return items.filter((item) => `${item.code} ${item.name} ${item.kind} ${item.status}`.toLowerCase().includes(keyword))
  }, [items, query])

  async function load() {
    const data = await authedJson<Department[]>("/api/v1/departments")
    setItems(data)
    setMessage("")
  }

  async function save() {
    const saved = await authedJson<Department>(draft.id ? `/api/v1/departments/${draft.id}` : "/api/v1/departments", {
      method: draft.id ? "PUT" : "POST",
      body: JSON.stringify(draft),
    })
    setDraft(saved)
    setMessage("科室已保存")
    await load()
    setView("list")
  }

  async function remove(item: Department) {
    if (!window.confirm(`删除科室「${item.name}」？如果已有用户或业务数据引用，数据库会阻止删除。`)) return
    await authedJson<Department>(`/api/v1/departments/${item.id}`, { method: "DELETE" })
    setMessage("科室已删除")
    await load()
  }

  function create() {
    setDraft(empty)
    setMessage("")
    setView("form")
  }

  function edit(item: Department) {
    setDraft(item)
    setMessage("")
    setView("form")
  }

  useEffect(() => {
    load().catch((error) => setMessage(`科室 API 未连接：${error instanceof Error ? error.message : "未知错误"}`))
  }, [])

  if (view === "form") {
    return <section className="rounded-lg border border-line bg-surface">
      <div className="flex flex-wrap items-center justify-between gap-3 border-b border-line p-4">
        <div>
          <h2 className="text-base font-semibold">{draft.id ? "编辑科室" : "新增科室"}</h2>
          <p className="mt-1 text-sm text-muted">科室编码建议与 HIS/EMR 科室编码保持一致，后续用户、报表和数据权限都会引用这里。</p>
        </div>
        <div className="flex gap-2">
          <button className="rounded-lg border border-line px-4 py-2 text-sm" onClick={() => setView("list")}>返回列表</button>
          <button className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white" onClick={save}>保存科室</button>
        </div>
      </div>
      {message && <div className="border-b border-line bg-blue-50 px-4 py-3 text-sm text-primary">{message}</div>}
      <div className="grid max-w-4xl gap-4 p-4 md:grid-cols-2">
        <label className="grid gap-1 text-sm">
          <span className="text-muted">科室编码</span>
          <input className="rounded-lg border border-line px-3 py-2" value={draft.code} onChange={(event) => setDraft({ ...draft, code: event.target.value })} placeholder="如 CARD" />
        </label>
        <label className="grid gap-1 text-sm">
          <span className="text-muted">科室名称</span>
          <input className="rounded-lg border border-line px-3 py-2" value={draft.name} onChange={(event) => setDraft({ ...draft, name: event.target.value })} placeholder="如 心内科" />
        </label>
        <label className="grid gap-1 text-sm">
          <span className="text-muted">科室类型</span>
          <select className="rounded-lg border border-line px-3 py-2" value={draft.kind} onChange={(event) => setDraft({ ...draft, kind: event.target.value })}>
            {Object.entries(kindLabels).map(([value, label]) => <option key={value} value={value}>{label}</option>)}
          </select>
        </label>
        <label className="grid gap-1 text-sm">
          <span className="text-muted">状态</span>
          <select className="rounded-lg border border-line px-3 py-2" value={draft.status} onChange={(event) => setDraft({ ...draft, status: event.target.value })}>
            {Object.entries(statusLabels).map(([value, label]) => <option key={value} value={value}>{label}</option>)}
          </select>
        </label>
      </div>
    </section>
  }

  return <section className="rounded-lg border border-line bg-surface">
    <div className="flex flex-wrap items-center justify-between gap-3 border-b border-line p-4">
      <div>
        <h2 className="text-base font-semibold">科室列表</h2>
        <p className="mt-1 text-sm text-muted">这里维护的科室会出现在用户所属科室、管理范围、随访方案、满意度报表和数据权限里。</p>
      </div>
      <button className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white" onClick={create}>新增科室</button>
    </div>
    {message && <div className="border-b border-line bg-blue-50 px-4 py-3 text-sm text-primary">{message}</div>}
    <div className="border-b border-line p-4">
      <input className="h-10 w-full rounded-lg border border-line px-3 text-sm" placeholder="搜索科室编码、名称、类型或状态" value={query} onChange={(event) => setQuery(event.target.value)} />
    </div>
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead className="bg-gray-50 text-xs uppercase text-muted">
          <tr>
            <th className="px-4 py-3 text-left">科室名称</th>
            <th className="px-4 py-3 text-left">编码</th>
            <th className="px-4 py-3 text-left">类型</th>
            <th className="px-4 py-3 text-left">状态</th>
            <th className="px-4 py-3 text-left">更新时间</th>
            <th className="px-4 py-3 text-right">操作</th>
          </tr>
        </thead>
        <tbody>
          {filtered.map((item) => <tr key={item.id} className="border-t border-line hover:bg-gray-50">
            <td className="px-4 py-3 font-medium">{item.name}</td>
            <td className="px-4 py-3 font-mono text-xs">{item.code}</td>
            <td className="px-4 py-3">{kindLabels[item.kind] || item.kind}</td>
            <td className="px-4 py-3"><span className="rounded-full bg-gray-100 px-2 py-1 text-xs text-muted">{statusLabels[item.status] || item.status}</span></td>
            <td className="px-4 py-3">{item.updatedAt?.slice(0, 10)}</td>
            <td className="px-4 py-3 text-right">
              <button className="rounded-lg border border-line px-3 py-1.5 text-xs text-primary hover:border-primary" onClick={() => edit(item)}>编辑</button>
              <button className="ml-2 rounded-lg border border-red-200 px-3 py-1.5 text-xs text-red-600 hover:bg-red-50" onClick={() => remove(item)}>删除</button>
            </td>
          </tr>)}
          {!filtered.length && <tr><td colSpan={6} className="px-4 py-8 text-center text-muted">暂无科室。</td></tr>}
        </tbody>
      </table>
    </div>
  </section>
}
