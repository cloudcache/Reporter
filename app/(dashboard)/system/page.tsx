"use client"

import { useState } from "react"
import { Settings, Users, Shield, Bell, Database, Save } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Checkbox } from "@/components/ui/checkbox"

const tabs = [
  { id: "general", label: "通用设置", icon: Settings },
  { id: "users", label: "用户管理", icon: Users },
  { id: "roles", label: "角色权限", icon: Shield },
  { id: "notifications", label: "通知设置", icon: Bell },
  { id: "backup", label: "数据备份", icon: Database },
]

export default function SystemPage() {
  const [activeTab, setActiveTab] = useState("general")

  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-bold">系统设置</h1>
        <p className="text-muted-foreground">管理系统配置和用户权限</p>
      </div>

      <div className="flex gap-6">
        {/* 侧边导航 */}
        <div className="w-48 shrink-0">
          <nav className="space-y-1">
            {tabs.map((tab) => {
              const Icon = tab.icon
              return (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id)}
                  className={`w-full flex items-center px-3 py-2 text-sm rounded-lg transition-colors ${
                    activeTab === tab.id
                      ? "bg-primary text-primary-foreground"
                      : "text-muted-foreground hover:bg-muted"
                  }`}
                >
                  <Icon className="h-4 w-4 mr-2" />
                  {tab.label}
                </button>
              )
            })}
          </nav>
        </div>

        {/* 内容区域 */}
        <div className="flex-1">
          {activeTab === "general" && (
            <Card>
              <CardHeader>
                <CardTitle>通用设置</CardTitle>
                <CardDescription>配置系统基本信息和显示选项</CardDescription>
              </CardHeader>
              <CardContent className="space-y-6">
                <div className="space-y-2">
                  <Label htmlFor="systemName">系统名称</Label>
                  <Input id="systemName" defaultValue="医疗研究管理系统" />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="orgName">机构名称</Label>
                  <Input id="orgName" defaultValue="XX医院研究中心" />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="adminEmail">管理员邮箱</Label>
                  <Input id="adminEmail" type="email" defaultValue="admin@hospital.com" />
                </div>
                <div className="flex items-center space-x-2">
                  <Checkbox id="maintenance" />
                  <Label htmlFor="maintenance" className="font-normal">启用维护模式</Label>
                </div>
                <Button>
                  <Save className="h-4 w-4 mr-2" />
                  保存设置
                </Button>
              </CardContent>
            </Card>
          )}

          {activeTab === "users" && (
            <Card>
              <CardHeader>
                <CardTitle>用户管理</CardTitle>
                <CardDescription>管理系统用户账号</CardDescription>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  <div className="flex justify-end">
                    <Button>添加用户</Button>
                  </div>
                  <table className="w-full">
                    <thead>
                      <tr className="border-b">
                        <th className="text-left py-3 px-4 font-medium text-muted-foreground">用户名</th>
                        <th className="text-left py-3 px-4 font-medium text-muted-foreground">姓名</th>
                        <th className="text-left py-3 px-4 font-medium text-muted-foreground">角色</th>
                        <th className="text-left py-3 px-4 font-medium text-muted-foreground">状态</th>
                        <th className="text-left py-3 px-4 font-medium text-muted-foreground">操作</th>
                      </tr>
                    </thead>
                    <tbody>
                      <tr className="border-b">
                        <td className="py-3 px-4">admin</td>
                        <td className="py-3 px-4">系统管理员</td>
                        <td className="py-3 px-4">管理员</td>
                        <td className="py-3 px-4">
                          <span className="px-2 py-1 rounded-full text-xs bg-green-100 text-green-800">启用</span>
                        </td>
                        <td className="py-3 px-4">
                          <Button variant="ghost" size="sm">编辑</Button>
                        </td>
                      </tr>
                      <tr className="border-b">
                        <td className="py-3 px-4">doctor1</td>
                        <td className="py-3 px-4">张医生</td>
                        <td className="py-3 px-4">医生</td>
                        <td className="py-3 px-4">
                          <span className="px-2 py-1 rounded-full text-xs bg-green-100 text-green-800">启用</span>
                        </td>
                        <td className="py-3 px-4">
                          <Button variant="ghost" size="sm">编辑</Button>
                        </td>
                      </tr>
                    </tbody>
                  </table>
                </div>
              </CardContent>
            </Card>
          )}

          {activeTab === "roles" && (
            <Card>
              <CardHeader>
                <CardTitle>角色权限</CardTitle>
                <CardDescription>配置角色和权限设置</CardDescription>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  {["管理员", "医生", "护士", "研究员"].map((role) => (
                    <div key={role} className="p-4 border rounded-lg">
                      <div className="flex items-center justify-between mb-3">
                        <span className="font-medium">{role}</span>
                        <Button variant="outline" size="sm">编辑权限</Button>
                      </div>
                      <div className="flex flex-wrap gap-2">
                        {["查看患者", "编辑患者", "数据导出", "系统设置"].slice(0, role === "管理员" ? 4 : 2).map((perm) => (
                          <span key={perm} className="px-2 py-1 text-xs bg-muted rounded">
                            {perm}
                          </span>
                        ))}
                      </div>
                    </div>
                  ))}
                </div>
              </CardContent>
            </Card>
          )}

          {activeTab === "notifications" && (
            <Card>
              <CardHeader>
                <CardTitle>通知设置</CardTitle>
                <CardDescription>配置系统通知和提醒</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="flex items-center justify-between p-4 border rounded-lg">
                  <div>
                    <p className="font-medium">随访提醒</p>
                    <p className="text-sm text-muted-foreground">在随访日期前发送提醒</p>
                  </div>
                  <Checkbox defaultChecked />
                </div>
                <div className="flex items-center justify-between p-4 border rounded-lg">
                  <div>
                    <p className="font-medium">数据更新通知</p>
                    <p className="text-sm text-muted-foreground">当数据集有更新时发送通知</p>
                  </div>
                  <Checkbox defaultChecked />
                </div>
                <div className="flex items-center justify-between p-4 border rounded-lg">
                  <div>
                    <p className="font-medium">系统维护通知</p>
                    <p className="text-sm text-muted-foreground">系统维护前发送通知</p>
                  </div>
                  <Checkbox />
                </div>
              </CardContent>
            </Card>
          )}

          {activeTab === "backup" && (
            <Card>
              <CardHeader>
                <CardTitle>数据备份</CardTitle>
                <CardDescription>管理系统数据备份和恢复</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="p-4 bg-muted/50 rounded-lg">
                  <p className="text-sm text-muted-foreground">上次备份时间</p>
                  <p className="font-medium">2024-03-20 03:00:00</p>
                </div>
                <div className="flex gap-4">
                  <Button>立即备份</Button>
                  <Button variant="outline">恢复数据</Button>
                </div>
                <div className="space-y-2">
                  <Label>自动备份频率</Label>
                  <select className="w-full h-9 rounded-md border border-input bg-transparent px-3 text-sm">
                    <option>每天</option>
                    <option>每周</option>
                    <option>每月</option>
                  </select>
                </div>
              </CardContent>
            </Card>
          )}
        </div>
      </div>
    </div>
  )
}
