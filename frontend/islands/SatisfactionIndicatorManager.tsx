import { useEffect, useMemo, useState } from "react"
import { authedJson } from "../lib/auth"

interface Template { id: string; label: string; components: Array<Record<string, unknown>> }
interface Library { templates: Template[] }
interface Project { id: string; name: string; targetType: string; formTemplateId: string }
interface Indicator { id: string; projectId?: string; targetType: string; level: number; parentId?: string; name: string; serviceStage?: string; serviceNode?: string; questionId?: string; weight: number; includeTotalScore: boolean; nationalDimension?: string; includeNational: boolean; enabled: boolean }
interface Binding { id: string; projectId?: string; indicatorId: string; formTemplateId: string; questionId: string; questionLabel?: string; scoreDirection: string; weight: number }
interface Question { id: string; label: string; type: string; templateId: string; templateLabel: string }

const targetLabels: Record<string, string> = { outpatient: "门诊", emergency: "急诊", inpatient: "住院", discharge: "出院", physical: "体检", staff: "员工" }
const nationalDimensions = ["综合体验", "诊疗流程", "医生服务", "护理服务", "环境设施", "费用感知", "国考维度"]
const serviceStages: Record<string, Array<{ stage: string; nodes: string[] }>> = {
  outpatient: [
    { stage: "预约挂号", nodes: ["预约便利", "挂号缴费", "信息指引"] },
    { stage: "候诊就医", nodes: ["候诊时间", "医生沟通", "诊疗解释"] },
    { stage: "检查检验", nodes: ["检查预约", "报告获取", "动线指引"] },
    { stage: "药房离院", nodes: ["取药等待", "用药交代", "复诊提醒"] },
  ],
  emergency: [
    { stage: "预检分诊", nodes: ["分诊效率", "急危重识别", "秩序维护"] },
    { stage: "急诊处置", nodes: ["等待时间", "医生沟通", "护士服务"] },
    { stage: "检查转归", nodes: ["检查效率", "留观安排", "转入住院"] },
  ],
  inpatient: [
    { stage: "入院办理", nodes: ["入院流程", "病区接待", "床位安排"] },
    { stage: "住院治疗", nodes: ["医生查房", "护理服务", "用药治疗"] },
    { stage: "检查手术", nodes: ["术前沟通", "检查安排", "疼痛管理"] },
    { stage: "出院准备", nodes: ["出院宣教", "费用解释", "复诊安排"] },
  ],
  discharge: [
    { stage: "出院办理", nodes: ["结算效率", "出院小结", "带药说明"] },
    { stage: "康复随访", nodes: ["康复指导", "用药提醒", "复诊预约"] },
  ],
  physical: [
    { stage: "体检预约", nodes: ["预约便利", "套餐解释", "到检指引"] },
    { stage: "体检过程", nodes: ["排队等候", "检查服务", "环境隐私"] },
    { stage: "报告解读", nodes: ["报告及时", "异常提醒", "复查建议"] },
  ],
  staff: [{ stage: "员工服务", nodes: ["流程体验", "支持响应", "问题反馈"] }],
}
const indicatorFlow = [
  { title: "分类", text: "门诊、急诊、住院、出院、体检按业务线分层。" },
  { title: "环节 / 节点", text: "把服务流程拆到可定位的服务环节和节点。" },
  { title: "三级指标", text: "一级主题、二级环节、三级评价项支持权重。" },
  { title: "题目绑定", text: "从问卷模板选择题目，按指标体系聚合分析。" },
  { title: "国考映射", text: "标记国考维度、是否纳入总分和对标口径。" },
]

export function SatisfactionIndicatorManager() {
  const [projects, setProjects] = useState<Project[]>([])
  const [templates, setTemplates] = useState<Template[]>([])
  const [indicators, setIndicators] = useState<Indicator[]>([])
  const [bindings, setBindings] = useState<Binding[]>([])
  const [projectId, setProjectId] = useState("")
  const [draft, setDraft] = useState<Indicator>({ id: "", targetType: "outpatient", level: 1, parentId: "", name: "", serviceStage: "", serviceNode: "", questionId: "", weight: 1, includeTotalScore: true, nationalDimension: "", includeNational: false, enabled: true })
  const [bindingDraft, setBindingDraft] = useState<Binding>({ id: "", indicatorId: "", formTemplateId: "", questionId: "", questionLabel: "", scoreDirection: "positive", weight: 1 })
  const [targetFilter, setTargetFilter] = useState("all")
  const [batchText, setBatchText] = useState("")
  const [message, setMessage] = useState("正在加载指标体系...")

  const project = projects.find((item) => item.id === projectId)
  const questions = useMemo(() => templates.flatMap((template) => flattenQuestions(template.components).map((question) => ({ ...question, templateId: template.id, templateLabel: template.label }))), [templates])
  const templateQuestions = questions.filter((question) => !bindingDraft.formTemplateId || question.templateId === bindingDraft.formTemplateId)
  const visibleIndicators = useMemo(() => targetFilter === "all" ? indicators : indicators.filter((item) => item.targetType === targetFilter), [indicators, targetFilter])
  const tree = useMemo(() => buildTree(visibleIndicators), [visibleIndicators])
  const activeStages = serviceStages[draft.targetType] || []

  async function load(nextProjectId = projectId) {
    try {
      const [library, nextProjects] = await Promise.all([authedJson<Library>("/api/v1/form-library"), loadProjects()])
      setTemplates(library.templates || [])
      setProjects(nextProjects)
      const active = nextProjectId
      setProjectId(active)
      await loadProject(active)
      setMessage("")
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "加载失败")
    }
  }

  async function loadProject(id: string) {
    const suffix = id ? `?projectId=${id}` : ""
    const [nextIndicators, nextBindings] = await Promise.all([
      authedJson<Indicator[]>(`/api/v1/satisfaction/indicators${suffix}`),
      authedJson<Binding[]>(`/api/v1/satisfaction/indicator-questions${suffix}`),
    ])
    setIndicators(nextIndicators)
    setBindings(nextBindings)
  }

  async function saveIndicator() {
    const saved = await authedJson<Indicator>(draft.id ? `/api/v1/satisfaction/indicators/${draft.id}` : "/api/v1/satisfaction/indicators", { method: draft.id ? "PUT" : "POST", body: JSON.stringify({ ...draft, projectId }) })
    setIndicators(indicators.some((item) => item.id === saved.id) ? indicators.map((item) => item.id === saved.id ? saved : item) : [...indicators, saved])
    setDraft({ id: "", targetType: project?.targetType || "outpatient", level: 1, parentId: "", name: "", serviceStage: "", serviceNode: "", questionId: "", weight: 1, includeTotalScore: true, nationalDimension: "", includeNational: false, enabled: true })
  }

  async function saveBinding() {
    const question = questions.find((item) => item.id === bindingDraft.questionId && item.templateId === bindingDraft.formTemplateId)
    const saved = await authedJson<Binding>(bindingDraft.id ? `/api/v1/satisfaction/indicator-questions/${bindingDraft.id}` : "/api/v1/satisfaction/indicator-questions", { method: bindingDraft.id ? "PUT" : "POST", body: JSON.stringify({ ...bindingDraft, projectId, questionLabel: bindingDraft.questionLabel || question?.label || bindingDraft.questionId }) })
    setBindings(bindings.some((item) => item.id === saved.id) ? bindings.map((item) => item.id === saved.id ? saved : item) : [...bindings, saved])
    setBindingDraft({ id: "", indicatorId: "", formTemplateId: project?.formTemplateId || "", questionId: "", questionLabel: "", scoreDirection: "positive", weight: 1 })
  }

  async function importIndicators() {
    const rows = parseIndicatorImport(batchText)
    if (!rows.length) return setMessage("没有识别到可导入指标")
    const imported: Indicator[] = []
    const byName = new Map(indicators.map((item) => [`${item.targetType}:${item.name}`, item.id]))
    for (const row of rows) {
      const parentId = row.parentName ? byName.get(`${row.targetType}:${row.parentName}`) || "" : ""
      const saved = await authedJson<Indicator>("/api/v1/satisfaction/indicators", { method: "POST", body: JSON.stringify({ ...row, parentId, projectId }) })
      imported.push(saved)
      byName.set(`${saved.targetType}:${saved.name}`, saved.id)
    }
    setIndicators([...indicators, ...imported])
    setBatchText("")
    setMessage(`已导入 ${imported.length} 个指标`)
  }

  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    load(params.get("projectId") || "")
    const onScopeChange = (event: Event) => {
      const detail = (event as CustomEvent<{ projectId?: string }>).detail
      const nextProjectId = detail?.projectId || ""
      setProjectId(nextProjectId)
      loadProject(nextProjectId)
    }
    window.addEventListener("project-scope-change", onScopeChange)
    return () => window.removeEventListener("project-scope-change", onScopeChange)
  }, [])

  return <div className="grid gap-5">
    <section className="rounded-lg border border-line bg-surface p-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="text-base font-semibold">满意度指标体系闭环</h2>
          <p className="mt-1 text-sm text-muted">指标体系作为数据中心资产维护，项目引用后按答卷、科室、医生、病种和就诊类型聚合。</p>
        </div>
        <a className="rounded-lg border border-line px-3 py-2 text-sm text-primary" href="/forms/design">去绑定问卷题目</a>
      </div>
      <div className="mt-4 grid gap-3 lg:grid-cols-5">
        {indicatorFlow.map((item, index) => <FlowStep key={item.title} index={index + 1} title={item.title} text={item.text} />)}
      </div>
    </section>
    <div className="grid gap-5 xl:grid-cols-[360px_minmax(0,1fr)]">
    <aside className="rounded-lg border border-line bg-surface p-4">
      <div className="flex items-center justify-between gap-3">
        <div><h2 className="text-base font-semibold">指标体系</h2><p className="mt-1 text-sm text-muted">独立维护指标树、权重和国考映射</p></div>
        <button className="rounded-lg border border-line px-3 py-2 text-sm" onClick={() => setDraft({ ...draft, id: "", name: "" })}>新增</button>
      </div>
      {message && <div className="mt-3 rounded-lg bg-blue-50 px-3 py-2 text-sm text-primary">{message}</div>}
      <label className="mt-4 grid gap-1 text-sm"><span className="font-medium text-muted">项目范围</span><select className="h-10 rounded-lg border border-line px-3" value={projectId} onChange={(event) => { setProjectId(event.target.value); loadProject(event.target.value) }}><option value="">全部项目</option>{projects.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}</select></label>
      <div className="mt-3 flex flex-wrap gap-2">
        {[{ id: "all", label: "全部" }, ...Object.entries(targetLabels).map(([id, label]) => ({ id, label }))].map((item) => (
          <button key={item.id} className={`rounded-lg px-3 py-1.5 text-xs ${targetFilter === item.id ? "bg-blue-50 text-primary" : "bg-gray-50 text-muted hover:text-ink"}`} onClick={() => setTargetFilter(item.id)}>{item.label}</button>
        ))}
      </div>
      <div className="mt-4 grid gap-2">
        {tree.map((item) => <IndicatorNode key={item.id} item={item} bindings={bindings} onEdit={setDraft} />)}
        {!tree.length && <div className="rounded-lg border border-dashed border-line bg-gray-50 px-3 py-6 text-center text-sm text-muted">当前分类还没有指标，可从右侧新增或批量导入。</div>}
      </div>
    </aside>

    <main className="grid gap-5">
      <section className="grid gap-4 rounded-lg border border-line bg-surface p-4">
        <div>
          <h2 className="text-base font-semibold">{draft.id ? "编辑指标" : "新增指标"}</h2>
          <p className="mt-1 text-sm text-muted">按“分类 → 服务环节 → 服务节点 → 三级指标”维护，权重用于总分和分析短板定位。</p>
        </div>
        <div className="grid gap-3 md:grid-cols-3">
          <Field label="指标名称" value={draft.name} onChange={(v) => setDraft({ ...draft, name: v })} />
          <Select label="就诊分类" value={draft.targetType} options={Object.keys(targetLabels)} labels={targetLabels} onChange={(v) => setDraft({ ...draft, targetType: v })} />
          <Select label="层级" value={String(draft.level)} options={["1", "2", "3"]} labels={{ "1": "一级指标", "2": "二级指标", "3": "三级指标" }} onChange={(v) => setDraft({ ...draft, level: Number(v) })} />
          <label className="grid gap-1 text-sm"><span className="font-medium text-muted">父级指标</span><select className="h-10 rounded-lg border border-line px-3" value={draft.parentId || ""} onChange={(event) => setDraft({ ...draft, parentId: event.target.value })}><option value="">无</option>{indicators.filter((item) => item.id !== draft.id && item.level < draft.level).map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}</select></label>
          <label className="grid gap-1 text-sm"><span className="font-medium text-muted">服务环节</span><select className="h-10 rounded-lg border border-line px-3" value={draft.serviceStage || ""} onChange={(event) => setDraft({ ...draft, serviceStage: event.target.value, serviceNode: "" })}><option value="">请选择</option>{activeStages.map((item) => <option key={item.stage} value={item.stage}>{item.stage}</option>)}</select></label>
          <label className="grid gap-1 text-sm"><span className="font-medium text-muted">服务节点</span><select className="h-10 rounded-lg border border-line px-3" value={draft.serviceNode || ""} onChange={(event) => setDraft({ ...draft, serviceNode: event.target.value })}><option value="">请选择</option>{activeStages.find((item) => item.stage === draft.serviceStage)?.nodes.map((node) => <option key={node} value={node}>{node}</option>)}</select></label>
          <Field label="权重" type="number" value={String(draft.weight)} onChange={(v) => setDraft({ ...draft, weight: Number(v) })} />
          <label className="grid gap-1 text-sm"><span className="font-medium text-muted">国考维度</span><select className="h-10 rounded-lg border border-line px-3" value={draft.nationalDimension || ""} onChange={(event) => setDraft({ ...draft, nationalDimension: event.target.value })}><option value="">不映射</option>{nationalDimensions.map((item) => <option key={item} value={item}>{item}</option>)}</select></label>
          <div className="grid grid-cols-2 gap-2 pt-6"><Toggle label="纳入总分" checked={draft.includeTotalScore} onChange={(v) => setDraft({ ...draft, includeTotalScore: v })} /><Toggle label="纳入国考" checked={draft.includeNational} onChange={(v) => setDraft({ ...draft, includeNational: v })} /></div>
        </div>
        <button className="w-fit rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white disabled:bg-gray-300" disabled={!draft.name.trim()} onClick={saveIndicator}>保存指标</button>
      </section>

      <section className="grid gap-4 rounded-lg border border-line bg-surface p-4">
        <div><h2 className="text-base font-semibold">批量维护指标</h2><p className="mt-1 text-sm text-muted">适合从 Excel 粘贴。每行：分类,层级,父级名称,指标名称,服务环节,服务节点,权重,国考维度,纳入国考。</p></div>
        <textarea className="min-h-28 rounded-lg border border-line px-3 py-2 text-sm" placeholder="outpatient,1,,综合体验,全流程,总体评价,1,综合体验,true&#10;outpatient,2,综合体验,候诊体验,候诊就医,候诊时间,1,诊疗流程,true" value={batchText} onChange={(event) => setBatchText(event.target.value)} />
        <button className="w-fit rounded-lg border border-line px-4 py-2 text-sm font-medium text-primary disabled:text-muted" disabled={!batchText.trim()} onClick={importIndicators}>批量导入指标</button>
      </section>

      <section className="grid gap-4 rounded-lg border border-line bg-surface p-4">
        <div><h2 className="text-base font-semibold">问卷题目绑定指标</h2><p className="mt-1 text-sm text-muted">从表单模板中选择题目，不再手工填写题目 ID。</p></div>
        <div className="grid gap-3 md:grid-cols-4">
          <label className="grid gap-1 text-sm"><span className="font-medium text-muted">指标</span><select className="h-10 rounded-lg border border-line px-3" value={bindingDraft.indicatorId} onChange={(event) => setBindingDraft({ ...bindingDraft, indicatorId: event.target.value })}><option value="">请选择</option>{indicators.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}</select></label>
          <label className="grid gap-1 text-sm"><span className="font-medium text-muted">表单模板</span><select className="h-10 rounded-lg border border-line px-3" value={bindingDraft.formTemplateId || project?.formTemplateId || ""} onChange={(event) => setBindingDraft({ ...bindingDraft, formTemplateId: event.target.value, questionId: "" })}>{templates.map((item) => <option key={item.id} value={item.id}>{item.label}</option>)}</select></label>
          <label className="grid gap-1 text-sm"><span className="font-medium text-muted">题目</span><select className="h-10 rounded-lg border border-line px-3" value={bindingDraft.questionId} onChange={(event) => { const question = templateQuestions.find((item) => item.id === event.target.value); setBindingDraft({ ...bindingDraft, questionId: event.target.value, questionLabel: question?.label || "" }) }}><option value="">请选择</option>{templateQuestions.map((item) => <option key={`${item.templateId}-${item.id}`} value={item.id}>{item.label} · {item.id}</option>)}</select></label>
          <div className="grid grid-cols-2 gap-2"><Select label="计分方向" value={bindingDraft.scoreDirection} options={["positive", "negative"]} labels={{ positive: "正向", negative: "反向" }} onChange={(v) => setBindingDraft({ ...bindingDraft, scoreDirection: v })} /><Field label="权重" type="number" value={String(bindingDraft.weight)} onChange={(v) => setBindingDraft({ ...bindingDraft, weight: Number(v) })} /></div>
        </div>
        <button className="w-fit rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white disabled:bg-gray-300" disabled={!bindingDraft.indicatorId || !bindingDraft.questionId} onClick={saveBinding}>保存绑定</button>
        <div className="grid gap-2">
          {bindings.map((item) => <button key={item.id} className="rounded-lg border border-line px-3 py-2 text-left text-sm hover:border-primary" onClick={() => setBindingDraft(item)}><span className="font-medium">{item.questionLabel || item.questionId}</span><span className="ml-2 text-muted">{indicators.find((indicator) => indicator.id === item.indicatorId)?.name || item.indicatorId} · 权重 {item.weight}</span></button>)}
        </div>
      </section>
    </main>
  </div>
  </div>
}

function loadProjects() {
  return authedJson<Project[]>("/api/v1/projects").catch(() => authedJson<Project[]>("/api/v1/satisfaction/projects"))
}

function flattenQuestions(components: Array<Record<string, unknown>>): Question[] {
  return components.flatMap((component) => {
    const type = String(component.type || "")
    const current = type && type !== "section" ? [{ id: String(component.id || ""), label: String(component.label || component.id || ""), type, templateId: "", templateLabel: "" }] : []
    const children = Array.isArray(component.children) ? flattenQuestions(component.children as Array<Record<string, unknown>>) : []
    return [...current, ...children].filter((item) => item.id)
  })
}

function buildTree(items: Indicator[]) {
  const byParent = new Map<string, Indicator[]>()
  items.forEach((item) => byParent.set(item.parentId || "", [...(byParent.get(item.parentId || "") || []), item]))
  const attach = (item: Indicator): Indicator & { children: Array<Indicator & { children: any[] }> } => ({ ...item, children: (byParent.get(item.id) || []).map(attach) })
  return (byParent.get("") || items.filter((item) => item.level === 1)).map(attach)
}

function parseIndicatorImport(text: string): Array<Indicator & { parentName?: string }> {
  return text.split(/\n+/).map((line) => line.trim()).filter(Boolean).map((line) => {
    const [targetType = "outpatient", level = "1", parentName = "", name = "", serviceStage = "", serviceNode = "", weight = "1", nationalDimension = "", includeNational = "false"] = line.split(/,|，/).map((item) => item.trim())
    return {
      id: "",
      targetType,
      level: Number(level || 1),
      parentId: "",
      parentName,
      name,
      serviceStage,
      serviceNode,
      questionId: "",
      weight: Number(weight || 1),
      includeTotalScore: true,
      nationalDimension,
      includeNational: ["true", "是", "1", "yes"].includes(includeNational.toLowerCase()),
      enabled: true,
    }
  }).filter((item) => item.name)
}

function FlowStep({ index, title, text }: { index: number; title: string; text: string }) {
  return <div className="rounded-lg border border-line bg-white p-3">
    <div className="flex items-center gap-2"><span className="grid h-6 w-6 place-items-center rounded-full bg-blue-50 text-xs font-semibold text-primary">{index}</span><span className="font-medium">{title}</span></div>
    <div className="mt-2 text-xs leading-5 text-muted">{text}</div>
  </div>
}

function IndicatorNode({ item, bindings, onEdit }: { item: Indicator & { children?: any[] }; bindings: Binding[]; onEdit: (item: Indicator) => void }) {
  const bound = bindings.filter((binding) => binding.indicatorId === item.id).length
  return <div className="rounded-lg border border-line bg-white p-3">
    <button className="w-full text-left" onClick={() => onEdit(item)}>
      <div className="font-medium">{item.name}</div>
      <div className="mt-1 text-xs text-muted">L{item.level} · 权重 {item.weight} · {item.nationalDimension || "未映射国考"} · 绑定 {bound} 题</div>
    </button>
    {!!item.children?.length && <div className="mt-2 grid gap-2 border-l border-line pl-3">{item.children.map((child) => <IndicatorNode key={child.id} item={child} bindings={bindings} onEdit={onEdit} />)}</div>}
  </div>
}

function Field({ label, value, type = "text", onChange }: { label: string; value: string; type?: string; onChange: (value: string) => void }) {
  return <label className="grid gap-1 text-sm"><span className="font-medium text-muted">{label}</span><input type={type} className="h-10 rounded-lg border border-line px-3" value={value} onChange={(event) => onChange(event.target.value)} /></label>
}

function Select({ label, value, options, labels, onChange }: { label: string; value: string; options: string[]; labels: Record<string, string>; onChange: (value: string) => void }) {
  return <label className="grid gap-1 text-sm"><span className="font-medium text-muted">{label}</span><select className="h-10 rounded-lg border border-line px-3" value={value} onChange={(event) => onChange(event.target.value)}>{options.map((option) => <option key={option} value={option}>{labels[option] || option}</option>)}</select></label>
}

function Toggle({ label, checked, onChange }: { label: string; checked: boolean; onChange: (value: boolean) => void }) {
  return <label className="flex h-10 items-center gap-2 rounded-lg border border-line px-3 text-sm"><input type="checkbox" checked={checked} onChange={(event) => onChange(event.target.checked)} />{label}</label>
}
