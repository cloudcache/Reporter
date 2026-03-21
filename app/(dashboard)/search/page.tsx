"use client"

import { useState } from "react"
import { Search, Filter, X, UserCircle } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Label } from "@/components/ui/label"

interface SearchResult {
  id: string
  name: string
  gender: string
  age: number
  diagnosis: string
  admissionDate: string
}

const mockResults: SearchResult[] = [
  { id: "P001", name: "张三", gender: "男", age: 45, diagnosis: "高血压", admissionDate: "2024-01-15" },
  { id: "P002", name: "李四", gender: "女", age: 32, diagnosis: "糖尿病", admissionDate: "2024-02-20" },
  { id: "P003", name: "王五", gender: "男", age: 58, diagnosis: "冠心病", admissionDate: "2024-03-05" },
  { id: "P004", name: "赵六", gender: "女", age: 41, diagnosis: "高血压", admissionDate: "2024-03-10" },
]

export default function SearchPage() {
  const [keyword, setKeyword] = useState("")
  const [showFilters, setShowFilters] = useState(false)
  const [results, setResults] = useState<SearchResult[]>([])
  const [hasSearched, setHasSearched] = useState(false)

  const handleSearch = () => {
    setHasSearched(true)
    if (!keyword.trim()) {
      setResults(mockResults)
    } else {
      const filtered = mockResults.filter(
        (item) =>
          item.name.includes(keyword) ||
          item.id.includes(keyword) ||
          item.diagnosis.includes(keyword)
      )
      setResults(filtered)
    }
  }

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") {
      handleSearch()
    }
  }

  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-bold">搜索</h1>
        <p className="text-muted-foreground">搜索患者、随访记录或数据</p>
      </div>

      {/* 搜索框 */}
      <Card className="mb-6">
        <CardContent className="pt-6">
          <div className="flex gap-4">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="输入患者姓名、ID 或诊断..."
                value={keyword}
                onChange={(e) => setKeyword(e.target.value)}
                onKeyDown={handleKeyPress}
                className="pl-10"
              />
            </div>
            <Button onClick={handleSearch}>搜索</Button>
            <Button
              variant="outline"
              onClick={() => setShowFilters(!showFilters)}
            >
              <Filter className="h-4 w-4 mr-2" />
              高级筛选
            </Button>
          </div>

          {/* 高级筛选面板 */}
          {showFilters && (
            <div className="mt-4 p-4 border rounded-lg bg-muted/30">
              <div className="flex items-center justify-between mb-4">
                <span className="font-medium">筛选条件</span>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setShowFilters(false)}
                >
                  <X className="h-4 w-4" />
                </Button>
              </div>
              <div className="grid gap-4 md:grid-cols-3">
                <div className="space-y-2">
                  <Label>性别</Label>
                  <select className="w-full h-9 rounded-md border border-input bg-transparent px-3 text-sm">
                    <option value="">全部</option>
                    <option value="male">男</option>
                    <option value="female">女</option>
                  </select>
                </div>
                <div className="space-y-2">
                  <Label>年龄范围</Label>
                  <div className="flex gap-2">
                    <Input placeholder="最小" type="number" />
                    <span className="self-center">-</span>
                    <Input placeholder="最大" type="number" />
                  </div>
                </div>
                <div className="space-y-2">
                  <Label>入院日期</Label>
                  <Input type="date" />
                </div>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* 搜索结果 */}
      {hasSearched && (
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">
              搜索结果 ({results.length})
            </CardTitle>
          </CardHeader>
          <CardContent>
            {results.length === 0 ? (
              <div className="text-center py-12 text-muted-foreground">
                <Search className="h-12 w-12 mx-auto mb-4 opacity-50" />
                <p>未找到匹配的结果</p>
                <p className="text-sm">请尝试其他搜索关键词</p>
              </div>
            ) : (
              <div className="space-y-3">
                {results.map((item) => (
                  <div
                    key={item.id}
                    className="flex items-center p-4 rounded-lg border hover:bg-muted/50 cursor-pointer transition-colors"
                  >
                    <UserCircle className="h-10 w-10 text-muted-foreground mr-4" />
                    <div className="flex-1">
                      <div className="flex items-center gap-3">
                        <span className="font-medium">{item.name}</span>
                        <span className="text-sm text-muted-foreground">
                          {item.id}
                        </span>
                        <span className="text-sm px-2 py-0.5 rounded bg-muted">
                          {item.gender} / {item.age}岁
                        </span>
                      </div>
                      <div className="text-sm text-muted-foreground mt-1">
                        诊断: {item.diagnosis} | 入院日期: {item.admissionDate}
                      </div>
                    </div>
                    <Button variant="outline" size="sm">
                      查看详情
                    </Button>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {!hasSearched && (
        <div className="text-center py-20 text-muted-foreground">
          <Search className="h-16 w-16 mx-auto mb-4 opacity-30" />
          <p className="text-lg">输入关键词开始搜索</p>
          <p className="text-sm">支持按患者姓名、ID、诊断等进行搜索</p>
        </div>
      )}
    </div>
  )
}
