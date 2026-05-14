import { useEffect, useMemo, useState } from "react"
import { apiBase, requireSession } from "../lib/auth"

interface Dataset {
  id: string
  name: string
  description: string
  owner: string
  recordCount: number
  formCount: number
  status: "active" | "archived"
  updatedAt?: string
}

const emptyDataset: Dataset = {
  id: "",
  name: "",
  description: "",
  owner: "",
  recordCount: 0,
  formCount: 0,
  status: "active",
}

const statusLabel = {
  active: "活跃",
  archived: "归档",
}

export function DatasetManager() {
  const [datasets, setDatasets] = useState<Dataset[]>([])
  const [keyword, setKeyword] = useState("")
  const [draft, setDraft] = useState<Dataset>(emptyDataset)
  const [selectedId, setSelectedId] = useState("")
  const [view, setView] = useState<"list" | "form">("list")
  const [message, setMessage] = useState("正在连接数据集 API...")
  const selected = useMemo(() => datasets.find((dataset) => dataset.id === selectedId), [datasets, selectedId])

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
      const list = await fetch(`${apiBase}/api/v1/datasets`, {
        credentials: "include",
      })
      if (!list.ok) throw new Error(await list.text())
      setDatasets(await list.json())
      setMessage("")
    } catch (error) {
      setMessage(`数据集 API 未连接：${error instanceof Error ? error.message : "未知错误"}`)
    }
  }

  async function search() {
    try {
      setDatasets(await api<Dataset[]>(`/api/v1/datasets?q=${encodeURIComponent(keyword)}`))
      setMessage("")
    } catch (error) {
      setMessage(`搜索失败：${error instanceof Error ? error.message : "未知错误"}`)
    }
  }

  async function save() {
    try {
      const method = selectedId ? "PUT" : "POST"
      const path = selectedId ? `/api/v1/datasets/${selectedId}` : "/api/v1/datasets"
      const dataset = await api<Dataset>(path, { method, body: JSON.stringify(draft) })
      if (selectedId) {
        setDatasets(datasets.map((item) => item.id === selectedId ? dataset : item))
      } else {
        setDatasets([...datasets, dataset])
      }
      setDraft(dataset)
      setSelectedId(dataset.id)
      setView("list")
      setMessage("已保存数据集")
    } catch (error) {
      setMessage(`保存失败：${error instanceof Error ? error.message : "未知错误"}`)
    }
  }

  async function removeDataset(id: string) {
    const dataset = datasets.find((item) => item.id === id)
    if (!dataset || !window.confirm(`删除数据集「${dataset.name}」？`)) return
    try {
      await api<Dataset>(`/api/v1/datasets/${id}`, { method: "DELETE" })
      setDatasets(datasets.filter((item) => item.id !== id))
      if (selectedId === id) {
        setSelectedId("")
        setDraft(emptyDataset)
      }
      setMessage("已删除数据集")
    } catch (error) {
      setMessage(`删除失败：${error instanceof Error ? error.message : "未知错误"}`)
    }
  }

  function selectDataset(dataset: Dataset) {
    setSelectedId(dataset.id)
    setDraft(dataset)
    setView("form")
    setMessage("")
  }

  function newDataset() {
    setSelectedId("")
    setDraft(emptyDataset)
    setView("form")
    setMessage("")
  }

  function backToList() {
    setSelectedId("")
    setDraft(emptyDataset)
    setView("list")
  }

  useEffect(() => {
    loginAndLoad()
  }, [])

  const totalRecords = datasets.reduce((sum, dataset) => sum + dataset.recordCount, 0)
  const totalForms = datasets.reduce((sum, dataset) => sum + dataset.formCount, 0)

  return (
    <div className="grid gap-5">
      {view === "list" && (
      <section className="grid gap-5">
        <div className="grid gap-4 md:grid-cols-3">
          <article className="rounded-lg border border-line bg-surface p-4">
            <p className="text-sm text-muted">数据集总数</p>
            <strong className="mt-2 block text-2xl font-semibold">{datasets.length}</strong>
          </article>
          <article className="rounded-lg border border-line bg-surface p-4">
            <p className="text-sm text-muted">总记录数</p>
            <strong className="mt-2 block text-2xl font-semibold">{totalRecords.toLocaleString()}</strong>
          </article>
          <article className="rounded-lg border border-line bg-surface p-4">
            <p className="text-sm text-muted">表单模板数</p>
            <strong className="mt-2 block text-2xl font-semibold">{totalForms}</strong>
          </article>
        </div>

        <div className="rounded-lg border border-line bg-surface">
          <div className="flex flex-wrap items-center justify-between gap-3 border-b border-line p-4">
            <div className="flex gap-2">
              <input className="h-10 w-72 rounded-lg border border-line bg-gray-50 px-3 text-sm outline-none focus:border-primary focus:bg-white" placeholder="名称、描述、负责人" value={keyword} onChange={(event) => setKeyword(event.target.value)} />
              <button className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white" onClick={search}>搜索</button>
            </div>
            <button className="rounded-lg border border-line px-4 py-2 text-sm font-medium hover:border-primary" onClick={newDataset}>
              新建数据集
            </button>
          </div>
          {message && <div className="border-b border-line bg-blue-50 px-4 py-3 text-sm text-primary">{message}</div>}
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="bg-gray-50 text-xs uppercase text-muted">
                <tr>
                  <th className="px-4 py-3 text-left">名称</th>
                  <th className="px-4 py-3 text-left">负责人</th>
                  <th className="px-4 py-3 text-left">记录数</th>
                  <th className="px-4 py-3 text-left">表单数</th>
                  <th className="px-4 py-3 text-left">状态</th>
                  <th className="px-4 py-3 text-left">更新时间</th>
                  <th className="px-4 py-3 text-right">操作</th>
                </tr>
              </thead>
              <tbody>
                {datasets.map((dataset) => (
                  <tr key={dataset.id} className={`cursor-pointer border-t border-line hover:bg-gray-50 ${dataset.id === selectedId ? "bg-blue-50" : ""}`} onClick={() => selectDataset(dataset)}>
                    <td className="px-4 py-3">
                      <div className="font-medium text-ink">{dataset.name}</div>
                      <div className="mt-1 text-xs text-muted">{dataset.description}</div>
                    </td>
                    <td className="px-4 py-3">{dataset.owner}</td>
                    <td className="px-4 py-3">{dataset.recordCount.toLocaleString()}</td>
                    <td className="px-4 py-3">{dataset.formCount}</td>
                    <td className="px-4 py-3"><span className="rounded-full bg-gray-100 px-2 py-1 text-xs">{statusLabel[dataset.status]}</span></td>
                    <td className="px-4 py-3">{dataset.updatedAt?.slice(0, 10)}</td>
                    <td className="px-4 py-3 text-right">
                      <button
                        className="rounded-lg border border-red-200 px-3 py-1.5 text-xs font-medium text-red-600 hover:bg-red-50"
                        onClick={(event) => {
                          event.stopPropagation()
                          removeDataset(dataset.id)
                        }}
                      >
                        删除
                      </button>
                    </td>
                  </tr>
                ))}
                {datasets.length === 0 && (
                  <tr>
                    <td className="px-4 py-8 text-center text-sm text-muted" colSpan={7}>暂无数据集</td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      </section>
      )}

      {view === "form" && (
      <section className="rounded-lg border border-line bg-surface">
        <div className="flex items-center justify-between gap-3 border-b border-line p-4">
          <div>
            <h2 className="text-base font-semibold">{selected ? "编辑数据集" : "新建数据集"}</h2>
            <p className="mt-1 text-sm text-muted">维护研究数据集、负责人、记录规模和状态。</p>
          </div>
          <div className="flex gap-2">
            <button className="rounded-lg border border-line px-4 py-2 text-sm hover:border-primary" onClick={backToList}>返回列表</button>
            <button className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white" onClick={save}>保存</button>
          </div>
        </div>
        {message && <div className="border-b border-line bg-blue-50 px-4 py-3 text-sm text-primary">{message}</div>}
        <div className="grid max-w-5xl gap-5 p-4 text-sm md:grid-cols-2 xl:grid-cols-3">
          <label className="grid gap-1">
            <span className="text-muted">名称</span>
            <input className="rounded-lg border border-line px-3 py-2" value={draft.name} onChange={(event) => setDraft({ ...draft, name: event.target.value })} />
          </label>
          {selectedId && (
            <label className="grid gap-1">
              <span className="text-muted">数据集 ID</span>
              <input className="rounded-lg border border-line bg-gray-50 px-3 py-2 text-muted" value={selectedId} readOnly />
            </label>
          )}
          <label className="grid gap-1">
            <span className="text-muted">描述</span>
            <textarea className="min-h-24 rounded-lg border border-line px-3 py-2" value={draft.description} onChange={(event) => setDraft({ ...draft, description: event.target.value })} />
          </label>
          <label className="grid gap-1">
            <span className="text-muted">负责人/科室</span>
            <input className="rounded-lg border border-line px-3 py-2" value={draft.owner} onChange={(event) => setDraft({ ...draft, owner: event.target.value })} />
          </label>
          <div className="grid grid-cols-2 gap-2">
            <label className="grid gap-1">
              <span className="text-muted">记录数</span>
              <input type="number" className="rounded-lg border border-line px-3 py-2" value={draft.recordCount} onChange={(event) => setDraft({ ...draft, recordCount: Number(event.target.value) })} />
            </label>
            <label className="grid gap-1">
              <span className="text-muted">表单数</span>
              <input type="number" className="rounded-lg border border-line px-3 py-2" value={draft.formCount} onChange={(event) => setDraft({ ...draft, formCount: Number(event.target.value) })} />
            </label>
          </div>
          <label className="grid gap-1">
            <span className="text-muted">状态</span>
            <select className="rounded-lg border border-line px-3 py-2" value={draft.status} onChange={(event) => setDraft({ ...draft, status: event.target.value as Dataset["status"] })}>
              <option value="active">活跃</option>
              <option value="archived">归档</option>
            </select>
          </label>
          {selectedId && (
            <button className="rounded-lg border border-red-200 px-4 py-2 text-sm font-medium text-red-600 hover:bg-red-50" onClick={() => removeDataset(selectedId)}>删除数据集</button>
          )}
        </div>
      </section>
      )}
    </div>
  )
}
