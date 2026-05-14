import { useEffect, useRef, useState } from "react"

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

interface Props {
  token: string
  endpoints: SipEndpoint[]
  calls: CallSession[]
  recordings: Recording[]
  initialTarget?: string
  initialPatientId?: string
  initialPatientName?: string
  lockedPatient?: boolean
  hideActivity?: boolean
  onClose?: () => void
  onCallsChange: (calls: CallSession[]) => void
  onRecordingsChange: (recordings: Recording[]) => void
  onMessage: (message: string) => void
}

const apiBase = "http://127.0.0.1:8080"

export function SipSoftphonePanel({ token, endpoints, calls, recordings, initialTarget, initialPatientId, initialPatientName, lockedPatient, hideActivity, onClose, onCallsChange, onRecordingsChange, onMessage }: Props) {
  const [endpointId, setEndpointId] = useState(endpoints[0]?.id || "")
  const [extension, setExtension] = useState("8001")
  const [target, setTarget] = useState(initialTarget || "13800010001")
  const [patientId, setPatientId] = useState(initialPatientId || "P001")
  const [seatId, setSeatId] = useState("SEAT001")
  const [registered, setRegistered] = useState(false)
  const [activeCall, setActiveCall] = useState<CallSession | null>(null)
  const [recording, setRecording] = useState(false)
  const [recordStartedAt, setRecordStartedAt] = useState<number | null>(null)
  const recorderRef = useRef<MediaRecorder | null>(null)
  const streamRef = useRef<MediaStream | null>(null)
  const chunksRef = useRef<BlobPart[]>([])

  const selectedEndpoint = endpoints.find((endpoint) => endpoint.id === endpointId) || endpoints[0]

  useEffect(() => {
    if (endpoints.length > 0 && !endpointId) setEndpointId(endpoints[0].id)
  }, [endpoints, endpointId])

  useEffect(() => {
    if (initialTarget) setTarget(initialTarget)
    if (initialPatientId) setPatientId(initialPatientId)
    setActiveCall(null)
  }, [initialTarget, initialPatientId])

  async function authed<T>(path: string, init?: RequestInit): Promise<T> {
    const response = await fetch(`${apiBase}${path}`, {
      ...init,
      credentials: "include",
      headers: {
        ...(init?.body instanceof FormData ? {} : { "Content-Type": "application/json" }),
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
        ...(init?.headers || {}),
      },
    })
    if (!response.ok) throw new Error(await response.text())
    return response.json()
  }

  function registerSip() {
    if (!selectedEndpoint) {
      onMessage("请先配置 SIP WSS 网关")
      return
    }
    setRegistered(true)
    onMessage(`已准备注册 ${extension}@${selectedEndpoint.domain}。生产环境接入 SIP.js/JsSIP 后由这里发起 REGISTER。`)
  }

  async function dial() {
    try {
      const call = await authed<CallSession>("/api/v1/call-center/calls", {
        method: "POST",
        body: JSON.stringify({
          seatId,
          patientId,
          direction: "outbound",
          phoneNumber: target,
          status: "connected",
          interviewForm: "outpatient-satisfaction",
        }),
      })
      setActiveCall(call)
      onCallsChange([call, ...calls])
      onMessage(`已接通 ${initialPatientName || "患者"} 的通话 ${call.id}，录音将自动开启。`)
      await startRecording(call)
    } catch (error) {
      onMessage(`呼叫失败：${error instanceof Error ? error.message : "未知错误"}`)
    }
  }

  async function startRecording(callToRecord = activeCall) {
    if (!callToRecord) {
      onMessage("请先创建或接通一通电话")
      return
    }
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
      const mimeType = MediaRecorder.isTypeSupported("audio/webm;codecs=opus") ? "audio/webm;codecs=opus" : "audio/webm"
      const recorder = new MediaRecorder(stream, { mimeType })
      chunksRef.current = []
      recorder.ondataavailable = (event) => {
        if (event.data.size > 0) chunksRef.current.push(event.data)
      }
      recorder.onstop = () => {
        stream.getTracks().forEach((track) => track.stop())
      }
      recorder.start(1000)
      recorderRef.current = recorder
      streamRef.current = stream
      setRecording(true)
      setRecordStartedAt(Date.now())
      onMessage(`通话 ${callToRecord.id} 已接通，录音已自动开始。`)
    } catch (error) {
      onMessage(`无法开始录音：${error instanceof Error ? error.message : "未知错误"}`)
    }
  }

  async function stopRecording() {
    const recorder = recorderRef.current
    if (!recorder || !activeCall) return null
    await new Promise<void>((resolve) => {
      recorder.addEventListener("stop", () => resolve(), { once: true })
      recorder.stop()
    })
    streamRef.current?.getTracks().forEach((track) => track.stop())
    setRecording(false)
    const duration = recordStartedAt ? Math.max(1, Math.round((Date.now() - recordStartedAt) / 1000)) : 0
    const blob = new Blob(chunksRef.current, { type: recorder.mimeType || "audio/webm" })
    const form = new FormData()
    form.append("callId", activeCall.id)
    form.append("duration", String(duration))
    form.append("source", "browser_media_recorder")
    form.append("file", blob, `${activeCall.id}.webm`)
    try {
      const uploaded = await authed<Recording>("/api/v1/call-center/recordings/upload", { method: "POST", body: form })
      onRecordingsChange([uploaded, ...recordings])
      setActiveCall({ ...activeCall, recordingId: uploaded.id, status: "recorded" })
      onMessage(`录音已上传并存储：${uploaded.id}`)
      return uploaded
    } catch (error) {
      onMessage(`录音上传失败：${error instanceof Error ? error.message : "未知错误"}`)
      return null
    }
  }

  async function endCall() {
    if (!activeCall) return
    const uploaded = recording ? await stopRecording() : null
    try {
      const updated = await authed<CallSession>(`/api/v1/call-center/calls/${activeCall.id}?status=${uploaded ? "recorded" : "ended"}`, { method: "PUT" })
      setActiveCall({ ...updated, recordingId: uploaded?.id || updated.recordingId })
      onCallsChange(calls.map((call) => call.id === updated.id ? { ...updated, recordingId: uploaded?.id || updated.recordingId } : call))
      onMessage(`通话已结束${uploaded ? "，录音已自动停止并上传" : ""}。`)
    } catch (error) {
      onMessage(`结束通话失败：${error instanceof Error ? error.message : "未知错误"}`)
    }
  }

  return (
    <section className={hideActivity ? "grid gap-4" : "grid gap-5 xl:grid-cols-[420px_minmax(0,1fr)]"}>
      <article className="rounded-lg border border-line bg-surface p-4">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-sm font-semibold">SIP Web Phone</h3>
            {initialPatientId && <p className="mt-1 text-xs text-muted">已从患者列表带入信息，也支持临时修改号码或患者。</p>}
          </div>
          <div className="flex items-center gap-2">
            <span className={`rounded-full px-2 py-1 text-xs ${registered ? "bg-green-50 text-green-700" : "bg-gray-100 text-muted"}`}>
              {registered ? "已注册" : "离线"}
            </span>
            {onClose && <button className="rounded-lg border border-line px-2 py-1 text-xs hover:border-primary" onClick={onClose}>关闭</button>}
          </div>
        </div>

        <div className="mt-4 grid gap-3 text-sm">
          {initialPatientId && (
            <div className="rounded-lg border border-blue-100 bg-blue-50 p-3">
              <div className="font-medium text-ink">{initialPatientName || "患者"}</div>
              <div className="mt-1 text-sm text-primary">{target || "-"}</div>
              <div className="mt-1 text-xs text-muted">患者 ID：{patientId || "-"}</div>
            </div>
          )}
          <div className="rounded-lg bg-gray-50 p-3 text-xs text-muted">
            当前坐席 {seatId} · 分机 {extension} · 网关 {selectedEndpoint?.name || "未配置"}
          </div>
          {!initialPatientId && (
            <label className="grid gap-1">
              <span className="text-muted">患者 ID</span>
              <input className="rounded-lg border border-line px-3 py-2" value={patientId} onChange={(event) => setPatientId(event.target.value)} />
            </label>
          )}
          <label className="grid gap-1">
            <span className="text-muted">电话号码</span>
            <input className="rounded-lg border border-line px-3 py-2 text-lg font-semibold tracking-normal" value={target} onChange={(event) => setTarget(event.target.value)} />
          </label>
          <div className="grid grid-cols-2 gap-2">
            <button className="rounded-lg bg-primary px-4 py-2 font-medium text-white" onClick={dial}>{lockedPatient ? "呼叫患者" : "拨号"}</button>
            <button className="rounded-lg border border-line px-4 py-2 font-medium hover:border-primary" onClick={registerSip}>重新连接</button>
          </div>
          <div className="grid grid-cols-2 gap-2">
            <button className="rounded-lg border border-line px-4 py-2 font-medium hover:border-primary disabled:opacity-50" disabled={!activeCall || recording} onClick={() => startRecording()}>补录</button>
            <button className="rounded-lg border border-red-200 px-4 py-2 font-medium text-red-600 hover:bg-red-50 disabled:opacity-50" disabled={!activeCall} onClick={endCall}>结束通话</button>
          </div>
        </div>
      </article>

      {!hideActivity && <article className="rounded-lg border border-line bg-surface">
        <div className="border-b border-line p-4">
          <h3 className="text-sm font-semibold">通话与录音存储</h3>
        </div>
        <div className="grid gap-4 p-4 lg:grid-cols-2">
          <div className="grid gap-3">
            {calls.map((call) => (
              <div key={call.id} className="rounded-lg border border-line p-3 text-sm">
                <div className="font-medium">{call.phoneNumber} · {call.status}</div>
                <div className="mt-1 text-muted">坐席 {call.seatId} · 患者 {call.patientId || "-"}</div>
                {call.recordingId && <div className="mt-1 text-xs text-primary">录音 {call.recordingId}</div>}
              </div>
            ))}
          </div>
          <div className="grid gap-3">
            {recordings.map((item) => (
              <div key={item.id} className="rounded-lg bg-gray-50 p-3 text-sm">
                <div className="font-medium">录音 {item.id}</div>
                <div className="mt-1 text-muted">{item.duration}s · {item.status} · {item.source || "-"}</div>
                <div className="mt-1 text-xs text-muted">{item.filename || item.storageUri}</div>
                {item.sizeBytes ? <div className="mt-1 text-xs text-muted">{Math.round(item.sizeBytes / 1024)} KB · {item.mimeType}</div> : null}
              </div>
            ))}
          </div>
        </div>
      </article>}
    </section>
  )
}
