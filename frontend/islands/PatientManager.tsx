import { useEffect, useMemo, useState } from "react"
import { SipSoftphonePanel } from "./SipSoftphonePanel"
import { PatientGroupManager } from "./PatientGroupManager"
import { FollowupChannelActions, type IntegrationChannel } from "./FollowupChannelActions"
import { authedJson } from "../lib/auth"

interface Patient {
  id: string
  patientNo: string
  medicalRecordNo?: string
  name: string
  gender: string
  birthDate?: string
  age: number
  idCardNo?: string
  phone: string
  address?: string
  insuranceType?: string
  bloodType?: string
  allergies?: string[]
  emergencyContact?: string
  emergencyPhone?: string
  diagnosis: string
  status: "active" | "follow_up" | "inactive"
  lastVisitAt: string
}

interface SipEndpoint {
  id: string
  name: string
  wssUrl: string
  domain: string
  proxy: string
}

interface CallSession {
  id: string
  seatId: string
  patientId?: string
  direction: string
  phoneNumber: string
  status: string
  recordingId?: string
  analysisId?: string
  interviewForm?: string
}

interface Recording {
  id: string
  callId: string
  storageUri: string
  duration: number
  filename?: string
  mimeType?: string
  sizeBytes?: number
  source?: string
  status: string
}

interface FormTemplate { id: string; label: string; scenario?: string }
interface FormLibrary { templates: FormTemplate[] }

function asArray<T>(value: T[] | null | undefined): T[] {
  return Array.isArray(value) ? value : []
}

interface ClinicalVisit { id: string; visitNo: string; visitType?: string; departmentName?: string; attendingDoctor?: string; visitAt?: string; dischargeAt?: string; diagnosisName?: string; status: string }
interface MedicalRecord { id: string; recordType: string; title: string; chiefComplaint?: string; diagnosisName?: string; recordedAt?: string }
interface Diagnosis { id: string; diagnosisCode?: string; diagnosisName: string; diagnosisType: string; diagnosedAt?: string; departmentName?: string; doctorName?: string }
interface PatientHistory { id: string; historyType: string; title: string; content?: string; recordedAt?: string }
interface MedicationOrder { id: string; drugName: string; dosage?: string; dosageUnit?: string; frequency?: string; route?: string; startAt?: string; status: string; compliance?: string }
interface LabReport { id: string; reportName: string; specimen?: string; reportedAt?: string; status: string; results?: Array<{ id: string; itemName: string; resultValue?: string; unit?: string; referenceRange?: string; abnormalFlag?: string }> }
interface ExamReport { id: string; examName: string; examType?: string; bodyPart?: string; reportConclusion?: string; reportedAt?: string }
interface SurgeryRecord { id: string; operationName: string; operationDate?: string; surgeonName?: string; outcome?: string }
interface FollowupRecord { id: string; followupType?: string; channel?: string; status: string; summary?: string; satisfactionScore?: number; followedAt?: string; operatorName?: string }
interface InterviewFact { id: string; factType: string; factLabel: string; factValue?: string; confidence?: number; extractedAt?: string }
interface Patient360 {
  patient: Patient
  visits: ClinicalVisit[]
  medicalRecords: MedicalRecord[]
  diagnoses: Diagnosis[]
  histories: PatientHistory[]
  medications: MedicationOrder[]
  labReports: LabReport[]
  examReports: ExamReport[]
  surgeries: SurgeryRecord[]
  followupRecords: FollowupRecord[]
  interviewFacts: InterviewFact[]
}

const emptyPatient: Patient = {
  id: "",
  patientNo: "",
  medicalRecordNo: "",
  name: "",
  gender: "男",
  birthDate: "",
  age: 0,
  idCardNo: "",
  phone: "",
  address: "",
  insuranceType: "",
  bloodType: "",
  allergies: [],
  emergencyContact: "",
  emergencyPhone: "",
  diagnosis: "",
  status: "active",
  lastVisitAt: "",
}

const statusLabel = {
  active: "在管",
  follow_up: "随访中",
  inactive: "归档",
}

export function PatientManager() {
  const [patients, setPatients] = useState<Patient[]>([])
  const [keyword, setKeyword] = useState("")
  const [draft, setDraft] = useState<Patient>(emptyPatient)
  const [selectedId, setSelectedId] = useState("")
  const [view, setView] = useState<"list" | "form">("list")
  const [message, setMessage] = useState("正在连接患者 API...")
  const [callPatient, setCallPatient] = useState<Patient | null>(null)
  const [sipEndpoints, setSipEndpoints] = useState<SipEndpoint[]>([])
  const [calls, setCalls] = useState<CallSession[]>([])
  const [recordings, setRecordings] = useState<Recording[]>([])
  const [integrationChannels, setIntegrationChannels] = useState<IntegrationChannel[]>([])
  const [formTemplates, setFormTemplates] = useState<FormTemplate[]>([])
  const [workspace, setWorkspace] = useState<"patients" | "groups">("patients")
  const [patient360, setPatient360] = useState<Patient360 | null>(null)
  const [patient360Tab, setPatient360Tab] = useState<"timeline" | "diagnosis" | "medication" | "exam" | "followup">("timeline")
  const [callWorkTab, setCallWorkTab] = useState<"form" | "records" | "notes">("form")
  const [callFormValues, setCallFormValues] = useState<Record<string, string>>({})
  const safePatients = useMemo(() => asArray(patients), [patients])
  const safeFormTemplates = useMemo(() => asArray(formTemplates), [formTemplates])
  const selected = useMemo(() => safePatients.find((patient) => patient.id === selectedId), [safePatients, selectedId])
  const defaultFollowupTemplateId = useMemo(() => {
    const matched = safeFormTemplates.find((template) => /随访|满意|followup|satisfaction/i.test(`${template.label} ${template.scenario || ""}`))
    return matched?.id || safeFormTemplates[0]?.id || "outpatient-satisfaction"
  }, [safeFormTemplates])
  const defaultFollowupTemplate = useMemo(() => safeFormTemplates.find((template) => template.id === defaultFollowupTemplateId), [defaultFollowupTemplateId, safeFormTemplates])

  async function api<T>(path: string, init?: RequestInit): Promise<T> {
    return authedJson<T>(path, init)
  }

  async function loginAndLoad() {
    try {
      const [patientData, channelData, endpointData, library] = await Promise.all([
        authedJson<Patient[]>("/api/v1/patients"),
        authedJson<IntegrationChannel[]>("/api/v1/integration-channels").catch(() => []),
        authedJson<SipEndpoint[]>("/api/v1/call-center/sip-endpoints").catch(() => []),
        authedJson<FormLibrary>("/api/v1/form-library").catch(() => ({ templates: [] })),
      ])
      setPatients(asArray(patientData))
      setIntegrationChannels(asArray(channelData))
      setSipEndpoints(asArray(endpointData))
      setFormTemplates(asArray(library?.templates))
      setMessage("")
    } catch (error) {
      setMessage(`患者 API 未连接：${error instanceof Error ? error.message : "未知错误"}`)
    }
  }

  async function search() {
    try {
      const list = await api<Patient[]>(`/api/v1/patients?q=${encodeURIComponent(keyword)}`)
      setPatients(asArray(list))
      setMessage("")
    } catch (error) {
      setMessage(`搜索失败：${error instanceof Error ? error.message : "未知错误"}`)
    }
  }

  async function save() {
    try {
      const method = selectedId ? "PUT" : "POST"
      const path = selectedId ? `/api/v1/patients/${selectedId}` : "/api/v1/patients"
      const patient = await api<Patient>(path, { method, body: JSON.stringify(draft) })
      if (selectedId) {
        setPatients(safePatients.map((item) => item.id === selectedId ? patient : item))
      } else {
        setPatients([...safePatients, patient])
      }
      setDraft(patient)
      setSelectedId(patient.id)
      setView("list")
      setMessage("已保存患者信息")
    } catch (error) {
      setMessage(`保存失败：${error instanceof Error ? error.message : "未知错误"}`)
    }
  }

  async function loadPatient360(patientId: string) {
    try {
      setPatient360(await api<Patient360>(`/api/v1/patients/${patientId}/360`))
    } catch (error) {
      setPatient360(null)
      setMessage(`Patient 360 加载失败：${error instanceof Error ? error.message : "未知错误"}`)
    }
  }

  function selectPatient(patient: Patient) {
    setSelectedId(patient.id)
    setDraft(patient)
    setView("form")
    setMessage("")
    loadPatient360(patient.id)
  }

  function newPatient() {
    setSelectedId("")
    setDraft(emptyPatient)
    setPatient360(null)
    setView("form")
    setMessage("")
  }

  function backToList() {
    setView("list")
    setSelectedId("")
    setDraft(emptyPatient)
    setPatient360(null)
  }

  async function openCall(patient: Patient) {
    if (!patient.phone) {
      setMessage("该患者没有联系电话，无法发起外呼")
      return
    }
    try {
      setCallPatient(patient)
      setCallWorkTab("form")
      setCallFormValues(autoFillPatientCallForm(patient))
      const [endpointData, callData, recordingData] = await Promise.all([
        api<SipEndpoint[]>("/api/v1/call-center/sip-endpoints"),
        api<CallSession[]>("/api/v1/call-center/calls"),
        api<Recording[]>("/api/v1/call-center/recordings"),
      ])
      setSipEndpoints(endpointData)
      setCalls(callData)
      setRecordings(recordingData)
      setMessage("")
    } catch (error) {
      setMessage(`打开外呼失败：${error instanceof Error ? error.message : "未知错误"}`)
    }
  }

  useEffect(() => {
    loginAndLoad()
  }, [])

  return (
    <div className="grid gap-5">
      <div className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-line bg-surface p-3">
        <div className="flex rounded-lg border border-line bg-gray-50 p-1 text-sm">
          <button className={`rounded-md px-4 py-2 font-medium ${workspace === "patients" ? "bg-white text-ink shadow-sm" : "text-muted hover:text-ink"}`} onClick={() => setWorkspace("patients")}>患者档案</button>
          <button className={`rounded-md px-4 py-2 font-medium ${workspace === "groups" ? "bg-white text-ink shadow-sm" : "text-muted hover:text-ink"}`} onClick={() => setWorkspace("groups")}>患者分组</button>
        </div>
        <div className="text-sm text-muted">{workspace === "patients" ? "查询、建档、编辑患者主索引" : "分组、标签、绑定随访方案统一在患者管理中维护"}</div>
      </div>

      {workspace === "groups" && <PatientGroupManager />}

      {workspace === "patients" && view === "list" && (
      <section className="rounded-lg border border-line bg-surface">
        <div className="flex flex-col gap-3 border-b border-line p-4 md:flex-row md:items-center md:justify-between">
          <div className="flex min-w-0 flex-1 gap-2">
            <input className="h-10 min-w-0 flex-1 rounded-lg border border-line bg-gray-50 px-3 text-sm outline-none focus:border-primary focus:bg-white" placeholder="姓名、编号、手机号、诊断" value={keyword} onChange={(event) => setKeyword(event.target.value)} />
            <button className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white" onClick={search}>搜索</button>
          </div>
          <button className="h-10 rounded-lg border border-line px-4 text-sm font-medium hover:border-primary" onClick={newPatient}>
            新增患者
          </button>
        </div>
        {message && <div className="border-b border-line bg-blue-50 px-4 py-3 text-sm text-primary">{message}</div>}
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead className="bg-gray-50 text-xs uppercase text-muted">
              <tr>
                <th className="px-4 py-3 text-left">患者编号</th>
                <th className="px-4 py-3 text-left">姓名</th>
                <th className="px-4 py-3 text-left">性别</th>
                <th className="px-4 py-3 text-left">年龄</th>
                <th className="px-4 py-3 text-left">联系电话</th>
                <th className="px-4 py-3 text-left">诊断</th>
                <th className="px-4 py-3 text-left">状态</th>
                <th className="px-4 py-3 text-left">最近就诊</th>
                <th className="px-4 py-3 text-right">操作</th>
              </tr>
            </thead>
            <tbody>
              {safePatients.map((patient) => (
                <tr key={patient.id} className={`cursor-pointer border-t border-line hover:bg-gray-50 ${patient.id === selectedId ? "bg-blue-50" : ""}`} onClick={() => selectPatient(patient)}>
                  <td className="px-4 py-3 font-mono text-xs">{patient.patientNo}</td>
                  <td className="px-4 py-3 font-medium text-ink">{patient.name}</td>
                  <td className="px-4 py-3">{patient.gender}</td>
                  <td className="px-4 py-3">{patient.age}</td>
                  <td className="px-4 py-3">
                    <button
                      className="font-medium text-primary hover:underline disabled:text-muted disabled:no-underline"
                      disabled={!patient.phone}
                      onClick={(event) => {
                        event.stopPropagation()
                        openCall(patient)
                      }}
                    >
                      {patient.phone || "-"}
                    </button>
                  </td>
                  <td className="px-4 py-3">{patient.diagnosis}</td>
                  <td className="px-4 py-3"><span className="rounded-full bg-gray-100 px-2 py-1 text-xs">{statusLabel[patient.status]}</span></td>
                  <td className="px-4 py-3">{patient.lastVisitAt}</td>
                  <td className="px-4 py-3 text-right">
                    <FollowupChannelActions
                      target={{
                        patientId: patient.id,
                        patientName: patient.name,
                        patientPhone: patient.phone,
                        formTemplateId: defaultFollowupTemplateId,
                        title: `${patient.name}随访问卷`,
                      }}
                      channels={integrationChannels}
                      sipEndpoints={sipEndpoints}
                      onPhone={() => openCall(patient)}
                      onMessage={setMessage}
                    />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
      )}

      {workspace === "patients" && view === "form" && (
      <section className="rounded-lg border border-line bg-surface">
        <div className="flex items-center justify-between gap-3 border-b border-line p-4">
          <div>
            <h2 className="text-base font-semibold">{selected ? "编辑患者" : "新增患者"}</h2>
            <p className="mt-1 text-sm text-muted">维护患者主索引、电子病历基础信息和随访联系信息。</p>
          </div>
          <div className="flex gap-2">
            <button className="rounded-lg border border-line px-4 py-2 text-sm hover:border-primary" onClick={backToList}>返回列表</button>
            <button className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white" onClick={save}>保存</button>
          </div>
        </div>
        {message && <div className="border-b border-line bg-blue-50 px-4 py-3 text-sm text-primary">{message}</div>}
        <div className="grid max-w-5xl gap-5 p-4 text-sm md:grid-cols-2 xl:grid-cols-3">
          <label className="grid gap-1">
            <span className="text-muted">患者编号</span>
            <input className="rounded-lg border border-line px-3 py-2" value={draft.patientNo} onChange={(event) => setDraft({ ...draft, patientNo: event.target.value })} />
          </label>
          <label className="grid gap-1">
            <span className="text-muted">病案号/病历号</span>
            <input className="rounded-lg border border-line px-3 py-2" value={draft.medicalRecordNo || ""} onChange={(event) => setDraft({ ...draft, medicalRecordNo: event.target.value })} />
          </label>
          <label className="grid gap-1">
            <span className="text-muted">姓名</span>
            <input className="rounded-lg border border-line px-3 py-2" value={draft.name} onChange={(event) => setDraft({ ...draft, name: event.target.value })} />
          </label>
          <div className="grid grid-cols-2 gap-2">
            <label className="grid gap-1">
              <span className="text-muted">性别</span>
              <select className="rounded-lg border border-line px-3 py-2" value={draft.gender} onChange={(event) => setDraft({ ...draft, gender: event.target.value })}>
                <option>男</option>
                <option>女</option>
                <option>其他</option>
              </select>
            </label>
            <label className="grid gap-1">
              <span className="text-muted">年龄</span>
              <input type="number" className="rounded-lg border border-line px-3 py-2" value={draft.age} onChange={(event) => setDraft({ ...draft, age: Number(event.target.value) })} />
            </label>
          </div>
          <div className="grid grid-cols-2 gap-2">
            <label className="grid gap-1">
              <span className="text-muted">出生日期</span>
              <input type="date" className="rounded-lg border border-line px-3 py-2" value={draft.birthDate || ""} onChange={(event) => setDraft({ ...draft, birthDate: event.target.value })} />
            </label>
            <label className="grid gap-1">
              <span className="text-muted">血型</span>
              <input className="rounded-lg border border-line px-3 py-2" value={draft.bloodType || ""} onChange={(event) => setDraft({ ...draft, bloodType: event.target.value })} />
            </label>
          </div>
          <label className="grid gap-1">
            <span className="text-muted">证件号</span>
            <input className="rounded-lg border border-line px-3 py-2" value={draft.idCardNo || ""} onChange={(event) => setDraft({ ...draft, idCardNo: event.target.value })} />
          </label>
          <label className="grid gap-1">
            <span className="text-muted">联系电话</span>
            <input className="rounded-lg border border-line px-3 py-2" value={draft.phone} onChange={(event) => setDraft({ ...draft, phone: event.target.value })} />
          </label>
          <label className="grid gap-1">
            <span className="text-muted">联系地址</span>
            <input className="rounded-lg border border-line px-3 py-2" value={draft.address || ""} onChange={(event) => setDraft({ ...draft, address: event.target.value })} />
          </label>
          <div className="grid grid-cols-2 gap-2">
            <label className="grid gap-1">
              <span className="text-muted">紧急联系人</span>
              <input className="rounded-lg border border-line px-3 py-2" value={draft.emergencyContact || ""} onChange={(event) => setDraft({ ...draft, emergencyContact: event.target.value })} />
            </label>
            <label className="grid gap-1">
              <span className="text-muted">紧急联系电话</span>
              <input className="rounded-lg border border-line px-3 py-2" value={draft.emergencyPhone || ""} onChange={(event) => setDraft({ ...draft, emergencyPhone: event.target.value })} />
            </label>
          </div>
          <label className="grid gap-1">
            <span className="text-muted">过敏史，逗号分隔</span>
            <input className="rounded-lg border border-line px-3 py-2" value={(draft.allergies || []).join(",")} onChange={(event) => setDraft({ ...draft, allergies: event.target.value.split(",").map((item) => item.trim()).filter(Boolean) })} />
          </label>
          <label className="grid gap-1">
            <span className="text-muted">医保类型</span>
            <input className="rounded-lg border border-line px-3 py-2" value={draft.insuranceType || ""} onChange={(event) => setDraft({ ...draft, insuranceType: event.target.value })} />
          </label>
          <label className="grid gap-1">
            <span className="text-muted">诊断</span>
            <input className="rounded-lg border border-line px-3 py-2" value={draft.diagnosis} onChange={(event) => setDraft({ ...draft, diagnosis: event.target.value })} />
          </label>
          <label className="grid gap-1">
            <span className="text-muted">状态</span>
            <select className="rounded-lg border border-line px-3 py-2" value={draft.status} onChange={(event) => setDraft({ ...draft, status: event.target.value as Patient["status"] })}>
              <option value="active">在管</option>
              <option value="follow_up">随访中</option>
              <option value="inactive">归档</option>
            </select>
          </label>
          <label className="grid gap-1">
            <span className="text-muted">最近就诊日期</span>
            <input type="date" className="rounded-lg border border-line px-3 py-2" value={draft.lastVisitAt} onChange={(event) => setDraft({ ...draft, lastVisitAt: event.target.value })} />
          </label>
        </div>
        {selected && <Patient360Panel data={patient360} activeTab={patient360Tab} setActiveTab={setPatient360Tab} />}
      </section>
      )}

      {callPatient && (
        <div className="fixed inset-0 z-50 grid place-items-center bg-gray-900/45 p-4">
          <div className="max-h-[92vh] w-full max-w-7xl overflow-y-auto rounded-lg bg-gray-50 shadow-xl">
            <div className="flex items-center justify-between border-b border-line bg-white px-5 py-4">
              <div>
                <h2 className="text-base font-semibold">随访通话工作台</h2>
                <p className="mt-1 text-sm text-muted">{callPatient.name} · {callPatient.phone} · {callPatient.diagnosis || "未记录诊断"} · {defaultFollowupTemplate?.label || "随访表单"}</p>
              </div>
              <button className="rounded-lg border border-line px-3 py-2 text-sm hover:border-primary" onClick={() => setCallPatient(null)}>关闭</button>
            </div>
            <div className="grid gap-4 p-5 xl:grid-cols-[360px_minmax(0,1fr)]">
              <SipSoftphonePanel
                token=""
                endpoints={sipEndpoints}
                calls={calls}
                recordings={recordings}
                initialTarget={callPatient.phone}
                initialPatientId={callPatient.id}
                initialPatientName={callPatient.name}
                hideActivity
                onClose={() => setCallPatient(null)}
                onCallsChange={setCalls}
                onRecordingsChange={setRecordings}
                onMessage={setMessage}
              />
              <section className="rounded-lg border border-line bg-white">
                <div className="flex flex-wrap items-center justify-between gap-3 border-b border-line p-4">
                  <div>
                    <h3 className="font-semibold">随访工作区</h3>
                    <p className="mt-1 text-sm text-muted">{defaultFollowupTemplate?.label || "患者随访"} · {callPatient.diagnosis || "未记录诊断"}</p>
                  </div>
                  <div className="flex flex-wrap items-center justify-end gap-3">
                    <FollowupChannelActions
                      target={{
                        patientId: callPatient.id,
                        patientName: callPatient.name,
                        patientPhone: callPatient.phone,
                        formTemplateId: defaultFollowupTemplateId,
                        title: `${callPatient.name}随访问卷`,
                      }}
                      channels={integrationChannels}
                      sipEndpoints={[]}
                      onMessage={setMessage}
                    />
                    <div className="flex rounded-lg border border-line bg-gray-50 p-1 text-sm">
                      {[
                        ["form", "表单录入"],
                        ["records", "通话录音"],
                        ["notes", "核对资料"],
                      ].map(([id, label]) => (
                        <button key={id} className={`rounded-md px-3 py-1.5 ${callWorkTab === id ? "bg-white text-ink shadow-sm" : "text-muted hover:text-ink"}`} onClick={() => setCallWorkTab(id as "form" | "records" | "notes")}>{label}</button>
                      ))}
                    </div>
                  </div>
                </div>
                {callWorkTab === "form" && (
                  <PatientCallForm values={callFormValues} onChange={setCallFormValues} />
                )}
                {callWorkTab === "records" && (
                  <div className="grid gap-4 p-4 lg:grid-cols-2">
                    <InfoList title="通话记录" items={calls.filter((call) => call.patientId === callPatient.id).map((call) => ({ title: `${call.phoneNumber} · ${call.status}`, meta: `坐席 ${call.seatId} · ${call.direction}`, desc: call.recordingId ? `录音 ${call.recordingId}` : "" }))} />
                    <InfoList title="录音文件" items={recordings.map((item) => ({ title: `录音 ${item.id}`, meta: `${item.duration}s · ${item.status}`, desc: item.filename || item.storageUri }))} />
                  </div>
                )}
                {callWorkTab === "notes" && (
                  <div className="grid gap-4 p-4 text-sm md:grid-cols-2">
                    <Info label="患者" value={`${callPatient.name} · ${callPatient.patientNo} · ${callPatient.phone}`} />
                    <Info label="诊断" value={callPatient.diagnosis || "-"} />
                    <Info label="最近就诊" value={callPatient.lastVisitAt || "-"} />
                    <Info label="表单模板" value={defaultFollowupTemplate?.label || defaultFollowupTemplateId} />
                  </div>
                )}
              </section>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

function autoFillPatientCallForm(patient: Patient) {
  return {
    patient_name: patient.name || "",
    gender: patient.gender || "",
    age: String(patient.age || ""),
    phone: patient.phone || "",
    followup_date: new Date().toISOString().slice(0, 10),
    followup_method: "电话",
    current_symptom: "",
    medication_compliance: "一般",
    revisit_plan: "",
    followup_summary: "",
  }
}

function PatientCallForm({ values, onChange }: { values: Record<string, string>; onChange: (values: Record<string, string>) => void }) {
  const setValue = (key: string, value: string) => onChange({ ...values, [key]: value })
  return <div className="grid gap-4 p-4">
    <div className="rounded-lg border border-blue-100 bg-blue-50 px-3 py-2 text-sm text-primary">患者基础信息、随访日期和随访方式已自动带入，可在通话中核对后直接修改。</div>
    <div className="rounded-lg bg-gray-50 px-4 py-3 font-medium">患者基础信息</div>
    <div className="grid gap-4 md:grid-cols-2">
      <Input label="患者姓名" required value={values.patient_name || ""} onChange={(value) => setValue("patient_name", value)} />
      <Select label="性别" value={values.gender || ""} options={["男", "女", "其他"]} onChange={(value) => setValue("gender", value)} />
      <Input label="年龄" value={values.age || ""} onChange={(value) => setValue("age", value)} />
      <Input label="联系电话" value={values.phone || ""} onChange={(value) => setValue("phone", value)} />
    </div>
    <div className="rounded-lg bg-gray-50 px-4 py-3 font-medium">随访记录</div>
    <div className="grid gap-4 md:grid-cols-2">
      <Input label="随访日期" required type="date" value={values.followup_date || ""} onChange={(value) => setValue("followup_date", value)} />
      <Select label="随访方式" value={values.followup_method || "电话"} options={["电话", "短信", "微信", "QQ", "Web"]} onChange={(value) => setValue("followup_method", value)} />
      <Input label="当前症状" value={values.current_symptom || ""} onChange={(value) => setValue("current_symptom", value)} />
      <Select label="用药依从性" value={values.medication_compliance || "一般"} options={["很不满意", "不满意", "一般", "满意", "非常满意"]} onChange={(value) => setValue("medication_compliance", value)} />
      <Input label="复诊计划" value={values.revisit_plan || ""} onChange={(value) => setValue("revisit_plan", value)} />
      <label className="grid gap-1 md:col-span-2">
        <span className="font-medium text-muted">随访小结</span>
        <textarea className="min-h-28 rounded-lg border border-line px-3 py-2 outline-none focus:border-primary" value={values.followup_summary || ""} onChange={(event) => setValue("followup_summary", event.target.value)} />
      </label>
    </div>
  </div>
}

function Input({ label, value, onChange, type = "text", required = false }: { label: string; value: string; onChange: (value: string) => void; type?: string; required?: boolean }) {
  return <label className="grid gap-1">
    <span className="font-medium text-muted">{label} {required && <span className="text-red-500">*</span>}</span>
    <input type={type} className="rounded-lg border border-line px-3 py-2 outline-none focus:border-primary" value={value} onChange={(event) => onChange(event.target.value)} />
  </label>
}

function Select({ label, value, options, onChange }: { label: string; value: string; options: string[]; onChange: (value: string) => void }) {
  return <label className="grid gap-1">
    <span className="font-medium text-muted">{label}</span>
    <select className="rounded-lg border border-line px-3 py-2 outline-none focus:border-primary" value={value} onChange={(event) => onChange(event.target.value)}>
      {options.map((option) => <option key={option} value={option}>{option}</option>)}
    </select>
  </label>
}

function Info({ label, value }: { label: string; value: string }) {
  return <div className="rounded-lg border border-line bg-gray-50 p-3">
    <div className="text-xs text-muted">{label}</div>
    <div className="mt-1 font-medium text-ink">{value}</div>
  </div>
}

function Patient360Panel({ data, activeTab, setActiveTab }: { data: Patient360 | null; activeTab: "timeline" | "diagnosis" | "medication" | "exam" | "followup"; setActiveTab: (tab: "timeline" | "diagnosis" | "medication" | "exam" | "followup") => void }) {
  const safe = useMemo(() => data ? {
    ...data,
    visits: asArray(data.visits),
    medicalRecords: asArray(data.medicalRecords),
    diagnoses: asArray(data.diagnoses),
    histories: asArray(data.histories),
    medications: asArray(data.medications),
    labReports: asArray(data.labReports),
    examReports: asArray(data.examReports),
    surgeries: asArray(data.surgeries),
    followupRecords: asArray(data.followupRecords),
    interviewFacts: asArray(data.interviewFacts),
  } : null, [data])
  const timeline = useMemo(() => {
    if (!safe) return []
    return [
      ...safe.visits.map((item) => ({ at: item.visitAt || "", type: "就诊", title: `${item.departmentName || "未记录科室"} · ${item.visitNo || "-"}`, desc: [item.visitType, item.attendingDoctor, item.diagnosisName].filter(Boolean).join(" · ") })),
      ...safe.medicalRecords.map((item) => ({ at: item.recordedAt || "", type: "病历", title: item.title || item.recordType || "病历记录", desc: [item.recordType, item.chiefComplaint, item.diagnosisName].filter(Boolean).join(" · ") })),
      ...safe.labReports.map((item) => ({ at: item.reportedAt || "", type: "检验", title: item.reportName || "检验报告", desc: `${item.specimen || ""} ${item.status || ""}`.trim() })),
      ...safe.examReports.map((item) => ({ at: item.reportedAt || "", type: "检查", title: item.examName || "检查报告", desc: [item.examType, item.bodyPart, item.reportConclusion].filter(Boolean).join(" · ") })),
      ...safe.followupRecords.map((item) => ({ at: item.followedAt || "", type: "随访", title: item.followupType || "随访记录", desc: [item.channel, item.status, item.summary].filter(Boolean).join(" · ") })),
      ...safe.interviewFacts.map((item) => ({ at: item.extractedAt || "", type: "访谈事实", title: item.factLabel || "抽取事实", desc: item.factValue || "" })),
    ].sort((a, b) => (b.at || "").localeCompare(a.at || ""))
  }, [safe])
  const tabs = [
    ["timeline", "全病程"],
    ["diagnosis", "诊断病史"],
    ["medication", "用药"],
    ["exam", "检验检查"],
    ["followup", "随访访谈"],
  ] as const
  return <div className="border-t border-line p-4">
    <div className="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h3 className="text-base font-semibold">Patient 360 全病程资料</h3>
        <p className="mt-1 text-sm text-muted">来自就诊、病历、诊断、用药、检验检查、随访和访谈抽取事实。</p>
      </div>
      <div className="flex rounded-lg border border-line bg-gray-50 p-1 text-sm">
        {tabs.map(([id, label]) => <button key={id} className={`rounded-md px-3 py-1.5 ${activeTab === id ? "bg-white text-primary shadow-sm" : "text-muted hover:text-ink"}`} onClick={() => setActiveTab(id)}>{label}</button>)}
      </div>
    </div>
    {!data && <div className="mt-4 rounded-lg border border-dashed border-line bg-gray-50 p-4 text-sm text-muted">暂无 Patient 360 数据，需先完成 HIS/EMR/LIS/PACS/随访数据同步。</div>}
    {safe && activeTab === "timeline" && <div className="mt-4 grid gap-2">
      {!timeline.length && <div className="rounded-lg border border-dashed border-line bg-gray-50 p-4 text-sm text-muted">已加载患者主索引，暂无可展示的全病程事件。</div>}
      {timeline.map((item, index) => <div key={`${item.type}-${index}`} className="grid gap-2 rounded-lg border border-line p-3 md:grid-cols-[150px_90px_minmax(0,1fr)]">
        <div className="text-sm text-muted">{item.at || "-"}</div>
        <div className="font-medium text-primary">{item.type}</div>
        <div><div className="font-medium text-ink">{item.title}</div><div className="mt-1 text-sm text-muted">{item.desc || "-"}</div></div>
      </div>)}
    </div>}
    {safe && activeTab === "diagnosis" && <div className="mt-4 grid gap-4 xl:grid-cols-2">
      <InfoList title="诊断" items={safe.diagnoses.map((item) => ({ title: item.diagnosisName || "未命名诊断", meta: [item.diagnosisCode, item.diagnosisType, item.departmentName, item.doctorName, item.diagnosedAt].filter(Boolean).join(" · ") }))} />
      <InfoList title="病史" items={safe.histories.map((item) => ({ title: item.title || item.historyType || "病史记录", meta: [item.historyType, item.recordedAt].filter(Boolean).join(" · "), desc: item.content }))} />
    </div>}
    {safe && activeTab === "medication" && <InfoList className="mt-4" title="用药记录" items={safe.medications.map((item) => ({ title: item.drugName || "用药记录", meta: [item.dosage && `${item.dosage}${item.dosageUnit || ""}`, item.frequency, item.route, item.startAt, item.status, item.compliance].filter(Boolean).join(" · ") }))} />}
    {safe && activeTab === "exam" && <div className="mt-4 grid gap-4 xl:grid-cols-2">
      <InfoList title="检验报告" items={safe.labReports.map((item) => ({ title: item.reportName || "检验报告", meta: [item.specimen, item.reportedAt, item.status].filter(Boolean).join(" · "), desc: asArray(item.results).map((result) => `${result.itemName}: ${result.resultValue || "-"}${result.unit || ""} ${result.abnormalFlag || ""}`).join("；") }))} />
      <InfoList title="检查报告" items={safe.examReports.map((item) => ({ title: item.examName || "检查报告", meta: [item.examType, item.bodyPart, item.reportedAt].filter(Boolean).join(" · "), desc: item.reportConclusion }))} />
      <InfoList title="手术记录" items={safe.surgeries.map((item) => ({ title: item.operationName || "手术记录", meta: [item.operationDate, item.surgeonName, item.outcome].filter(Boolean).join(" · ") }))} />
    </div>}
    {safe && activeTab === "followup" && <div className="mt-4 grid gap-4 xl:grid-cols-2">
      <InfoList title="随访记录" items={safe.followupRecords.map((item) => ({ title: item.followupType || "随访记录", meta: [item.channel, item.status, item.followedAt, item.operatorName, item.satisfactionScore ? `满意度 ${item.satisfactionScore}` : ""].filter(Boolean).join(" · "), desc: item.summary }))} />
      <InfoList title="访谈抽取事实" items={safe.interviewFacts.map((item) => ({ title: item.factLabel || "抽取事实", meta: [item.factType, item.extractedAt, item.confidence ? `置信度 ${Math.round(item.confidence * 100)}%` : ""].filter(Boolean).join(" · "), desc: item.factValue }))} />
    </div>}
  </div>
}

function InfoList({ title, items, className = "" }: { title: string; items: Array<{ title: string; meta?: string; desc?: string }>; className?: string }) {
  return <div className={`rounded-lg border border-line ${className}`}>
    <div className="border-b border-line bg-gray-50 px-4 py-3 font-medium">{title}</div>
    <div className="grid gap-2 p-3">
      {!items.length && <div className="rounded-lg border border-dashed border-line p-3 text-sm text-muted">暂无数据</div>}
      {items.map((item, index) => <div key={`${item.title}-${index}`} className="rounded-lg border border-line p-3">
        <div className="font-medium text-ink">{item.title}</div>
        {item.meta && <div className="mt-1 text-xs text-muted">{item.meta}</div>}
        {item.desc && <div className="mt-2 text-sm leading-6 text-muted">{item.desc}</div>}
      </div>)}
    </div>
  </div>
}
