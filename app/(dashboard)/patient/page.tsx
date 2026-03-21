"use client"

import { useState } from "react"
import { Plus, Search, MoreHorizontal, Eye, Edit, Trash2 } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

interface Patient {
  id: string
  name: string
  gender: string
  age: number
  phone: string
  diagnosis: string
  admissionDate: string
  status: "active" | "discharged" | "followup"
}

const mockPatients: Patient[] = [
  { id: "P001", name: "张三", gender: "男", age: 45, phone: "138****1234", diagnosis: "高血压", admissionDate: "2024-01-15", status: "active" },
  { id: "P002", name: "李四", gender: "女", age: 32, phone: "139****5678", diagnosis: "糖尿病", admissionDate: "2024-02-20", status: "followup" },
  { id: "P003", name: "王五", gender: "男", age: 58, phone: "137****9012", diagnosis: "冠心病", admissionDate: "2024-03-05", status: "discharged" },
  { id: "P004", name: "赵六", gender: "女", age: 41, phone: "136****3456", diagnosis: "高血压", admissionDate: "2024-03-10", status: "active" },
  { id: "P005", name: "孙七", gender: "男", age: 55, phone: "135****7890", diagnosis: "糖尿病", admissionDate: "2024-03-15", status: "active" },
]

const statusMap = {
  active: { label: "在院", className: "bg-green-100 text-green-800" },
  discharged: { label: "出院", className: "bg-gray-100 text-gray-800" },
  followup: { label: "随访中", className: "bg-blue-100 text-blue-800" },
}

export default function PatientPage() {
  const [searchKeyword, setSearchKeyword] = useState("")
  const [patients] = useState<Patient[]>(mockPatients)

  const filteredPatients = patients.filter(
    (p) =>
      p.name.includes(searchKeyword) ||
      p.id.includes(searchKeyword) ||
      p.diagnosis.includes(searchKeyword)
  )

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold">患者管理</h1>
          <p className="text-muted-foreground">管理患者信息和病历记录</p>
        </div>
        <Button>
          <Plus className="h-4 w-4 mr-2" />
          新增患者
        </Button>
      </div>

      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle className="text-lg">患者列表</CardTitle>
            <div className="relative w-64">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="搜索患者..."
                value={searchKeyword}
                onChange={(e) => setSearchKeyword(e.target.value)}
                className="pl-10"
              />
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b">
                  <th className="text-left py-3 px-4 font-medium text-muted-foreground">患者ID</th>
                  <th className="text-left py-3 px-4 font-medium text-muted-foreground">姓名</th>
                  <th className="text-left py-3 px-4 font-medium text-muted-foreground">性别</th>
                  <th className="text-left py-3 px-4 font-medium text-muted-foreground">年龄</th>
                  <th className="text-left py-3 px-4 font-medium text-muted-foreground">联系电话</th>
                  <th className="text-left py-3 px-4 font-medium text-muted-foreground">诊断</th>
                  <th className="text-left py-3 px-4 font-medium text-muted-foreground">入院日期</th>
                  <th className="text-left py-3 px-4 font-medium text-muted-foreground">状态</th>
                  <th className="text-left py-3 px-4 font-medium text-muted-foreground">操作</th>
                </tr>
              </thead>
              <tbody>
                {filteredPatients.map((patient) => (
                  <tr key={patient.id} className="border-b hover:bg-muted/50">
                    <td className="py-3 px-4 font-mono text-sm">{patient.id}</td>
                    <td className="py-3 px-4 font-medium">{patient.name}</td>
                    <td className="py-3 px-4">{patient.gender}</td>
                    <td className="py-3 px-4">{patient.age}</td>
                    <td className="py-3 px-4">{patient.phone}</td>
                    <td className="py-3 px-4">{patient.diagnosis}</td>
                    <td className="py-3 px-4">{patient.admissionDate}</td>
                    <td className="py-3 px-4">
                      <span className={`px-2 py-1 rounded-full text-xs ${statusMap[patient.status].className}`}>
                        {statusMap[patient.status].label}
                      </span>
                    </td>
                    <td className="py-3 px-4">
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button variant="ghost" size="icon">
                            <MoreHorizontal className="h-4 w-4" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem>
                            <Eye className="h-4 w-4 mr-2" />
                            查看详情
                          </DropdownMenuItem>
                          <DropdownMenuItem>
                            <Edit className="h-4 w-4 mr-2" />
                            编辑
                          </DropdownMenuItem>
                          <DropdownMenuItem className="text-destructive">
                            <Trash2 className="h-4 w-4 mr-2" />
                            删除
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {filteredPatients.length === 0 && (
            <div className="text-center py-12 text-muted-foreground">
              <p>未找到匹配的患者记录</p>
            </div>
          )}

          {/* 分页 */}
          <div className="flex items-center justify-between mt-4">
            <p className="text-sm text-muted-foreground">
              共 {filteredPatients.length} 条记录
            </p>
            <div className="flex gap-2">
              <Button variant="outline" size="sm" disabled>
                上一页
              </Button>
              <Button variant="outline" size="sm" disabled>
                下一页
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
