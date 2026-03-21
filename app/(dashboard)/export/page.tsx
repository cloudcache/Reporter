"use client"

import { useState } from "react"
import { Download, FileSpreadsheet, FileText, Calendar, CheckCircle } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Label } from "@/components/ui/label"
import { Checkbox } from "@/components/ui/checkbox"

const datasets = [
  { id: "DS001", name: "高血压随访研究", recordCount: 1250 },
  { id: "DS002", name: "糖尿病管理研究", recordCount: 890 },
  { id: "DS003", name: "心血管疾病筛查", recordCount: 2100 },
  { id: "DS004", name: "老年痴呆早期筛查", recordCount: 456 },
]

const exportHistory = [
  { id: "E001", dataset: "高血压随访研究", format: "Excel", date: "2024-03-20 14:30", records: 1250, status: "completed" },
  { id: "E002", dataset: "糖尿病管理研究", format: "CSV", date: "2024-03-18 10:15", records: 890, status: "completed" },
  { id: "E003", dataset: "心血管疾病筛查", format: "Excel", date: "2024-03-15 09:00", records: 2100, status: "completed" },
]

export default function ExportPage() {
  const [selectedDatasets, setSelectedDatasets] = useState<string[]>([])
  const [exportFormat, setExportFormat] = useState<"excel" | "csv">("excel")
  const [dateRange, setDateRange] = useState({ start: "", end: "" })

  const toggleDataset = (id: string) => {
    setSelectedDatasets((prev) =>
      prev.includes(id) ? prev.filter((d) => d !== id) : [...prev, id]
    )
  }

  const handleExport = () => {
    // 实际导出逻辑
    alert(`导出 ${selectedDatasets.length} 个数据集，格式: ${exportFormat.toUpperCase()}`)
  }

  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-bold">数据导出</h1>
        <p className="text-muted-foreground">导出研究数据用于分析和报告</p>
      </div>

      <div className="grid gap-6 lg:grid-cols-3">
        {/* 导出配置 */}
        <div className="lg:col-span-2 space-y-6">
          {/* 选择数据集 */}
          <Card>
            <CardHeader>
              <CardTitle className="text-lg">选择数据集</CardTitle>
              <CardDescription>选择要导出的数据集</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-3">
                {datasets.map((dataset) => (
                  <div
                    key={dataset.id}
                    className={`flex items-center justify-between p-4 border rounded-lg cursor-pointer transition-colors ${
                      selectedDatasets.includes(dataset.id)
                        ? "border-primary bg-primary/5"
                        : "hover:bg-muted/50"
                    }`}
                    onClick={() => toggleDataset(dataset.id)}
                  >
                    <div className="flex items-center">
                      <Checkbox
                        checked={selectedDatasets.includes(dataset.id)}
                        className="mr-3"
                      />
                      <div>
                        <p className="font-medium">{dataset.name}</p>
                        <p className="text-sm text-muted-foreground">
                          {dataset.recordCount.toLocaleString()} 条记录
                        </p>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>

          {/* 导出选项 */}
          <Card>
            <CardHeader>
              <CardTitle className="text-lg">导出选项</CardTitle>
              <CardDescription>配置导出格式和筛选条件</CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              {/* 导出格式 */}
              <div className="space-y-3">
                <Label>导出格式</Label>
                <div className="flex gap-4">
                  <div
                    className={`flex-1 p-4 border rounded-lg cursor-pointer transition-colors ${
                      exportFormat === "excel"
                        ? "border-primary bg-primary/5"
                        : "hover:bg-muted/50"
                    }`}
                    onClick={() => setExportFormat("excel")}
                  >
                    <div className="flex items-center">
                      <FileSpreadsheet className="h-8 w-8 text-green-600 mr-3" />
                      <div>
                        <p className="font-medium">Excel (.xlsx)</p>
                        <p className="text-sm text-muted-foreground">
                          适合数据分析和报表
                        </p>
                      </div>
                    </div>
                  </div>
                  <div
                    className={`flex-1 p-4 border rounded-lg cursor-pointer transition-colors ${
                      exportFormat === "csv"
                        ? "border-primary bg-primary/5"
                        : "hover:bg-muted/50"
                    }`}
                    onClick={() => setExportFormat("csv")}
                  >
                    <div className="flex items-center">
                      <FileText className="h-8 w-8 text-blue-600 mr-3" />
                      <div>
                        <p className="font-medium">CSV (.csv)</p>
                        <p className="text-sm text-muted-foreground">
                          通用格式，便于导入其他系统
                        </p>
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              {/* 日期范围 */}
              <div className="space-y-3">
                <Label>日期范围（可选）</Label>
                <div className="flex items-center gap-4">
                  <div className="flex-1">
                    <Label className="text-xs text-muted-foreground">开始日期</Label>
                    <input
                      type="date"
                      className="w-full h-9 rounded-md border border-input bg-transparent px-3 text-sm mt-1"
                      value={dateRange.start}
                      onChange={(e) =>
                        setDateRange({ ...dateRange, start: e.target.value })
                      }
                    />
                  </div>
                  <div className="flex-1">
                    <Label className="text-xs text-muted-foreground">结束日期</Label>
                    <input
                      type="date"
                      className="w-full h-9 rounded-md border border-input bg-transparent px-3 text-sm mt-1"
                      value={dateRange.end}
                      onChange={(e) =>
                        setDateRange({ ...dateRange, end: e.target.value })
                      }
                    />
                  </div>
                </div>
              </div>

              <Button
                className="w-full"
                disabled={selectedDatasets.length === 0}
                onClick={handleExport}
              >
                <Download className="h-4 w-4 mr-2" />
                导出数据
              </Button>
            </CardContent>
          </Card>
        </div>

        {/* 导出历史 */}
        <div>
          <Card>
            <CardHeader>
              <CardTitle className="text-lg">导出历史</CardTitle>
              <CardDescription>最近的导出记录</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {exportHistory.map((item) => (
                  <div key={item.id} className="p-3 border rounded-lg">
                    <div className="flex items-start justify-between mb-2">
                      <p className="font-medium text-sm">{item.dataset}</p>
                      <CheckCircle className="h-4 w-4 text-green-600" />
                    </div>
                    <div className="space-y-1 text-xs text-muted-foreground">
                      <div className="flex items-center">
                        <FileSpreadsheet className="h-3 w-3 mr-1" />
                        {item.format} · {item.records.toLocaleString()} 条
                      </div>
                      <div className="flex items-center">
                        <Calendar className="h-3 w-3 mr-1" />
                        {item.date}
                      </div>
                    </div>
                    <Button variant="link" className="h-auto p-0 text-xs mt-2">
                      重新下载
                    </Button>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  )
}
