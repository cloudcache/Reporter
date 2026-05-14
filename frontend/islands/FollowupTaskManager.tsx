import { useEffect, useState } from "react"
import { authedJson } from "../lib/auth"
import { SipSoftphonePanel } from "./SipSoftphonePanel"
import { FollowupChannelActions, type IntegrationChannel } from "./FollowupChannelActions"

interface Task {
  id: string
  planId?: string
  patientId: string
  patientName?: string
  patientPhone?: string
  formTemplateId?: string
  assigneeId?: string
  assigneeName?: string
  role?: string
  channel: string
  status: string
  priority: string
  dueAt: string
  result?: Record<string, unknown>
  lastEvent?: string
}

interface SipEndpoint { id: string; name: string; wssUrl: string; domain: string; proxy: string }
interface CallSession { id: string; seatId: string; patientId?: string; direction: string; phoneNumber: string; status: string }
interface Recording { id: string; callId: string; storageUri: string; duration: number; status: string }
interface Patient {
  id: string
  patientNo: string
  medicalRecordNo?: string
  name: string
  gender: string
  birthDate?: string
  age: number
  phone: string
  bloodType?: string
  diagnosis: string
  lastVisitAt: string
}
interface ClinicalVisit {
  id: string
  visitNo: string
  visitType: string
  departmentCode?: string
  departmentName?: string
  attendingDoctor?: string
  visitAt?: string
  dischargeAt?: string
  diagnosisCode?: string
  diagnosisName?: string
}
interface FormComponent {
  id: string
  type: string
  label: string
  required?: boolean
  helpText?: string
  placeholder?: string
  options?: Array<{ label: string; value: string }>
  rows?: string[]
  columns?: string[]
  scale?: number
}
interface FormTemplate { id: string; label: string; hint: string; scenario?: string; components: FormComponent[] }
interface FormLibrary { templates: FormTemplate[] }
interface FollowupPlan { id: string; name: string; scenario: string; formTemplateId: string }

const statusLabel: Record<string, string> = { pending: "待随访", assigned: "已分配", in_progress: "进行中", completed: "已完成", failed: "失败" }
const channelLabels: Record<string, string> = { phone: "电话随访", sms: "短信随访", wechat: "微信随访", qq: "QQ 随访", web: "网页随访" }
const today = new Date().toISOString().slice(0, 10)

export function FollowupTaskManager() {
  const [tasks, setTasks] = useState<Task[]>([])
  const [message, setMessage] = useState("正在加载随访任务...")
  const [filter, setFilter] = useState("")
  const [callTask, setCallTask] = useState<Task | null>(null)
  const [sipEndpoints, setSipEndpoints] = useState<SipEndpoint[]>([])
  const [calls, setCalls] = useState<CallSession[]>([])
  const [recordings, setRecordings] = useState<Recording[]>([])
  const [integrationChannels, setIntegrationChannels] = useState<IntegrationChannel[]>([])
  const [templates, setTemplates] = useState<FormTemplate[]>([])
  const [plans, setPlans] = useState<FollowupPlan[]>([])
  const [activeWorkTab, setActiveWorkTab] = useState<"form" | "records" | "notes">("form")
  const [formValues, setFormValues] = useState<Record<string, unknown>>({})

  async function load(status = filter) {
    try {
      const [taskData, endpointData, channelData, callData, recordingData, library, planData] = await Promise.all([
        authedJson<Task[]>(`/api/v1/followup/tasks${status ? `?status=${status}` : ""}`),
        authedJson<SipEndpoint[]>("/api/v1/call-center/sip-endpoints"),
        authedJson<IntegrationChannel[]>("/api/v1/integration-channels").catch(() => []),
        authedJson<CallSession[]>("/api/v1/call-center/calls"),
        authedJson<Recording[]>("/api/v1/call-center/recordings"),
        authedJson<FormLibrary>("/api/v1/form-library"),
        authedJson<FollowupPlan[]>("/api/v1/followup/plans"),
      ])
      setTasks(taskData)
      setSipEndpoints(endpointData)
      setIntegrationChannels(channelData)
      setCalls(callData)
      setRecordings(recordingData)
      setTemplates(library.templates || [])
      setPlans(planData)
      setMessage("")
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "加载失败")
    }
  }

  async function transition(task: Task, status: string, event: string) {
    await authedJson<Task>(`/api/v1/followup/tasks/${task.id}`, { method: "PUT", body: JSON.stringify({ ...task, status, lastEvent: event }) })
    await load()
  }

  async function openCall(task: Task) {
    setCallTask(task)
    setActiveWorkTab("form")
    let patient: Patient | undefined
    let visit: ClinicalVisit | undefined
    try {
      const [patientData, visits] = await Promise.all([
        authedJson<Patient>(`/api/v1/patients/${task.patientId}`),
        authedJson<ClinicalVisit[]>(`/api/v1/patients/${task.patientId}/visits`),
      ])
      patient = patientData
      visit = visits[0]
    } catch {
      patient = undefined
    }
    setFormValues(autoFillFormValues(task, patient, visit))
    if (task.status === "pending" || task.status === "assigned") {
      await transition(task, "in_progress", "打开电话随访工作台")
    }
  }

  async function saveDraft(task: Task) {
    await authedJson<Task>(`/api/v1/followup/tasks/${task.id}`, { method: "PUT", body: JSON.stringify({ ...task, status: "in_progress", result: formValues, lastEvent: "保存随访表单草稿" }) })
    await load()
    setMessage("随访表单草稿已保存")
  }

  async function submitFollowup(task: Task) {
    await authedJson<Task>(`/api/v1/followup/tasks/${task.id}`, { method: "PUT", body: JSON.stringify({ ...task, status: "completed", result: formValues, lastEvent: "随访完成并提交表单" }) })
    setCallTask(null)
    await load()
  }

  const activeTemplate = callTask ? templates.find((template) => template.id === callTask.formTemplateId) : undefined
  const activePlan = callTask ? plans.find((plan) => plan.id === callTask.planId) : undefined
  const templateLabel = (task: Task) => templates.find((template) => template.id === task.formTemplateId)?.label || task.formTemplateId || "未绑定模板"
  const taskType = (task: Task) => plans.find((plan) => plan.id === task.planId)?.scenario || templates.find((template) => template.id === task.formTemplateId)?.scenario || channelLabel(task.channel)

  useEffect(() => { load("") }, [])

  return (
    <section className="rounded-lg border border-line bg-surface">
      <div className="flex flex-wrap items-center justify-between gap-3 border-b border-line p-4">
        <div><h2 className="text-base font-semibold">随访任务工作台</h2><p className="mt-1 text-sm text-muted">医生制定方案，护士/随访员执行电话、问卷和结果闭环。</p></div>
        <select className="rounded-lg border border-line px-3 py-2 text-sm" value={filter} onChange={(e) => { setFilter(e.target.value); load(e.target.value) }}>
          <option value="">全部状态</option><option value="pending">待随访</option><option value="in_progress">进行中</option><option value="completed">已完成</option><option value="failed">失败</option>
        </select>
      </div>
      {message && <div className="border-b border-line bg-blue-50 px-4 py-3 text-sm text-primary">{message}</div>}
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 text-muted"><tr><th className="px-4 py-3 text-left">患者</th><th className="px-4 py-3 text-left">电话</th><th className="px-4 py-3 text-left">任务类型</th><th className="px-4 py-3 text-left">表单模板</th><th className="px-4 py-3 text-left">执行人</th><th className="px-4 py-3 text-left">状态</th><th className="px-4 py-3 text-left">到期</th><th className="px-4 py-3 text-right">操作</th></tr></thead>
          <tbody>{tasks.map((task) => <tr key={task.id} className="border-t border-line">
            <td className="px-4 py-3 font-medium">{task.patientName || task.patientId}</td><td className="px-4 py-3">{task.patientPhone}</td><td className="px-4 py-3">{taskType(task)}</td><td className="px-4 py-3">{templateLabel(task)}</td><td className="px-4 py-3">{task.assigneeName || task.role}</td><td className="px-4 py-3">{statusLabel[task.status] || task.status}</td><td className="px-4 py-3">{task.dueAt}</td>
            <td className="px-4 py-3 text-right">
              <div className="flex flex-wrap justify-end gap-2">
                <FollowupChannelActions
                  target={{
                    patientId: task.patientId,
                    patientName: task.patientName,
                    patientPhone: task.patientPhone,
                    formTemplateId: task.formTemplateId,
                    title: `${task.patientName || "患者"}${templateLabel(task)}`,
                  }}
                  channels={integrationChannels}
                  sipEndpoints={sipEndpoints}
                  onPhone={() => openCall(task)}
                  onMessage={setMessage}
                />
                <button className="rounded-lg border border-line px-3 py-1.5 text-xs font-medium hover:border-primary" onClick={() => transition(task, "in_progress", "开始执行")}>开始</button>
                <button className="rounded-lg border border-line px-3 py-1.5 text-xs font-medium hover:border-primary" onClick={() => transition(task, "completed", "随访完成")}>完成</button>
                <button className="rounded-lg border border-red-200 px-3 py-1.5 text-xs font-medium text-red-600 hover:border-red-400" onClick={() => transition(task, "failed", "随访失败")}>失败</button>
              </div>
            </td>
          </tr>)}</tbody>
        </table>
      </div>
      {callTask && (
        <div className="fixed inset-0 z-50 grid place-items-center bg-gray-900/45 p-4">
          <div className="max-h-[94vh] w-full max-w-7xl overflow-y-auto rounded-lg bg-gray-50 shadow-xl">
            <div className="flex items-center justify-between border-b border-line bg-white px-5 py-4">
              <div>
                <h2 className="text-base font-semibold">随访通话工作台</h2>
                <p className="mt-1 text-sm text-muted">{callTask.patientName} · {callTask.patientPhone} · {taskType(callTask)} · {templateLabel(callTask)}</p>
              </div>
              <button className="rounded-lg border border-line px-3 py-2 text-sm hover:border-primary" onClick={() => setCallTask(null)}>关闭</button>
            </div>
            <div className="grid gap-4 p-5 xl:grid-cols-[360px_minmax(0,1fr)]">
              <SipSoftphonePanel
                token=""
                endpoints={sipEndpoints}
                calls={calls}
                recordings={recordings}
                initialTarget={callTask.patientPhone}
                initialPatientId={callTask.patientId}
                initialPatientName={callTask.patientName}
                lockedPatient
                hideActivity
                onClose={() => setCallTask(null)}
                onCallsChange={setCalls}
                onRecordingsChange={setRecordings}
                onMessage={setMessage}
              />
              <section className="rounded-lg border border-line bg-white">
                <div className="flex flex-wrap items-center justify-between gap-3 border-b border-line p-4">
                  <div>
                    <h3 className="font-semibold">随访工作区</h3>
                    <p className="mt-1 text-sm text-muted">{activePlan?.name ? `${activePlan.name} · ` : ""}{activeTemplate?.label || callTask.formTemplateId || "未绑定表单模板"}</p>
                  </div>
                  <div className="flex rounded-lg border border-line bg-gray-50 p-1 text-sm">
                    {[
                      ["form", "表单录入"],
                      ["records", "通话录音"],
                      ["notes", "核对资料"],
                    ].map(([id, label]) => (
                      <button key={id} className={`rounded-md px-3 py-1.5 ${activeWorkTab === id ? "bg-white text-ink shadow-sm" : "text-muted hover:text-ink"}`} onClick={() => setActiveWorkTab(id as "form" | "records" | "notes")}>{label}</button>
                    ))}
                  </div>
                </div>

                {activeWorkTab === "form" && (
                  <div className="p-4">
                    {activeTemplate ? (
                      <>
                        <div className="mb-4 rounded-lg border border-blue-100 bg-blue-50 px-3 py-2 text-sm text-primary">患者基础信息、就诊信息、随访日期和随访方式已自动带入，可在通话中核对后直接修改。</div>
                        <FollowupForm components={activeTemplate.components} values={formValues} onChange={setFormValues} />
                      </>
                    ) : (
                      <div className="rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-700">当前任务没有绑定可用表单模板，请先在随访方案中选择表单模板。</div>
                    )}
                    <div className="mt-4 flex justify-end gap-2 border-t border-line pt-4">
                      <button className="rounded-lg border border-line px-4 py-2 text-sm hover:border-primary" onClick={() => saveDraft(callTask)}>保存草稿</button>
                      <button className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white" onClick={() => submitFollowup(callTask)}>完成随访并提交</button>
                    </div>
                  </div>
                )}

                {activeWorkTab === "records" && (
                  <div className="grid gap-4 p-4 lg:grid-cols-2">
                    <div className="grid content-start gap-3">
                      <h4 className="text-sm font-semibold">通话记录</h4>
                      {calls.filter((call) => call.patientId === callTask.patientId).map((call) => (
                        <div key={call.id} className="rounded-lg border border-line p-3 text-sm">
                          <div className="font-medium">{call.phoneNumber} · {call.status}</div>
                          <div className="mt-1 text-muted">坐席 {call.seatId} · {call.direction}</div>
                          {call.recordingId && <div className="mt-1 text-xs text-primary">录音 {call.recordingId}</div>}
                        </div>
                      ))}
                    </div>
                    <div className="grid content-start gap-3">
                      <h4 className="text-sm font-semibold">录音文件</h4>
                      {recordings.map((item) => (
                        <div key={item.id} className="rounded-lg bg-gray-50 p-3 text-sm">
                          <div className="font-medium">录音 {item.id}</div>
                          <div className="mt-1 text-muted">{item.duration}s · {item.status}</div>
                          <div className="mt-1 truncate text-xs text-muted">{item.storageUri}</div>
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                {activeWorkTab === "notes" && (
                  <div className="grid gap-4 p-4 text-sm md:grid-cols-2">
                    <Info label="患者" value={`${callTask.patientName || callTask.patientId} · ${callTask.patientPhone || "-"}`} />
                    <Info label="随访任务" value={`${statusLabel[callTask.status] || callTask.status} · 到期 ${callTask.dueAt || "-"}`} />
                    <Info label="表单模板" value={activeTemplate?.label || callTask.formTemplateId || "-"} />
                    <Info label="最近事件" value={callTask.lastEvent || "-"} />
                  </div>
                )}
              </section>
            </div>
          </div>
        </div>
      )}
    </section>
  )
}

function autoFillFormValues(task: Task, patient?: Patient, visit?: ClinicalVisit) {
  const existing = task.result || {}
  const gender = normalizeGender(patient?.gender || "")
  const values: Record<string, unknown> = {
    patient_id: task.patientId,
    patient_no: patient?.patientNo || task.patientId,
    patient_name: patient?.name || task.patientName || "",
    patient_gender: gender,
    patient_age: patient?.age || "",
    patient_phone: patient?.phone || task.patientPhone || "",
    blood_type: patient?.bloodType || "",
    visit_id: visit?.id || "",
    visit_no: visit?.visitNo || "",
    visit_date: dateOnly(visit?.visitAt || patient?.lastVisitAt || ""),
    discharge_date: dateOnly(visit?.dischargeAt || ""),
    department: visit?.departmentName || visit?.departmentCode || "",
    doctor_name: visit?.attendingDoctor || "",
    diagnosis: visit?.diagnosisName || patient?.diagnosis || "",
    discharge_diagnosis: visit?.diagnosisName || patient?.diagnosis || "",
    follow_date: today,
    follow_method: task.channel || "phone",
  }
  return { ...values, ...existing }
}

function channelLabel(value: string) {
  return channelLabels[value] || value || "随访任务"
}

function normalizeGender(value: string) {
  if (value === "男" || value.toLowerCase() === "m" || value.toLowerCase() === "male") return "male"
  if (value === "女" || value.toLowerCase() === "f" || value.toLowerCase() === "female") return "female"
  return value || ""
}

function dateOnly(value: string) {
  return value ? value.slice(0, 10) : ""
}

function FollowupForm({ components, values, onChange }: { components: FormComponent[]; values: Record<string, unknown>; onChange: (values: Record<string, unknown>) => void }) {
  function setValue(id: string, value: unknown) {
    onChange({ ...values, [id]: value })
  }
  return (
    <div className="grid gap-4 md:grid-cols-2">
      {components.map((component) => <FormField key={component.id} component={component} value={values[component.id]} onChange={(value) => setValue(component.id, value)} />)}
    </div>
  )
}

function FormField({ component, value, onChange }: { component: FormComponent; value: unknown; onChange: (value: unknown) => void }) {
  if (component.type === "section") {
    return <div className="md:col-span-2"><h4 className="rounded-lg bg-gray-50 px-3 py-2 text-sm font-semibold text-ink">{component.label}</h4></div>
  }
  const label = <span className="text-sm font-medium text-muted">{component.label}{component.required ? <span className="text-red-500"> *</span> : null}</span>
  if (component.type === "textarea") {
    return <label className="grid gap-1 md:col-span-2">{label}<textarea className="min-h-24 rounded-lg border border-line px-3 py-2 text-base leading-6 outline-none focus:border-primary focus:ring-2 focus:ring-blue-100" placeholder={component.placeholder} value={String(value || "")} onChange={(e) => onChange(e.target.value)} />{component.helpText && <span className="text-xs text-muted">{component.helpText}</span>}</label>
  }
  if (component.type === "single_select" || (component.type === "remote_options" && component.options?.length)) {
    return <label className="grid gap-1">{label}<select className="rounded-lg border border-line px-3 py-2 text-base outline-none focus:border-primary focus:ring-2 focus:ring-blue-100" value={String(value || "")} onChange={(e) => onChange(e.target.value)}><option value="">请选择</option>{(component.options || []).map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}</select></label>
  }
  if (component.type === "remote_options") {
    return <label className="grid gap-1">{label}<input className="rounded-lg border border-line bg-blue-50/40 px-3 py-2 text-base outline-none focus:border-primary focus:bg-white focus:ring-2 focus:ring-blue-100" placeholder="系统自动带入，可修改" value={String(value || "")} onChange={(e) => onChange(e.target.value)} />{component.helpText && <span className="text-xs text-muted">{component.helpText}</span>}</label>
  }
  if (component.type === "multi_select") {
    const selected = Array.isArray(value) ? value.map(String) : []
    return <fieldset className="grid gap-2"><legend className="text-sm font-medium text-muted">{component.label}</legend><div className="grid gap-2 rounded-lg border border-line p-3">{(component.options || []).map((option) => <label key={option.value} className="flex items-center gap-2 text-sm"><input type="checkbox" checked={selected.includes(option.value)} onChange={(e) => onChange(e.target.checked ? [...selected, option.value] : selected.filter((item) => item !== option.value))} />{option.label}</label>)}</div></fieldset>
  }
  if (component.type === "rating" || component.type === "likert") {
    const options = component.options || Array.from({ length: component.scale || 5 }, (_, index) => ({ label: String(index + 1), value: String(index + 1) }))
    return <fieldset className="grid gap-2"><legend className="text-sm font-medium text-muted">{component.label}</legend><div className="flex flex-wrap gap-2">{options.map((option) => <button type="button" key={option.value} className={`rounded-lg border px-3 py-2 text-sm ${String(value || "") === option.value ? "border-primary bg-blue-50 text-primary" : "border-line"}`} onClick={() => onChange(option.value)}>{option.label}</button>)}</div></fieldset>
  }
  if (component.type === "matrix") {
    return <div className="overflow-x-auto md:col-span-2"><div className="mb-1 text-sm font-medium text-muted">{component.label}</div><table className="w-full rounded-lg border border-line text-sm"><tbody>{(component.rows || []).map((row) => <tr key={row} className="border-t border-line"><td className="px-3 py-2 font-medium">{row}</td>{(component.columns || []).map((column) => <td key={column} className="px-3 py-2"><label className="flex items-center gap-1"><input type="radio" name={`${component.id}-${row}`} onChange={() => onChange({ ...(typeof value === "object" && value ? value as Record<string, unknown> : {}), [row]: column })} />{column}</label></td>)}</tr>)}</tbody></table></div>
  }
  const inputType = component.type === "number" ? "number" : component.type === "date" ? "date" : "text"
  return <label className="grid gap-1">{label}<input type={inputType} className="rounded-lg border border-line px-3 py-2 text-base outline-none focus:border-primary focus:ring-2 focus:ring-blue-100" placeholder={component.placeholder} value={String(value || "")} onChange={(e) => onChange(inputType === "number" ? Number(e.target.value) : e.target.value)} />{component.helpText && <span className="text-xs text-muted">{component.helpText}</span>}</label>
}

function Info({ label, value }: { label: string; value: string }) {
  return <div className="rounded-lg border border-line p-3"><div className="text-xs text-muted">{label}</div><div className="mt-1 font-medium">{value}</div></div>
}
