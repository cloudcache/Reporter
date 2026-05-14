import { useEffect, useState } from "react"
import { authedJson } from "../lib/auth"

interface Entry { key: string; label: string; value: string }
interface Dict { id: string; code: string; name: string; category: string; description?: string; items: Entry[] }
const empty: Dict = { id: "", code: "", name: "", category: "患者基础", description: "", items: [] }

export function DictionaryManager() {
  const [items, setItems] = useState<Dict[]>([])
  const [draft, setDraft] = useState<Dict>(empty)
  const [raw, setRaw] = useState("")
  const [message, setMessage] = useState("正在加载字典...")

  async function load() {
    setItems(await authedJson<Dict[]>("/api/v1/dictionaries"))
    setMessage("")
  }
  async function save() {
    const body = { ...draft, items: raw.split("\n").map((line) => line.split(",")).filter((p) => p[0]).map((p) => ({ key: p[0]?.trim(), label: p[1]?.trim() || p[0]?.trim(), value: p[2]?.trim() || p[0]?.trim() })) }
    const saved = await authedJson<Dict>(draft.id ? `/api/v1/dictionaries/${draft.id}` : "/api/v1/dictionaries", { method: draft.id ? "PUT" : "POST", body: JSON.stringify(body) })
    setDraft(saved); setRaw(saved.items.map((i) => `${i.key},${i.label},${i.value}`).join("\n")); setMessage("字典已保存"); await load()
  }
  function edit(item: Dict) { setDraft(item); setRaw(item.items.map((i) => `${i.key},${i.label},${i.value}`).join("\n")) }
  useEffect(() => { load().catch((e) => setMessage(e instanceof Error ? e.message : "加载失败")) }, [])

  return <div className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_420px]">
    <section className="rounded-lg border border-line bg-surface"><div className="flex items-center justify-between border-b border-line p-4"><h2 className="text-base font-semibold">字典管理</h2><button className="rounded-lg border border-line px-3 py-2 text-sm" onClick={() => { setDraft(empty); setRaw("") }}>新增字典</button></div>{message && <div className="border-b border-line bg-blue-50 px-4 py-3 text-sm text-primary">{message}</div>}<div className="grid gap-2 p-4">{items.map((item) => <button key={item.id} className="rounded-lg border border-line px-3 py-2 text-left hover:border-primary" onClick={() => edit(item)}><span className="font-medium">{item.name}</span><span className="ml-2 text-xs text-muted">{item.code}</span><span className="block text-xs text-muted">{item.category} · {item.items.length} 项</span></button>)}</div></section>
    <aside className="rounded-lg border border-line bg-surface p-4"><h2 className="text-base font-semibold">{draft.id ? "编辑字典" : "新增字典"}</h2><div className="mt-4 grid gap-3 text-sm"><Text label="编码" value={draft.code} onChange={(v) => setDraft({ ...draft, code: v })} /><Text label="名称" value={draft.name} onChange={(v) => setDraft({ ...draft, name: v })} /><Text label="分类" value={draft.category} onChange={(v) => setDraft({ ...draft, category: v })} /><label className="grid gap-1"><span className="text-muted">字典项，每行 key,label,value</span><textarea className="min-h-56 rounded-lg border border-line px-3 py-2 font-mono text-xs" value={raw} onChange={(e) => setRaw(e.target.value)} /></label><button className="rounded-lg bg-primary px-4 py-2 font-medium text-white" onClick={save}>保存字典</button></div></aside>
  </div>
}

function Text({ label, value, onChange }: { label: string; value: string; onChange: (value: string) => void }) {
  return <label className="grid gap-1"><span className="text-muted">{label}</span><input className="rounded-lg border border-line px-3 py-2" value={value} onChange={(e) => onChange(e.target.value)} /></label>
}
