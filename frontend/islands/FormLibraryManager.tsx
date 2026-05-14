import { useEffect, useState } from "react"
import { authedJson } from "../lib/auth"

interface Item { id: string; kind: string; label: string; hint: string; scenario?: string; components: Array<Record<string, unknown>>; sortOrder: number; enabled?: boolean }
interface Library { templates: Item[]; commonComponents: Item[]; atomicComponents: Item[] }

const empty: Item = { id: "", kind: "common", label: "", hint: "", scenario: "", components: [], sortOrder: 100, enabled: true }

export function FormLibraryManager() {
  const [items, setItems] = useState<Item[]>([])
  const [draft, setDraft] = useState<Item>(empty)
  const [json, setJson] = useState("[]")
  const [message, setMessage] = useState("正在加载组件库...")

  async function load() {
    const data = await authedJson<Library>("/api/v1/form-library")
    setItems([...(data.templates || []), ...(data.commonComponents || []), ...(data.atomicComponents || [])])
    setMessage("")
  }

  async function save() {
    const body = { ...draft, components: JSON.parse(json || "[]") }
    const saved = await authedJson<Item>(draft.id ? `/api/v1/form-library/${draft.id}` : "/api/v1/form-library", { method: draft.id ? "PUT" : "POST", body: JSON.stringify(body) })
    setDraft(saved)
    setJson(JSON.stringify(saved.components, null, 2))
    setMessage("组件库条目已保存")
    await load()
  }

  function edit(item: Item) {
    setDraft(item)
    setJson(JSON.stringify(item.components || [], null, 2))
  }

  useEffect(() => { load().catch((error) => setMessage(error instanceof Error ? error.message : "加载失败")) }, [])

  return (
    <div className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_460px]">
      <section className="rounded-lg border border-line bg-surface">
        <div className="flex items-center justify-between border-b border-line p-4"><h2 className="text-base font-semibold">表单组件库</h2><button className="rounded-lg border border-line px-3 py-2 text-sm" onClick={() => { setDraft(empty); setJson("[]") }}>新增</button></div>
        {message && <div className="border-b border-line bg-blue-50 px-4 py-3 text-sm text-primary">{message}</div>}
        <div className="grid gap-2 p-4">{items.map((item) => <button key={item.id} className="rounded-lg border border-line px-3 py-2 text-left hover:border-primary" onClick={() => edit(item)}><span className="font-medium">{item.label}</span><span className="ml-2 text-xs text-muted">{item.kind} {item.scenario}</span><span className="mt-1 block text-xs text-muted">{item.hint}</span></button>)}</div>
      </section>
      <aside className="rounded-lg border border-line bg-surface p-4">
        <h2 className="text-base font-semibold">{draft.id ? "编辑条目" : "新增条目"}</h2>
        <div className="mt-4 grid gap-3 text-sm">
          <Text label="ID" value={draft.id} onChange={(v) => setDraft({ ...draft, id: v })} />
          <Text label="名称" value={draft.label} onChange={(v) => setDraft({ ...draft, label: v })} />
          <Text label="说明" value={draft.hint} onChange={(v) => setDraft({ ...draft, hint: v })} />
          <label className="grid gap-1"><span className="text-muted">类型</span><select className="rounded-lg border border-line px-3 py-2" value={draft.kind} onChange={(e) => setDraft({ ...draft, kind: e.target.value })}><option value="template">业务模板</option><option value="common">公共组件</option><option value="atom">原子组件</option></select></label>
          <Text label="场景" value={draft.scenario || ""} onChange={(v) => setDraft({ ...draft, scenario: v })} />
          <label className="grid gap-1"><span className="text-muted">组件 JSON</span><textarea className="min-h-72 rounded-lg border border-line px-3 py-2 font-mono text-xs" value={json} onChange={(e) => setJson(e.target.value)} /></label>
          <button className="rounded-lg bg-primary px-4 py-2 font-medium text-white" onClick={save}>保存</button>
        </div>
      </aside>
    </div>
  )
}

function Text({ label, value, onChange }: { label: string; value: string; onChange: (value: string) => void }) {
  return <label className="grid gap-1"><span className="text-muted">{label}</span><input className="rounded-lg border border-line px-3 py-2" value={value} onChange={(e) => onChange(e.target.value)} /></label>
}
