import { useMemo, useState } from "react"
import { authedJson } from "../lib/auth"

export interface IntegrationChannel {
  id: string
  kind: string
  name: string
  enabled: boolean
}

export interface FollowupSipEndpoint {
  id: string
  name: string
}

interface FollowupTarget {
  patientId: string
  patientName?: string
  patientPhone?: string
  formTemplateId?: string
  projectId?: string
  title?: string
}

interface ShareLink {
  id: string
  url: string
  channel: string
}

interface Props {
  target: FollowupTarget
  channels: IntegrationChannel[]
  sipEndpoints: FollowupSipEndpoint[]
  onPhone?: () => void
  onMessage?: (message: string) => void
  className?: string
}

const supportedChannels = ["sms", "wechat", "qq", "web"]
const channelLabels: Record<string, string> = { sms: "短信", wechat: "微信", qq: "QQ", web: "Web" }

export function FollowupChannelActions({ target, channels, sipEndpoints, onPhone, onMessage, className = "" }: Props) {
  const [busy, setBusy] = useState("")
  const enabledChannels = useMemo(() => {
    const byKind = new Map<string, IntegrationChannel>()
    channels.forEach((channel) => {
      if (channel.enabled && supportedChannels.includes(channel.kind) && !byKind.has(channel.kind)) {
        byKind.set(channel.kind, channel)
      }
    })
    return Array.from(byKind.values())
  }, [channels])
  const canPhone = Boolean(target.patientPhone && sipEndpoints.length > 0 && onPhone)

  async function copyLink(url: string) {
    if (typeof navigator !== "undefined" && navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(url)
    }
  }

  async function createLink(channel: string) {
    const formTemplateId = target.formTemplateId || "outpatient-satisfaction"
    try {
      setBusy(channel)
      const created = await authedJson<ShareLink>("/api/v1/survey-share-links", {
        method: "POST",
        body: JSON.stringify({
          projectId: target.projectId || "",
          formTemplateId,
          title: target.title || `${target.patientName || "患者"}随访问卷`,
          channel,
          config: {
            patientId: target.patientId,
            patientName: target.patientName || "",
            patientPhone: target.patientPhone || "",
            deliveryChannel: channel,
          },
        }),
      })
      const absoluteUrl = typeof window === "undefined" ? created.url : new URL(created.url, window.location.origin).href
      await copyLink(absoluteUrl)
      if (channel === "web" && typeof window !== "undefined") window.open(absoluteUrl, "_blank", "noopener,noreferrer")
      onMessage?.(`已生成${channelLabels[channel] || channel}随访链接并复制：${absoluteUrl}`)
    } catch (error) {
      onMessage?.(`${channelLabels[channel] || channel}随访链接生成失败：${error instanceof Error ? error.message : "未知错误"}`)
    } finally {
      setBusy("")
    }
  }

  if (!canPhone && enabledChannels.length === 0) {
    return <span className={`text-xs text-muted ${className}`}>未启用随访接口</span>
  }

  return (
    <div className={`inline-flex flex-wrap justify-end gap-2 ${className}`} onClick={(event) => event.stopPropagation()}>
      {canPhone && (
        <button className="rounded-lg border border-line px-3 py-1.5 text-xs font-medium text-primary hover:border-primary" onClick={onPhone}>
          电话
        </button>
      )}
      {enabledChannels.map((channel) => (
        <button
          key={channel.kind}
          className="rounded-lg border border-line px-3 py-1.5 text-xs font-medium hover:border-primary disabled:bg-gray-100 disabled:text-muted"
          disabled={busy === channel.kind}
          onClick={() => createLink(channel.kind)}
          title={channel.name}
        >
          {busy === channel.kind ? "生成中" : channelLabels[channel.kind] || channel.name}
        </button>
      ))}
    </div>
  )
}
