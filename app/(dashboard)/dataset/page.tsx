"use client"

import { Plus, Database, FileText, Calendar, MoreHorizontal, Eye, Edit, Trash2 } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

interface Dataset {
  id: string
  name: string
  description: string
  recordCount: number
  formCount: number
  createdAt: string
  updatedAt: string
  status: "active" | "archived"
}

const mockDatasets: Dataset[] = [
  {
    id: "DS001",
    name: "高血压随访研究",
    description: "高血压患者长期随访数据采集",
    recordCount: 1250,
    formCount: 5,
    createdAt: "2023-06-01",
    updatedAt: "2024-03-20",
    status: "active",
  },
  {
    id: "DS002",
    name: "糖尿病管理研究",
    description: "2型糖尿病患者血糖管理跟踪",
    recordCount: 890,
    formCount: 4,
    createdAt: "2023-08-15",
    updatedAt: "2024-03-18",
    status: "active",
  },
  {
    id: "DS003",
    name: "心血管疾病筛查",
    description: "心血管疾病高危人群筛查数据",
    recordCount: 2100,
    formCount: 6,
    createdAt: "2023-03-10",
    updatedAt: "2024-02-28",
    status: "active",
  },
  {
    id: "DS004",
    name: "老年痴呆早期筛查",
    description: "认知功能评估和早期筛查",
    recordCount: 456,
    formCount: 3,
    createdAt: "2024-01-05",
    updatedAt: "2024-03-15",
    status: "active",
  },
]

export default function DatasetPage() {
  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold">数据集</h1>
          <p className="text-muted-foreground">管理研究数据集和采集表单</p>
        </div>
        <Button>
          <Plus className="h-4 w-4 mr-2" />
          新建数据集
        </Button>
      </div>

      {/* 数据集统计 */}
      <div className="grid gap-4 md:grid-cols-3 mb-6">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">数据集总数</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{mockDatasets.length}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">总记录数</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {mockDatasets.reduce((sum, ds) => sum + ds.recordCount, 0).toLocaleString()}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">表单模板数</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {mockDatasets.reduce((sum, ds) => sum + ds.formCount, 0)}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* 数据集列表 */}
      <div className="grid gap-4 md:grid-cols-2">
        {mockDatasets.map((dataset) => (
          <Card key={dataset.id} className="hover:shadow-md transition-shadow">
            <CardHeader>
              <div className="flex items-start justify-between">
                <div className="flex items-center">
                  <div className="p-2 rounded-lg bg-primary/10 text-primary mr-3">
                    <Database className="h-5 w-5" />
                  </div>
                  <div>
                    <CardTitle className="text-lg">{dataset.name}</CardTitle>
                    <CardDescription>{dataset.description}</CardDescription>
                  </div>
                </div>
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button variant="ghost" size="icon">
                      <MoreHorizontal className="h-4 w-4" />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <DropdownMenuItem>
                      <Eye className="h-4 w-4 mr-2" />
                      查看数据
                    </DropdownMenuItem>
                    <DropdownMenuItem>
                      <Edit className="h-4 w-4 mr-2" />
                      编辑设置
                    </DropdownMenuItem>
                    <DropdownMenuItem className="text-destructive">
                      <Trash2 className="h-4 w-4 mr-2" />
                      删除
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </div>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-2 gap-4 text-sm">
                <div className="flex items-center text-muted-foreground">
                  <Database className="h-4 w-4 mr-2" />
                  {dataset.recordCount.toLocaleString()} 条记录
                </div>
                <div className="flex items-center text-muted-foreground">
                  <FileText className="h-4 w-4 mr-2" />
                  {dataset.formCount} 个表单
                </div>
                <div className="flex items-center text-muted-foreground col-span-2">
                  <Calendar className="h-4 w-4 mr-2" />
                  最后更新: {dataset.updatedAt}
                </div>
              </div>
              <div className="mt-4 flex gap-2">
                <Button size="sm" className="flex-1">
                  录入数据
                </Button>
                <Button variant="outline" size="sm" className="flex-1">
                  查看数据
                </Button>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  )
}
