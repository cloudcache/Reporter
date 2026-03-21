"use client"

import { useState } from "react"
import { Plus, Calendar, Clock, CheckCircle, AlertCircle, User } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"

interface FollowUp {
  id: string
  patientId: string
  patientName: string
  type: string
  scheduledDate: string
  status: "pending" | "completed" | "overdue" | "cancelled"
  notes?: string
}

const mockFollowUps: FollowUp[] = [
  { id: "F001", patientId: "P001", patientName: "张三", type: "术后复查", scheduledDate: "2024-03-25", status: "pending" },
  { id: "F002", patientId: "P002", patientName: "李四", type: "用药随访", scheduledDate: "2024-03-20", status: "completed", notes: "血糖控制良好" },
  { id: "F003", patientId: "P003", patientName: "王五", type: "定期检查", scheduledDate: "2024-03-15", status: "overdue" },
  { id: "F004", patientId: "P004", patientName: "赵六", type: "术后复查", scheduledDate: "2024-03-28", status: "pending" },
  { id: "F005", patientId: "P001", patientName: "张三", type: "电话随访", scheduledDate: "2024-03-22", status: "pending" },
]

const statusConfig = {
  pending: { label: "待完成", icon: Clock, className: "text-amber-600 bg-amber-50" },
  completed: { label: "已完成", icon: CheckCircle, className: "text-green-600 bg-green-50" },
  overdue: { label: "已逾期", icon: AlertCircle, className: "text-red-600 bg-red-50" },
  cancelled: { label: "已取消", icon: AlertCircle, className: "text-gray-600 bg-gray-50" },
}

export default function FollowUpPage() {
  const [filter, setFilter] = useState<"all" | "pending" | "completed" | "overdue">("all")

  const filteredFollowUps = mockFollowUps.filter((f) => {
    if (filter === "all") return true
    return f.status === filter
  })

  const stats = {
    total: mockFollowUps.length,
    pending: mockFollowUps.filter((f) => f.status === "pending").length,
    completed: mockFollowUps.filter((f) => f.status === "completed").length,
    overdue: mockFollowUps.filter((f) => f.status === "overdue").length,
  }

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold">随访管理</h1>
          <p className="text-muted-foreground">管理患者随访计划和记录</p>
        </div>
        <Button>
          <Plus className="h-4 w-4 mr-2" />
          新建随访
        </Button>
      </div>

      {/* 统计卡片 */}
      <div className="grid gap-4 md:grid-cols-4 mb-6">
        <Card
          className={`cursor-pointer transition-colors ${filter === "all" ? "ring-2 ring-primary" : ""}`}
          onClick={() => setFilter("all")}
        >
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">全部随访</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats.total}</div>
          </CardContent>
        </Card>
        <Card
          className={`cursor-pointer transition-colors ${filter === "pending" ? "ring-2 ring-primary" : ""}`}
          onClick={() => setFilter("pending")}
        >
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-amber-600">待完成</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-amber-600">{stats.pending}</div>
          </CardContent>
        </Card>
        <Card
          className={`cursor-pointer transition-colors ${filter === "completed" ? "ring-2 ring-primary" : ""}`}
          onClick={() => setFilter("completed")}
        >
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-green-600">已完成</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-green-600">{stats.completed}</div>
          </CardContent>
        </Card>
        <Card
          className={`cursor-pointer transition-colors ${filter === "overdue" ? "ring-2 ring-primary" : ""}`}
          onClick={() => setFilter("overdue")}
        >
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-red-600">已逾期</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-red-600">{stats.overdue}</div>
          </CardContent>
        </Card>
      </div>

      {/* 随访列表 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">随访列表</CardTitle>
          <CardDescription>
            {filter === "all" ? "显示全部随访" : `筛选: ${statusConfig[filter as keyof typeof statusConfig]?.label}`}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            {filteredFollowUps.map((followUp) => {
              const config = statusConfig[followUp.status]
              const StatusIcon = config.icon
              return (
                <div
                  key={followUp.id}
                  className="flex items-center p-4 rounded-lg border hover:bg-muted/50 transition-colors"
                >
                  <div className={`p-2 rounded-full ${config.className} mr-4`}>
                    <StatusIcon className="h-5 w-5" />
                  </div>
                  <div className="flex-1">
                    <div className="flex items-center gap-3 mb-1">
                      <span className="font-medium">{followUp.type}</span>
                      <span className={`px-2 py-0.5 rounded text-xs ${config.className}`}>
                        {config.label}
                      </span>
                    </div>
                    <div className="flex items-center gap-4 text-sm text-muted-foreground">
                      <span className="flex items-center">
                        <User className="h-3.5 w-3.5 mr-1" />
                        {followUp.patientName} ({followUp.patientId})
                      </span>
                      <span className="flex items-center">
                        <Calendar className="h-3.5 w-3.5 mr-1" />
                        {followUp.scheduledDate}
                      </span>
                    </div>
                    {followUp.notes && (
                      <p className="text-sm text-muted-foreground mt-1">
                        备注: {followUp.notes}
                      </p>
                    )}
                  </div>
                  <div className="flex gap-2">
                    {followUp.status === "pending" && (
                      <Button size="sm">完成随访</Button>
                    )}
                    <Button variant="outline" size="sm">
                      查看详情
                    </Button>
                  </div>
                </div>
              )
            })}
          </div>

          {filteredFollowUps.length === 0 && (
            <div className="text-center py-12 text-muted-foreground">
              <Calendar className="h-12 w-12 mx-auto mb-4 opacity-50" />
              <p>暂无随访记录</p>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
