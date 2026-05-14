import { useEffect, useMemo, useState } from "react"
import { SipSoftphonePanel } from "./SipSoftphonePanel"
import { PatientGroupManager } from "./PatientGroupManager"
import { apiBase, requireSession } from "../lib/auth"

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
  const [workspace, setWorkspace] = useState<"patients" | "groups">("patients")
  const selected = useMemo(() => patients.find((patient) => patient.id === selectedId), [patients, selectedId])

  async function api<T>(path: string, init?: RequestInit): Promise<T> {
    requireSession()
    const response = await fetch(`${apiBase}${path}`, {
      ...init,
      credentials: "include",
      headers: {
        "Content-Type": "application/json",
        ...(init?.headers || {}),
      },
    })
    if (!response.ok) {
      throw new Error(await response.text())
    }
    return response.json()
  }

  async function loginAndLoad() {
    try {
      requireSession()
      const list = await fetch(`${apiBase}/api/v1/patients`, {
        credentials: "include",
      })
      if (!list.ok) throw new Error(await list.text())
      setPatients(await list.json())
      setMessage("")
    } catch (error) {
      setMessage(`患者 API 未连接：${error instanceof Error ? error.message : "未知错误"}`)
    }
  }

  async function search() {
    try {
      const list = await api<Patient[]>(`/api/v1/patients?q=${encodeURIComponent(keyword)}`)
      setPatients(list)
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
        setPatients(patients.map((item) => item.id === selectedId ? patient : item))
      } else {
        setPatients([...patients, patient])
      }
      setDraft(patient)
      setSelectedId(patient.id)
      setView("list")
      setMessage("已保存患者信息")
    } catch (error) {
      setMessage(`保存失败：${error instanceof Error ? error.message : "未知错误"}`)
    }
  }

  function selectPatient(patient: Patient) {
    setSelectedId(patient.id)
    setDraft(patient)
    setView("form")
    setMessage("")
  }

  function newPatient() {
    setSelectedId("")
    setDraft(emptyPatient)
    setView("form")
    setMessage("")
  }

  function backToList() {
    setView("list")
    setSelectedId("")
    setDraft(emptyPatient)
  }

  async function openCall(patient: Patient) {
    if (!patient.phone) {
      setMessage("该患者没有联系电话，无法发起外呼")
      return
    }
    try {
      setCallPatient(patient)
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
              {patients.map((patient) => (
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
                    <button
                      className="rounded-lg bg-primary px-3 py-1.5 text-xs font-medium text-white disabled:bg-gray-200 disabled:text-muted"
                      disabled={!patient.phone}
                      onClick={(event) => {
                        event.stopPropagation()
                        openCall(patient)
                      }}
                    >
                      电话随访
                    </button>
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
      </section>
      )}

      {callPatient && (
        <div className="fixed inset-0 z-50 grid place-items-center bg-gray-900/45 p-4">
          <div className="max-h-[92vh] w-full max-w-6xl overflow-y-auto rounded-lg bg-gray-50 shadow-xl">
            <div className="flex items-center justify-between border-b border-line bg-white px-5 py-4">
              <div>
                <h2 className="text-base font-semibold">电话随访</h2>
                <p className="mt-1 text-sm text-muted">{callPatient.name} · {callPatient.patientNo} · {callPatient.phone}</p>
              </div>
              <button className="rounded-lg border border-line px-3 py-2 text-sm hover:border-primary" onClick={() => setCallPatient(null)}>关闭</button>
            </div>
            <div className="p-5">
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
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
