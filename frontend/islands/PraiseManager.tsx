import { useEffect, useState } from "react"
import type { ReactNode } from "react"
import { authedFetch, authedJson } from "../lib/auth"

interface PraiseRecord {
  id: string
  projectId?: string
  praiseDate: string
  praiseType?: string
  praiseMethod?: string
  departmentName?: string
  staffName?: string
  patientName?: string
  quantity: number
  rewardAmount: number
  content?: string
  remark?: string
  status: string
}

const emptyPraise: PraiseRecord = {
  id: "",
  praiseDate: new Date().toISOString().slice(0, 10),
  praiseType: "好人好事",
  praiseMethod: "电话表扬",
  departmentName: "",
  staffName: "",
  patientName: "",
  quantity: 1,
  rewardAmount: 0,
  content: "",
  remark: "",
  status: "confirmed",
}

export function PraiseManager() {
  const [items, setItems] = useState<PraiseRecord[]>([])
  const [draft, setDraft] = useState<PraiseRecord>(emptyPraise)
  const [selectedId, setSelectedId] = useState("")
  const [keyword, setKeyword] = useState("")
  const [message, setMessage] = useState("")

  async function load(nextKeyword = keyword) {
    try {
      const params = new URLSearchParams()
      if (nextKeyword) params.set("q", nextKeyword)
      const data = await authedJson<PraiseRecord[]>(`/api/v1/praise-records${params.toString() ? `?${params}` : ""}`)
      setItems(data)
      setMessage("")
    } catch (error) {
      setMessage(`表扬记录加载失败：${error instanceof Error ? error.message : "未知错误"}`)
    }
  }

  async function save() {
    try {
      const path = selectedId ? `/api/v1/praise-records/${selectedId}` : "/api/v1/praise-records"
      const method = selectedId ? "PUT" : "POST"
      const saved = await authedJson<PraiseRecord>(path, { method, body: JSON.stringify(draft) })
      setDraft(saved)
      setSelectedId(saved.id)
      await load()
      setMessage("表扬登记已保存")
    } catch (error) {
      setMessage(`保存失败：${error instanceof Error ? error.message : "未知错误"}`)
    }
  }

  async function remove(id: string) {
    try {
      const response = await authedFetch(`/api/v1/praise-records/${id}`, { method: "DELETE" })
      if (!response.ok) throw new Error(await response.text())
      if (selectedId === id) {
        setSelectedId("")
        setDraft(emptyPraise)
      }
      await load()
      setMessage("已删除表扬登记")
    } catch (error) {
      setMessage(`删除失败：${error instanceof Error ? error.message : "未知错误"}`)
    }
  }

  useEffect(() => { load() }, [])

  return (
    <div className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_420px]">
      <section className="min-w-0 rounded-lg border border-line bg-surface">
        <div className="flex flex-wrap items-center justify-between gap-3 border-b border-line p-4">
          <div>
            <h2 className="text-lg font-semibold">表扬登记列表</h2>
            <p className="mt-1 text-sm text-muted">登记好人好事、表扬方式、人员科室、奖励金额，并进入评价投诉分析。</p>
          </div>
          <div className="flex gap-2">
            <input className="w-64 rounded-lg border border-line px-3 py-2 text-sm" placeholder="搜索科室、人员、患者、内容" value={keyword} onChange={(event) => setKeyword(event.target.value)} />
            <button className="rounded-lg border border-line px-3 py-2 text-sm" onClick={() => load(keyword)}>查询</button>
            <button className="rounded-lg bg-primary px-3 py-2 text-sm font-medium text-white" onClick={() => { setSelectedId(""); setDraft(emptyPraise) }}>新增</button>
          </div>
        </div>
        {message && <div className="m-4 rounded-lg bg-blue-50 px-3 py-2 text-sm text-primary">{message}</div>}
        <div className="overflow-x-auto">
          <table className="w-full min-w-[920px] text-left text-sm">
            <thead className="bg-gray-50 text-muted">
              <tr>
                <th className="px-4 py-3">日期</th>
                <th className="px-4 py-3">科室</th>
                <th className="px-4 py-3">人员</th>
                <th className="px-4 py-3">患者</th>
                <th className="px-4 py-3">方式</th>
                <th className="px-4 py-3">数量</th>
                <th className="px-4 py-3">奖励金额</th>
                <th className="px-4 py-3">状态</th>
                <th className="px-4 py-3">操作</th>
              </tr>
            </thead>
            <tbody>
              {items.map((item) => (
                <tr key={item.id} className="border-t border-line">
                  <td className="px-4 py-3">{item.praiseDate}</td>
                  <td className="px-4 py-3">{item.departmentName || "-"}</td>
                  <td className="px-4 py-3">{item.staffName || "-"}</td>
                  <td className="px-4 py-3">{item.patientName || "-"}</td>
                  <td className="px-4 py-3">{item.praiseMethod || "-"}</td>
                  <td className="px-4 py-3">{item.quantity}</td>
                  <td className="px-4 py-3">{Number(item.rewardAmount || 0).toFixed(2)}</td>
                  <td className="px-4 py-3">{statusLabel(item.status)}</td>
                  <td className="px-4 py-3">
                    <div className="flex gap-2">
                      <button className="rounded-md border border-line px-2 py-1 text-xs" onClick={() => { setSelectedId(item.id); setDraft(item) }}>编辑</button>
                      <button className="rounded-md border border-red-100 px-2 py-1 text-xs text-red-600" onClick={() => remove(item.id)}>删除</button>
                    </div>
                  </td>
                </tr>
              ))}
              {items.length === 0 && <tr><td className="px-4 py-8 text-center text-muted" colSpan={9}>暂无表扬记录</td></tr>}
            </tbody>
          </table>
        </div>
      </section>

      <aside className="rounded-lg border border-line bg-surface p-4 xl:sticky xl:top-24 xl:self-start">
        <h2 className="text-lg font-semibold">{selectedId ? "编辑表扬登记" : "新增表扬登记"}</h2>
        <div className="mt-4 grid gap-3">
          <Field label="表扬日期"><input type="date" className="rounded-lg border border-line px-3 py-2 text-sm" value={draft.praiseDate} onChange={(event) => setDraft({ ...draft, praiseDate: event.target.value })} /></Field>
          <Field label="表扬类别"><input className="rounded-lg border border-line px-3 py-2 text-sm" value={draft.praiseType || ""} onChange={(event) => setDraft({ ...draft, praiseType: event.target.value })} /></Field>
          <Field label="表扬方式"><select className="rounded-lg border border-line px-3 py-2 text-sm" value={draft.praiseMethod || ""} onChange={(event) => setDraft({ ...draft, praiseMethod: event.target.value })}><option>电话表扬</option><option>锦旗</option><option>感谢信</option><option>微信</option><option>现场</option></select></Field>
          <Field label="科室"><input className="rounded-lg border border-line px-3 py-2 text-sm" value={draft.departmentName || ""} onChange={(event) => setDraft({ ...draft, departmentName: event.target.value })} /></Field>
          <Field label="医护人员"><input className="rounded-lg border border-line px-3 py-2 text-sm" value={draft.staffName || ""} onChange={(event) => setDraft({ ...draft, staffName: event.target.value })} /></Field>
          <Field label="患者姓名"><input className="rounded-lg border border-line px-3 py-2 text-sm" value={draft.patientName || ""} onChange={(event) => setDraft({ ...draft, patientName: event.target.value })} /></Field>
          <div className="grid grid-cols-2 gap-3">
            <Field label="数量"><input type="number" min={1} className="rounded-lg border border-line px-3 py-2 text-sm" value={draft.quantity} onChange={(event) => setDraft({ ...draft, quantity: Number(event.target.value) || 1 })} /></Field>
            <Field label="奖励金额"><input type="number" min={0} step="0.01" className="rounded-lg border border-line px-3 py-2 text-sm" value={draft.rewardAmount} onChange={(event) => setDraft({ ...draft, rewardAmount: Number(event.target.value) || 0 })} /></Field>
          </div>
          <Field label="内容"><textarea className="rounded-lg border border-line px-3 py-2 text-sm min-h-24" value={draft.content || ""} onChange={(event) => setDraft({ ...draft, content: event.target.value })} /></Field>
          <Field label="备注"><textarea className="rounded-lg border border-line px-3 py-2 text-sm min-h-20" value={draft.remark || ""} onChange={(event) => setDraft({ ...draft, remark: event.target.value })} /></Field>
          <button className="rounded-lg bg-primary px-4 py-3 text-sm font-medium text-white" onClick={save}>保存登记</button>
        </div>
      </aside>
    </div>
  )
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return <label className="grid gap-1 text-sm"><span className="text-muted">{label}</span>{children}</label>
}

function statusLabel(status: string) {
  return ({ draft: "草稿", confirmed: "已确认", archived: "已归档" } as Record<string, string>)[status] || status
}
