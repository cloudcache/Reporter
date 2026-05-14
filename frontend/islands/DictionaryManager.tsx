import { useEffect, useMemo, useState } from "react"
import { authedJson } from "../lib/auth"

interface Entry { key: string; label: string; value: string }
interface Dict { id: string; code: string; name: string; category: string; description?: string; items: Entry[] }

const empty: Dict = { id: "", code: "", name: "", category: "电子病历", description: "", items: [] }
const preferredCategories = ["全部", "电子病历", "病例管理", "就诊信息", "用药信息", "患者基础", "随访中心"]

export function DictionaryManager() {
  const [items, setItems] = useState<Dict[]>([])
  const [draft, setDraft] = useState<Dict>(empty)
  const [raw, setRaw] = useState("")
  const [view, setView] = useState<"list" | "editor">("list")
  const [query, setQuery] = useState("")
  const [category, setCategory] = useState("全部")
  const [message, setMessage] = useState("正在加载字典...")

  const categories = useMemo(() => {
    const fromData = Array.from(new Set(items.map((item) => item.category).filter(Boolean)))
    return Array.from(new Set([...preferredCategories, ...fromData]))
  }, [items])

  const filteredItems = useMemo(() => {
    const keyword = query.trim().toLowerCase()
    return items.filter((item) => {
      const categoryMatched = category === "全部" || item.category === category
      if (!keyword) return categoryMatched
      const text = `${item.code} ${item.name} ${item.category} ${item.description || ""} ${item.items.map((entry) => `${entry.key} ${entry.label} ${entry.value}`).join(" ")}`.toLowerCase()
      return categoryMatched && text.includes(keyword)
    })
  }, [category, items, query])

  async function load() {
    setItems(await authedJson<Dict[]>("/api/v1/dictionaries"))
    setMessage("")
  }

  async function save() {
    const body = { ...draft, items: parseEntries(raw) }
    const saved = await authedJson<Dict>(draft.id ? `/api/v1/dictionaries/${draft.id}` : "/api/v1/dictionaries", { method: draft.id ? "PUT" : "POST", body: JSON.stringify(body) })
    setDraft(saved)
    setRaw(formatEntries(saved.items))
    setMessage("字典已保存")
    await load()
    setView("list")
  }

  function create() {
    setDraft(empty)
    setRaw("")
    setMessage("")
    setView("editor")
  }

  function edit(item: Dict) {
    setDraft(item)
    setRaw(formatEntries(item.items))
    setMessage("")
    setView("editor")
  }

  useEffect(() => { load().catch((e) => setMessage(e instanceof Error ? e.message : "加载失败")) }, [])

  if (view === "editor") {
    return <DictionaryEditor draft={draft} raw={raw} setDraft={setDraft} setRaw={setRaw} save={save} back={() => setView("list")} />
  }

  return <section className="rounded-lg border border-line bg-surface">
    <div className="flex flex-wrap items-center justify-between gap-3 border-b border-line p-4">
      <div>
        <h2 className="text-base font-semibold">字典列表</h2>
        <p className="mt-1 text-sm text-muted">按业务域维护字段和值域字典，电子病历、病例、就诊、用药字段已内置并落库。</p>
      </div>
      <button className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white" onClick={create}>新增字典</button>
    </div>
    {message && <div className="border-b border-line bg-blue-50 px-4 py-3 text-sm text-primary">{message}</div>}
    <div className="grid gap-3 border-b border-line p-4 lg:grid-cols-[220px_minmax(0,1fr)]">
      <select className="h-10 rounded-lg border border-line px-3 text-sm" value={category} onChange={(event) => setCategory(event.target.value)}>
        {categories.map((item) => <option key={item} value={item}>{item}</option>)}
      </select>
      <input className="h-10 rounded-lg border border-line px-3 text-sm" placeholder="搜索编码、名称、分类、字段 key 或中文标签" value={query} onChange={(event) => setQuery(event.target.value)} />
    </div>
    <div className="grid gap-3 p-4 lg:grid-cols-2">
      {filteredItems.map((item) => <DictionaryCard key={item.id} item={item} edit={() => edit(item)} />)}
      {!filteredItems.length && <div className="rounded-lg border border-dashed border-line bg-gray-50 p-6 text-sm text-muted">没有匹配的字典。</div>}
    </div>
  </section>
}

function DictionaryEditor({ draft, raw, setDraft, setRaw, save, back }: { draft: Dict; raw: string; setDraft: (value: Dict) => void; setRaw: (value: string) => void; save: () => void; back: () => void }) {
  const parsed = parseEntries(raw)
  return <section className="rounded-lg border border-line bg-surface">
    <div className="flex flex-wrap items-center justify-between gap-3 border-b border-line p-4">
      <div>
        <h2 className="text-base font-semibold">{draft.id ? "编辑字典" : "新增字典"}</h2>
        <p className="mt-1 text-sm text-muted">字典项按每行 `key,label,value` 录入，保存后进入数据库并可被数据映射、表单和随访配置复用。</p>
      </div>
      <div className="flex gap-2">
        <button className="rounded-lg border border-line px-4 py-2 text-sm" onClick={back}>返回列表</button>
        <button className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white" onClick={save}>保存字典</button>
      </div>
    </div>
    <div className="grid gap-5 p-4 xl:grid-cols-[minmax(0,1fr)_360px]">
      <div className="grid gap-4">
        <div className="grid gap-3 md:grid-cols-2">
          <Text label="编码" value={draft.code} onChange={(value) => setDraft({ ...draft, code: value })} />
          <Text label="名称" value={draft.name} onChange={(value) => setDraft({ ...draft, name: value })} />
          <Text label="分类" value={draft.category} onChange={(value) => setDraft({ ...draft, category: value })} />
          <Text label="说明" value={draft.description || ""} onChange={(value) => setDraft({ ...draft, description: value })} />
        </div>
        <label className="grid gap-1">
          <span className="text-sm font-medium text-muted">字典项</span>
          <textarea className="min-h-[420px] rounded-lg border border-line px-3 py-2 font-mono text-xs leading-6" value={raw} onChange={(event) => setRaw(event.target.value)} placeholder="record_no,病历号,record_no&#10;chief_complaint,主诉,chief_complaint" />
        </label>
      </div>
      <aside className="rounded-lg border border-line bg-gray-50 p-4">
        <h3 className="font-semibold">录入预览</h3>
        <p className="mt-1 text-sm text-muted">共 {parsed.length} 项，保存前可核对 key 和中文标签。</p>
        <div className="mt-3 max-h-[520px] overflow-y-auto rounded-lg border border-line bg-white">
          {parsed.map((entry) => <div key={`${entry.key}-${entry.value}`} className="border-b border-line px-3 py-2 last:border-0">
            <div className="text-sm font-medium">{entry.label}</div>
            <div className="mt-1 break-all font-mono text-xs text-muted">{entry.key} = {entry.value}</div>
          </div>)}
          {!parsed.length && <div className="p-4 text-sm text-muted">暂无字典项。</div>}
        </div>
      </aside>
    </div>
  </section>
}

function DictionaryCard({ item, edit }: { item: Dict; edit: () => void }) {
  return <button className="rounded-lg border border-line p-4 text-left hover:border-primary hover:bg-blue-50/40" onClick={edit}>
    <div className="flex flex-wrap items-start justify-between gap-3">
      <div>
        <div className="font-semibold">{item.name}</div>
        <div className="mt-1 font-mono text-xs text-muted">{item.code}</div>
      </div>
      <span className="rounded-full bg-gray-100 px-2 py-1 text-xs text-muted">{item.category}</span>
    </div>
    {item.description && <p className="mt-3 line-clamp-2 text-sm text-muted">{item.description}</p>}
    <div className="mt-3 flex flex-wrap gap-2">
      {item.items.slice(0, 6).map((entry) => <span key={entry.key} className="rounded-md border border-line bg-white px-2 py-1 text-xs text-muted">{entry.label}</span>)}
      {item.items.length > 6 && <span className="rounded-md bg-gray-100 px-2 py-1 text-xs text-muted">+{item.items.length - 6}</span>}
    </div>
  </button>
}

function parseEntries(raw: string): Entry[] {
  return raw.split("\n").map((line) => line.trim()).filter(Boolean).map((line) => {
    const [key, label, value] = line.split(",").map((part) => part.trim())
    return { key, label: label || key, value: value || key }
  }).filter((item) => item.key)
}

function formatEntries(items: Entry[]) {
  return (items || []).map((item) => `${item.key},${item.label},${item.value}`).join("\n")
}

function Text({ label, value, onChange }: { label: string; value: string; onChange: (value: string) => void }) {
  return <label className="grid gap-1"><span className="text-sm font-medium text-muted">{label}</span><input className="h-10 rounded-lg border border-line px-3 text-sm" value={value} onChange={(event) => onChange(event.target.value)} /></label>
}
