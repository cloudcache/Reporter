import { useEffect, useMemo, useState } from "react"
import { authedJson } from "../lib/auth"

interface Group {
  id: string
  name: string
  category: string
  mode: string
  assignmentMode: string
  followupPlanId?: string
  rules?: Record<string, unknown>
  permissions?: Record<string, unknown>
  memberCount: number
}

interface Tag {
  id: string
  name: string
  color: string
  description?: string
}

interface Plan {
  id: string
  name: string
  scenario: string
}

const emptyGroup: Group = { id: "", name: "", category: "专病", mode: "person", assignmentMode: "manual", followupPlanId: "", rules: {}, permissions: {}, memberCount: 0 }
const emptyTag: Tag = { id: "", name: "", color: "#2563eb", description: "" }
const categories = ["全部", "住院", "门诊", "专病", "专科", "肿瘤", "妇幼", "慢病"]

export function PatientGroupManager() {
  const [groups, setGroups] = useState<Group[]>([])
  const [tags, setTags] = useState<Tag[]>([])
  const [plans, setPlans] = useState<Plan[]>([])
  const [draft, setDraft] = useState<Group>(emptyGroup)
  const [tagDraft, setTagDraft] = useState<Tag>(emptyTag)
  const [membersText, setMembersText] = useState("")
  const [rulesText, setRulesText] = useState("{}")
  const [category, setCategory] = useState("全部")
  const [message, setMessage] = useState("正在加载患者分组...")

  const filteredGroups = useMemo(() => groups.filter((group) => category === "全部" || group.category === category), [groups, category])
  const selectedPlan = plans.find((plan) => plan.id === draft.followupPlanId)
  const totalMembers = groups.reduce((sum, group) => sum + group.memberCount, 0)

  async function load(selectFirst = false) {
    try {
      const [nextGroups, nextTags, nextPlans] = await Promise.all([
        authedJson<Group[]>("/api/v1/patient-groups"),
        authedJson<Tag[]>("/api/v1/patient-tags"),
        authedJson<Plan[]>("/api/v1/followup/plans"),
      ])
      setGroups(nextGroups)
      setTags(nextTags)
      setPlans(nextPlans)
      if (selectFirst && nextGroups[0]) editGroup(nextGroups[0])
      setMessage("")
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "加载失败")
    }
  }

  async function saveGroup() {
    let rules: Record<string, unknown> = {}
    try {
      rules = JSON.parse(rulesText || "{}")
    } catch {
      setMessage("自动规则不是有效 JSON，请检查后再保存")
      return
    }
    const body = { ...draft, rules }
    const saved = await authedJson<Group>(draft.id ? `/api/v1/patient-groups/${draft.id}` : "/api/v1/patient-groups", { method: draft.id ? "PUT" : "POST", body: JSON.stringify(body) })
    const ids = membersText.split(/[\n,，\s]+/).map((item) => item.trim()).filter(Boolean)
    if (ids.length) {
      await authedJson(`/api/v1/patient-groups/${saved.id}/members`, { method: "PUT", body: JSON.stringify({ patientIds: ids }) })
    }
    setDraft(saved)
    setMessage("患者分组已保存")
    await load()
  }

  async function saveTag() {
    const saved = await authedJson<Tag>("/api/v1/patient-tags", { method: "POST", body: JSON.stringify(tagDraft) })
    setTagDraft(saved)
    setMessage("患者标签已保存")
    await load()
  }

  function editGroup(group: Group) {
    setDraft(group)
    setRulesText(JSON.stringify(group.rules || {}, null, 2))
    setMembersText("")
  }

  function createGroup() {
    setDraft(emptyGroup)
    setRulesText("{}")
    setMembersText("")
  }

  useEffect(() => { load(true) }, [])

  return (
    <div className="grid gap-5 xl:grid-cols-[360px_minmax(0,1fr)]">
      <aside className="rounded-lg border border-line bg-surface">
        <div className="border-b border-line p-4">
          <div className="flex items-center justify-between gap-3">
            <div>
              <h2 className="text-base font-semibold">患者分组</h2>
              <p className="mt-1 text-sm text-muted">{groups.length} 个分组 · {totalMembers} 人次</p>
            </div>
            <button className="rounded-lg bg-primary px-3 py-2 text-sm font-medium text-white" onClick={createGroup}>新增</button>
          </div>
          <div className="mt-4 flex gap-2 overflow-x-auto pb-1">
            {categories.map((item) => (
              <button key={item} className={`shrink-0 rounded-lg border px-3 py-1.5 text-sm ${category === item ? "border-primary bg-blue-50 text-primary" : "border-line text-muted hover:text-ink"}`} onClick={() => setCategory(item)}>{item}</button>
            ))}
          </div>
        </div>
        {message && <div className="border-b border-line bg-blue-50 px-4 py-3 text-sm text-primary">{message}</div>}
        <div className="max-h-[640px] overflow-y-auto p-3">
          <div className="grid gap-2">
            {filteredGroups.map((group) => (
              <button key={group.id} className={`rounded-lg border p-3 text-left transition ${draft.id === group.id ? "border-primary bg-blue-50" : "border-line hover:border-primary/60 hover:bg-gray-50"}`} onClick={() => editGroup(group)}>
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <div className="font-medium text-ink">{group.name}</div>
                    <div className="mt-1 text-xs text-muted">{group.category} · {group.mode === "visit" ? "按就诊" : "按患者"} · {group.assignmentMode === "auto" ? "自动分组" : "手工分组"}</div>
                  </div>
                  <span className="rounded-full bg-white px-2 py-1 text-xs font-medium text-primary">{group.memberCount} 人</span>
                </div>
                <div className="mt-2 text-xs text-muted">方案：{plans.find((plan) => plan.id === group.followupPlanId)?.name || "未绑定"}</div>
              </button>
            ))}
            {!filteredGroups.length && <div className="rounded-lg border border-dashed border-line p-6 text-center text-sm text-muted">当前分类下暂无分组</div>}
          </div>
        </div>
      </aside>

      <main className="grid gap-5">
        <section className="rounded-lg border border-line bg-surface">
          <div className="flex flex-wrap items-center justify-between gap-3 border-b border-line p-4">
            <div>
              <h2 className="text-base font-semibold">{draft.id ? "编辑分组" : "新增分组"}</h2>
              <p className="mt-1 text-sm text-muted">按患者、就诊、诊断、科室或病种建立随访人群，直接绑定随访方案。</p>
            </div>
            <button className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white" onClick={saveGroup}>保存分组</button>
          </div>
          <div className="grid gap-5 p-4 lg:grid-cols-[minmax(0,1fr)_320px]">
            <div className="grid gap-4 md:grid-cols-2">
              <Text label="分组名称" value={draft.name} onChange={(v) => setDraft({ ...draft, name: v })} />
              <Select label="分组分类" value={draft.category} options={categories.filter((item) => item !== "全部")} onChange={(v) => setDraft({ ...draft, category: v })} />
              <Segmented label="分组单位" value={draft.mode} options={[["person", "按患者"], ["visit", "按就诊"]]} onChange={(v) => setDraft({ ...draft, mode: v })} />
              <Segmented label="分组方式" value={draft.assignmentMode} options={[["manual", "手工"], ["auto", "自动"]]} onChange={(v) => setDraft({ ...draft, assignmentMode: v })} />
              <label className="grid gap-1 md:col-span-2">
                <span className="text-sm font-medium text-muted">绑定随访方案</span>
                <select className="h-11 rounded-lg border border-line px-3 text-sm outline-none focus:border-primary focus:ring-2 focus:ring-blue-100" value={draft.followupPlanId || ""} onChange={(e) => setDraft({ ...draft, followupPlanId: e.target.value })}>
                  <option value="">不绑定</option>
                  {plans.map((plan) => <option key={plan.id} value={plan.id}>{plan.name}（{plan.scenario}）</option>)}
                </select>
              </label>
              <label className="grid gap-1 md:col-span-2">
                <span className="text-sm font-medium text-muted">手工加入患者</span>
                <textarea className="min-h-24 rounded-lg border border-line px-3 py-2 text-sm outline-none focus:border-primary focus:ring-2 focus:ring-blue-100" value={membersText} onChange={(e) => setMembersText(e.target.value)} placeholder="输入患者 ID，支持换行、逗号或空格分隔，例如 P001 P002" />
              </label>
              <label className="grid gap-1 md:col-span-2">
                <span className="text-sm font-medium text-muted">自动分组规则</span>
                <textarea className="min-h-28 rounded-lg border border-line bg-gray-50 px-3 py-2 font-mono text-xs outline-none focus:border-primary focus:bg-white focus:ring-2 focus:ring-blue-100" value={rulesText} onChange={(e) => setRulesText(e.target.value)} />
              </label>
            </div>
            <div className="grid content-start gap-3 rounded-lg border border-line bg-gray-50 p-4 text-sm">
              <h3 className="font-semibold">分组摘要</h3>
              <Info label="当前人数" value={`${draft.memberCount || 0} 人`} />
              <Info label="随访方案" value={selectedPlan ? `${selectedPlan.name} · ${selectedPlan.scenario}` : "未绑定"} />
              <Info label="访问模式" value={`${draft.mode === "visit" ? "按就诊" : "按患者"} / ${draft.assignmentMode === "auto" ? "自动分组" : "手工分组"}`} />
              <div className="rounded-lg bg-white p-3 text-xs leading-5 text-muted">手工成员会覆盖保存到当前分组；自动分组规则保存在后台，后续任务生成时可按规则筛选患者。</div>
            </div>
          </div>
        </section>

        <section className="rounded-lg border border-line bg-surface p-4">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <h2 className="text-base font-semibold">患者标签</h2>
              <p className="mt-1 text-sm text-muted">死亡、纠纷、重点随访等提醒标签集中维护。</p>
            </div>
            <button className="rounded-lg border border-line px-3 py-2 text-sm hover:border-primary" onClick={() => setTagDraft(emptyTag)}>新增标签</button>
          </div>
          <div className="mt-4 flex flex-wrap gap-2">{tags.map((tag) => <button key={tag.id} className="rounded-lg border border-line px-3 py-2 text-sm hover:border-primary" onClick={() => setTagDraft(tag)}><span className="mr-2 inline-block h-2.5 w-2.5 rounded-full" style={{ backgroundColor: tag.color }} />{tag.name}</button>)}</div>
          <div className="mt-4 grid gap-3 md:grid-cols-[1fr_130px_1.5fr_auto]">
            <input className="h-10 rounded-lg border border-line px-3 text-sm outline-none focus:border-primary focus:ring-2 focus:ring-blue-100" placeholder="标签名称" value={tagDraft.name} onChange={(e) => setTagDraft({ ...tagDraft, name: e.target.value })} />
            <input className="h-10 rounded-lg border border-line px-3 text-sm outline-none focus:border-primary focus:ring-2 focus:ring-blue-100" value={tagDraft.color} onChange={(e) => setTagDraft({ ...tagDraft, color: e.target.value })} />
            <input className="h-10 rounded-lg border border-line px-3 text-sm outline-none focus:border-primary focus:ring-2 focus:ring-blue-100" placeholder="说明" value={tagDraft.description || ""} onChange={(e) => setTagDraft({ ...tagDraft, description: e.target.value })} />
            <button className="h-10 rounded-lg bg-primary px-4 text-sm font-medium text-white" onClick={saveTag}>保存标签</button>
          </div>
        </section>
      </main>
    </div>
  )
}

function Text({ label, value, onChange }: { label: string; value: string; onChange: (value: string) => void }) {
  return <label className="grid gap-1"><span className="text-sm font-medium text-muted">{label}</span><input className="h-11 rounded-lg border border-line px-3 text-sm outline-none focus:border-primary focus:ring-2 focus:ring-blue-100" value={value} onChange={(e) => onChange(e.target.value)} /></label>
}

function Select({ label, value, options, onChange }: { label: string; value: string; options: string[]; onChange: (value: string) => void }) {
  return <label className="grid gap-1"><span className="text-sm font-medium text-muted">{label}</span><select className="h-11 rounded-lg border border-line px-3 text-sm outline-none focus:border-primary focus:ring-2 focus:ring-blue-100" value={value} onChange={(e) => onChange(e.target.value)}>{options.map((item) => <option key={item} value={item}>{item}</option>)}</select></label>
}

function Segmented({ label, value, options, onChange }: { label: string; value: string; options: string[][]; onChange: (value: string) => void }) {
  return (
    <div className="grid gap-1">
      <span className="text-sm font-medium text-muted">{label}</span>
      <div className="grid grid-cols-2 rounded-lg border border-line bg-gray-50 p-1">
        {options.map(([id, name]) => <button key={id} className={`rounded-md px-3 py-2 text-sm font-medium ${value === id ? "bg-white text-primary shadow-sm" : "text-muted hover:text-ink"}`} onClick={() => onChange(id)}>{name}</button>)}
      </div>
    </div>
  )
}

function Info({ label, value }: { label: string; value: string }) {
  return <div className="flex items-start justify-between gap-3 border-b border-line/70 pb-2 last:border-0"><span className="text-muted">{label}</span><span className="text-right font-medium text-ink">{value}</span></div>
}
