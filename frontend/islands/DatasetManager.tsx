import { useEffect, useMemo, useState } from "react"
import { authedJson } from "../lib/auth"

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
const datasetFlow = [
  { title: "数据源接入", text: "从 HIS、EMR、随访、投诉、检查检验等系统同步或导入。" },
  { title: "数据模板", text: "定义患者、就诊、用药、病史、检查、随访记录的字段口径。" },
  { title: "项目绑定", text: "项目选择数据范围，表单自动带入患者和就诊信息。" },
  { title: "分析报表", text: "按项目权限汇聚到数据中心、分析报表和患者 360。" },
]

export function DatasetManager() {
  const [datasets, setDatasets] = useState<Dataset[]>([])
  const [keyword, setKeyword] = useState("")
  const [draft, setDraft] = useState<Dataset>(emptyDataset)
  const [selectedId, setSelectedId] = useState("")
  const [view, setView] = useState<"list" | "form">("list")
  const [message, setMessage] = useState("正在连接数据集 API...")
  const selected = useMemo(() => datasets.find((dataset) => dataset.id === selectedId), [datasets, selectedId])

  async function loginAndLoad() {
    try {
      setDatasets(await authedJson<Dataset[]>("/api/v1/datasets"))
      setMessage("")
    } catch (error) {
      setMessage(`数据集 API 未连接：${error instanceof Error ? error.message : "未知错误"}`)
    }
  }

  async function search() {
    try {
      setDatasets(await authedJson<Dataset[]>(`/api/v1/datasets?q=${encodeURIComponent(keyword)}`))
      setMessage("")
    } catch (error) {
      setMessage(`搜索失败：${error instanceof Error ? error.message : "未知错误"}`)
    }
  }

  async function save() {
    try {
      const method = selectedId ? "PUT" : "POST"
      const path = selectedId ? `/api/v1/datasets/${selectedId}` : "/api/v1/datasets"
      const dataset = await authedJson<Dataset>(path, { method, body: JSON.stringify(draft) })
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
      await authedJson<Dataset>(`/api/v1/datasets/${id}`, { method: "DELETE" })
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
        <div className="rounded-lg border border-line bg-surface p-4">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <h2 className="text-base font-semibold">数据模板闭环</h2>
              <p className="mt-1 text-sm text-muted">数据集不是孤立表格，它负责把系统字段、项目数据范围和分析报表串起来。</p>
            </div>
            <button className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white" onClick={newDataset}>新建数据模板</button>
          </div>
          <div className="mt-4 grid gap-3 md:grid-cols-4">
            {datasetFlow.map((step, index) => <FlowStep key={step.title} index={index + 1} title={step.title} text={step.text} />)}
          </div>
        </div>
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
              新建数据模板
            </button>
          </div>
          {message && <div className="border-b border-line bg-blue-50 px-4 py-3 text-sm text-primary">{message}</div>}
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="bg-gray-50 text-xs uppercase text-muted">
                <tr>
                  <th className="px-4 py-3 text-left">数据模板 / 数据集</th>
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
                    <td className="px-4 py-8 text-center text-sm text-muted" colSpan={7}>暂无数据模板。先定义数据范围，再给项目绑定使用。</td>
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
            <h2 className="text-base font-semibold">{selected ? "编辑数据模板 / 数据集" : "新建数据模板 / 数据集"}</h2>
            <p className="mt-1 text-sm text-muted">维护数据口径、负责人、记录规模和状态，后续由项目引用。</p>
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
          <div className="rounded-lg border border-line bg-gray-50 p-3 text-sm leading-6 text-muted md:col-span-2 xl:col-span-3">
            建议口径：患者主索引、就诊记录、用药记录、既往史、检查记录、检验结果、投诉评价、随访记录。项目“表单数据”选择数据范围后，公开问卷和电话随访页会自动拉取。
          </div>
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

function FlowStep({ index, title, text }: { index: number; title: string; text: string }) {
  return <div className="rounded-lg border border-line bg-white p-3">
    <div className="flex items-center gap-2"><span className="grid h-6 w-6 place-items-center rounded-full bg-blue-50 text-xs font-semibold text-primary">{index}</span><span className="font-medium">{title}</span></div>
    <div className="mt-2 text-xs leading-5 text-muted">{text}</div>
  </div>
}
