import { useEffect, useState } from "react"
import { authedJson } from "../lib/auth"

interface Plan {
  id: string
  name: string
  scenario: string
  diseaseCode?: string
  departmentId?: string
  formTemplateId: string
  triggerType: string
  triggerOffset: number
  channel: string
  assigneeRole: string
  status: string
}

interface FormTemplate {
  id: string
  label: string
  hint: string
  scenario?: string
}

interface FormLibraryResponse {
  templates: FormTemplate[]
}

const empty: Plan = { id: "", name: "", scenario: "随访", diseaseCode: "", departmentId: "", formTemplateId: "discharge-follow-up", triggerType: "出院后", triggerOffset: 7, channel: "phone", assigneeRole: "agent", status: "active" }

export function FollowupPlanManager() {
  const [plans, setPlans] = useState<Plan[]>([])
  const [templates, setTemplates] = useState<FormTemplate[]>([])
  const [draft, setDraft] = useState<Plan>(empty)
  const [message, setMessage] = useState("正在加载随访方案...")
  const templateName = (id: string) => templates.find((item) => item.id === id)?.label || id

  async function load() {
    try {
      const [nextPlans, library] = await Promise.all([
        authedJson<Plan[]>("/api/v1/followup/plans"),
        authedJson<FormLibraryResponse>("/api/v1/form-library"),
      ])
      setPlans(nextPlans)
      setTemplates(library.templates || [])
      setMessage("")
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "加载失败")
    }
  }

  async function save() {
    const method = draft.id ? "PUT" : "POST"
    const path = draft.id ? `/api/v1/followup/plans/${draft.id}` : "/api/v1/followup/plans"
    const saved = await authedJson<Plan>(path, { method, body: JSON.stringify(draft) })
    setDraft(saved)
    setMessage("方案已保存")
    await load()
  }

  async function generate(id: string) {
    const tasks = await authedJson<unknown[]>(`/api/v1/followup/plans/${id}/generate`, { method: "POST" })
    setMessage(`已生成 ${tasks.length} 个随访任务`)
  }

  useEffect(() => { load() }, [])

  return (
    <div className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_420px]">
      <section className="rounded-lg border border-line bg-surface">
        <div className="flex items-center justify-between border-b border-line p-4">
          <h2 className="text-base font-semibold">随访方案</h2>
          <button className="rounded-lg border border-line px-3 py-2 text-sm" onClick={() => setDraft(empty)}>新增方案</button>
        </div>
        {message && <div className="border-b border-line bg-blue-50 px-4 py-3 text-sm text-primary">{message}</div>}
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead className="bg-gray-50 text-muted"><tr><th className="px-4 py-3 text-left">名称</th><th className="px-4 py-3 text-left">场景</th><th className="px-4 py-3 text-left">模板</th><th className="px-4 py-3 text-left">规则</th><th className="px-4 py-3 text-right">操作</th></tr></thead>
            <tbody>{plans.map((plan) => <tr key={plan.id} className="border-t border-line">
              <td className="px-4 py-3 font-medium">{plan.name}</td><td className="px-4 py-3">{plan.scenario}</td><td className="px-4 py-3">{templateName(plan.formTemplateId)}</td><td className="px-4 py-3">{plan.triggerType} {plan.triggerOffset} 天</td>
              <td className="px-4 py-3 text-right"><button className="mr-2 text-primary" onClick={() => setDraft(plan)}>编辑</button><button className="text-primary" onClick={() => generate(plan.id)}>生成任务</button></td>
            </tr>)}</tbody>
          </table>
        </div>
      </section>
      <aside className="rounded-lg border border-line bg-surface p-4">
        <h2 className="text-base font-semibold">{draft.id ? "编辑方案" : "新增方案"}</h2>
        <div className="mt-4 grid gap-3 text-sm">
          <Text label="方案名称" value={draft.name} onChange={(v) => setDraft({ ...draft, name: v })} />
          <Text label="场景" value={draft.scenario} onChange={(v) => setDraft({ ...draft, scenario: v })} />
          <Text label="疾病编码" value={draft.diseaseCode || ""} onChange={(v) => setDraft({ ...draft, diseaseCode: v })} />
          <label className="grid gap-1">
            <span className="text-muted">表单模板</span>
            <select
              className="rounded-lg border border-line px-3 py-2"
              value={draft.formTemplateId}
              onChange={(e) => {
                const selected = templates.find((item) => item.id === e.target.value)
                setDraft({ ...draft, formTemplateId: e.target.value, scenario: selected?.scenario || draft.scenario })
              }}
            >
              {templates.map((template) => (
                <option key={template.id} value={template.id}>{template.label}{template.scenario ? `（${template.scenario}）` : ""}</option>
              ))}
              {draft.formTemplateId && !templates.some((template) => template.id === draft.formTemplateId) && <option value={draft.formTemplateId}>{draft.formTemplateId}</option>}
            </select>
            {draft.formTemplateId && <span className="text-xs text-muted">模板 ID：{draft.formTemplateId}</span>}
          </label>
          <Text label="触发类型" value={draft.triggerType} onChange={(v) => setDraft({ ...draft, triggerType: v })} />
          <label className="grid gap-1"><span className="text-muted">触发间隔天数</span><input type="number" className="rounded-lg border border-line px-3 py-2" value={draft.triggerOffset} onChange={(e) => setDraft({ ...draft, triggerOffset: Number(e.target.value) })} /></label>
          <Text label="触达渠道" value={draft.channel} onChange={(v) => setDraft({ ...draft, channel: v })} />
          <Text label="执行角色" value={draft.assigneeRole} onChange={(v) => setDraft({ ...draft, assigneeRole: v })} />
          <button className="rounded-lg bg-primary px-4 py-2 font-medium text-white" onClick={save}>保存方案</button>
        </div>
      </aside>
    </div>
  )
}

function Text({ label, value, onChange }: { label: string; value: string; onChange: (value: string) => void }) {
  return <label className="grid gap-1"><span className="text-muted">{label}</span><input className="rounded-lg border border-line px-3 py-2" value={value} onChange={(e) => onChange(e.target.value)} /></label>
}
