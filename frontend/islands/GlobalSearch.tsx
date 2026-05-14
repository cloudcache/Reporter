import { useEffect, useMemo, useRef, useState } from "react"
import { authedJson } from "../lib/auth"

interface Patient { id: string; patientNo?: string; name: string; phone?: string; diagnosis?: string }
interface DataSource { id: string; name: string; protocol?: string; endpoint?: string }
interface Dictionary { id: string; code: string; name: string; category: string; items?: Array<{ key: string; label: string; value: string }> }
interface Project { id: string; name: string; targetType?: string; status?: string }
interface FormLibrary { templates?: Array<{ id: string; label: string; scenario?: string }>; commonComponents?: Array<{ id: string; label: string; scenario?: string }>; atomicComponents?: Array<{ id: string; label: string; scenario?: string }> }

interface SearchItem {
  id: string
  title: string
  subtitle: string
  href: string
  kind: string
}

export function GlobalSearch() {
  const [query, setQuery] = useState("")
  const [items, setItems] = useState<SearchItem[]>([])
  const [open, setOpen] = useState(false)
  const [loading, setLoading] = useState(false)
  const boxRef = useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    const onPointerDown = (event: PointerEvent) => {
      if (boxRef.current && !boxRef.current.contains(event.target as Node)) setOpen(false)
    }
    document.addEventListener("pointerdown", onPointerDown)
    return () => document.removeEventListener("pointerdown", onPointerDown)
  }, [])

  useEffect(() => {
    const keyword = query.trim()
    if (keyword.length < 2) {
      setItems([])
      setLoading(false)
      return
    }
    let cancelled = false
    setLoading(true)
    const timer = window.setTimeout(async () => {
      try {
        const results = await searchAll(keyword)
        if (!cancelled) {
          setItems(results)
          setOpen(true)
        }
      } catch {
        if (!cancelled) setItems([])
      } finally {
        if (!cancelled) setLoading(false)
      }
    }, 220)
    return () => {
      cancelled = true
      window.clearTimeout(timer)
    }
  }, [query])

  const groups = useMemo(() => {
    const grouped = new Map<string, SearchItem[]>()
    for (const item of items) grouped.set(item.kind, [...(grouped.get(item.kind) || []), item])
    return Array.from(grouped.entries())
  }, [items])

  function submit() {
    const first = items[0]
    if (first) window.location.href = first.href
  }

  return <div ref={boxRef} className="relative">
    <label className="relative block">
      <span className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-sm text-muted">⌕</span>
      <input
        className="h-10 w-full rounded-lg border border-line bg-gray-50 px-9 text-sm text-ink outline-none focus:border-primary focus:bg-white focus:ring-2 focus:ring-blue-100"
        placeholder="搜索患者、表单、数据源"
        value={query}
        onChange={(event) => { setQuery(event.target.value); setOpen(true) }}
        onFocus={() => query.trim().length >= 2 && setOpen(true)}
        onKeyDown={(event) => {
          if (event.key === "Enter") submit()
          if (event.key === "Escape") setOpen(false)
        }}
      />
    </label>
    {open && query.trim().length >= 2 && <div className="absolute left-0 right-0 top-12 z-50 max-h-[420px] overflow-y-auto rounded-lg border border-line bg-white p-2 shadow-xl">
      {loading && <div className="px-3 py-2 text-sm text-muted">正在搜索...</div>}
      {!loading && !items.length && <div className="px-3 py-2 text-sm text-muted">没有匹配结果</div>}
      {!loading && groups.map(([kind, groupItems]) => <div key={kind} className="py-1">
        <div className="px-3 py-1 text-xs font-semibold text-muted">{kind}</div>
        {groupItems.map((item) => <a key={item.id} className="block rounded-md px-3 py-2 hover:bg-blue-50" href={item.href}>
          <div className="truncate text-sm font-medium text-ink">{item.title}</div>
          <div className="mt-0.5 truncate text-xs text-muted">{item.subtitle}</div>
        </a>)}
      </div>)}
    </div>}
  </div>
}

async function searchAll(keyword: string): Promise<SearchItem[]> {
  const [patients, library, sources, dictionaries, projects] = await Promise.all([
    authedJson<Patient[]>(`/api/v1/patients?q=${encodeURIComponent(keyword)}`).catch(() => []),
    authedJson<FormLibrary>("/api/v1/form-library").catch(() => ({})),
    authedJson<DataSource[]>("/api/v1/data-sources").catch(() => []),
    authedJson<Dictionary[]>("/api/v1/dictionaries").catch(() => []),
    authedJson<Project[]>("/api/v1/projects").catch(() => []),
  ])
  const normalized = keyword.toLowerCase()
  return [
    ...patients.slice(0, 6).map((patient) => ({ id: `patient-${patient.id}`, kind: "患者", title: patient.name, subtitle: [patient.patientNo, patient.phone, patient.diagnosis].filter(Boolean).join(" · "), href: `/patients?focus=${encodeURIComponent(patient.id)}` })),
    ...matchForms(library, normalized),
    ...sources.filter((item) => includes(item.name, normalized) || includes(item.endpoint, normalized) || includes(item.protocol, normalized)).slice(0, 6).map((item) => ({ id: `source-${item.id}`, kind: "数据源", title: item.name, subtitle: [item.protocol, item.endpoint].filter(Boolean).join(" · "), href: `/sources?focus=${encodeURIComponent(item.id)}` })),
    ...dictionaries.filter((item) => dictionaryMatches(item, normalized)).slice(0, 6).map((item) => ({ id: `dict-${item.id}`, kind: "字典", title: item.name, subtitle: `${item.code} · ${item.category} · ${item.items?.length || 0} 项`, href: `/system/dictionaries?focus=${encodeURIComponent(item.id)}` })),
    ...projects.filter((item) => includes(item.name, normalized) || includes(item.targetType, normalized) || includes(item.status, normalized)).slice(0, 6).map((item) => ({ id: `project-${item.id}`, kind: "项目", title: item.name, subtitle: [item.targetType, item.status].filter(Boolean).join(" · "), href: `/projects?focus=${encodeURIComponent(item.id)}` })),
  ].slice(0, 24)
}

function matchForms(library: FormLibrary, keyword: string): SearchItem[] {
  const all = [
    ...(library.templates || []).map((item) => ({ ...item, kind: "表单模板", href: "/forms/library" })),
    ...(library.commonComponents || []).map((item) => ({ ...item, kind: "表单组件", href: "/forms/library" })),
    ...(library.atomicComponents || []).map((item) => ({ ...item, kind: "表单字段", href: "/forms/library" })),
  ]
  return all.filter((item) => includes(item.id, keyword) || includes(item.label, keyword) || includes(item.scenario, keyword)).slice(0, 8).map((item) => ({
    id: `form-${item.id}`,
    kind: item.kind,
    title: item.label,
    subtitle: [item.id, item.scenario].filter(Boolean).join(" · "),
    href: `${item.href}?focus=${encodeURIComponent(item.id)}`,
  }))
}

function dictionaryMatches(item: Dictionary, keyword: string) {
  if (includes(item.code, keyword) || includes(item.name, keyword) || includes(item.category, keyword)) return true
  return (item.items || []).some((entry) => includes(entry.key, keyword) || includes(entry.label, keyword) || includes(entry.value, keyword))
}

function includes(value: unknown, keyword: string) {
  return String(value || "").toLowerCase().includes(keyword)
}
