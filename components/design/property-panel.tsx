"use client"

import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Checkbox } from "@/components/ui/checkbox"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { useDesignStore } from "@/hooks/use-design-store"
import { DesignComponentType } from "@/types"
import { Plus, Trash2 } from "lucide-react"
import { useState } from "react"

export function PropertyPanel() {
  const { canvasItems, selectedItemId, updateItem } = useDesignStore()

  // 在画布和容器内查找选中的项目
  const findSelectedItem = () => {
    for (const item of canvasItems) {
      if (item.id === selectedItemId) return item
      if (item.children && Array.isArray(item.children)) {
        for (const slot of item.children as { id: string }[][]) {
          for (const child of slot) {
            if (child.id === selectedItemId) return child
          }
        }
      }
    }
    return null
  }

  const selectedItem = findSelectedItem()

  if (!selectedItem) {
    return (
      <div className="w-72 border-l bg-muted/30 p-4">
        <div className="h-full flex flex-col items-center justify-center text-muted-foreground">
          <p className="text-sm">选择一个组件以编辑其属性</p>
        </div>
      </div>
    )
  }

  return (
    <div className="w-72 border-l bg-muted/30 p-4 overflow-y-auto">
      <Card>
        <CardHeader className="py-3">
          <CardTitle className="text-sm font-medium">组件属性</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* 基础属性 */}
          <div className="space-y-2">
            <Label htmlFor="name">名称</Label>
            <Input
              id="name"
              value={selectedItem.name}
              onChange={(e) =>
                updateItem(selectedItem.id, { name: e.target.value })
              }
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="tooltip">提示文本</Label>
            <Input
              id="tooltip"
              value={selectedItem.tooltip}
              onChange={(e) =>
                updateItem(selectedItem.id, { tooltip: e.target.value })
              }
            />
          </div>

          {/* 文本输入组件属性 */}
          {selectedItem.type === DesignComponentType.TextInput && (
            <TextInputProperties item={selectedItem} />
          )}

          {/* 选择组件属性 */}
          {(selectedItem.type === DesignComponentType.SingleSelection ||
            selectedItem.type === DesignComponentType.MultiSelection) && (
            <SelectionProperties item={selectedItem} />
          )}

          {/* 表格组件属性 */}
          {selectedItem.type === DesignComponentType.Table && (
            <TableProperties item={selectedItem} />
          )}

          {/* 容器组件属性 */}
          {selectedItem.type === DesignComponentType.Container && (
            <ContainerProperties item={selectedItem} />
          )}
        </CardContent>
      </Card>
    </div>
  )
}

function TextInputProperties({ item }: { item: { id: string; config?: Record<string, unknown> } }) {
  const { updateItem } = useDesignStore()
  const config = item.config || {}

  const updateConfig = (key: string, value: unknown) => {
    updateItem(item.id, { config: { ...config, [key]: value } })
  }

  return (
    <>
      <div className="space-y-2">
        <Label htmlFor="placeholder">占位符</Label>
        <Input
          id="placeholder"
          value={(config.placeholder as string) || ""}
          onChange={(e) => updateConfig("placeholder", e.target.value)}
          placeholder="请输入..."
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="maxLength">最大长度</Label>
        <Input
          id="maxLength"
          type="number"
          value={(config.maxLength as number) || ""}
          onChange={(e) => updateConfig("maxLength", parseInt(e.target.value) || undefined)}
        />
      </div>
      <div className="flex items-center space-x-2">
        <Checkbox
          id="required"
          checked={(config.required as boolean) || false}
          onCheckedChange={(checked) => updateConfig("required", checked)}
        />
        <Label htmlFor="required" className="font-normal">必填</Label>
      </div>
    </>
  )
}

function SelectionProperties({ item }: { item: { id: string; config?: Record<string, unknown> } }) {
  const { updateItem } = useDesignStore()
  const config = item.config || {}
  const options = (config.options as { label: string; value: string }[]) || []
  const [newOption, setNewOption] = useState("")

  const updateConfig = (key: string, value: unknown) => {
    updateItem(item.id, { config: { ...config, [key]: value } })
  }

  const addOption = () => {
    if (newOption.trim()) {
      const newOptions = [...options, { label: newOption, value: newOption }]
      updateConfig("options", newOptions)
      setNewOption("")
    }
  }

  const removeOption = (index: number) => {
    const newOptions = options.filter((_, i) => i !== index)
    updateConfig("options", newOptions)
  }

  return (
    <>
      <div className="space-y-2">
        <Label>选项列表</Label>
        <div className="space-y-2">
          {options.map((opt, index) => (
            <div key={index} className="flex items-center gap-2">
              <Input
                value={opt.label}
                onChange={(e) => {
                  const newOptions = [...options]
                  newOptions[index] = { label: e.target.value, value: e.target.value }
                  updateConfig("options", newOptions)
                }}
                className="flex-1"
              />
              <Button
                variant="ghost"
                size="icon"
                onClick={() => removeOption(index)}
              >
                <Trash2 className="h-4 w-4 text-destructive" />
              </Button>
            </div>
          ))}
          <div className="flex items-center gap-2">
            <Input
              value={newOption}
              onChange={(e) => setNewOption(e.target.value)}
              placeholder="添加选项..."
              onKeyDown={(e) => e.key === "Enter" && addOption()}
            />
            <Button variant="outline" size="icon" onClick={addOption}>
              <Plus className="h-4 w-4" />
            </Button>
          </div>
        </div>
      </div>
      <div className="flex items-center space-x-2">
        <Checkbox
          id="required"
          checked={(config.required as boolean) || false}
          onCheckedChange={(checked) => updateConfig("required", checked)}
        />
        <Label htmlFor="required" className="font-normal">必填</Label>
      </div>
    </>
  )
}

function TableProperties({ item }: { item: { id: string; config?: Record<string, unknown> } }) {
  const { updateItem } = useDesignStore()
  const config = item.config || {}
  const columns = (config.columns as { key: string; title: string }[]) || []
  const [newColumn, setNewColumn] = useState("")

  const updateConfig = (key: string, value: unknown) => {
    updateItem(item.id, { config: { ...config, [key]: value } })
  }

  const addColumn = () => {
    if (newColumn.trim()) {
      const newColumns = [...columns, { key: newColumn.toLowerCase().replace(/\s/g, "_"), title: newColumn }]
      updateConfig("columns", newColumns)
      setNewColumn("")
    }
  }

  const removeColumn = (index: number) => {
    const newColumns = columns.filter((_, i) => i !== index)
    updateConfig("columns", newColumns)
  }

  return (
    <div className="space-y-2">
      <Label>表格列</Label>
      <div className="space-y-2">
        {columns.map((col, index) => (
          <div key={index} className="flex items-center gap-2">
            <Input
              value={col.title}
              onChange={(e) => {
                const newColumns = [...columns]
                newColumns[index] = { ...col, title: e.target.value }
                updateConfig("columns", newColumns)
              }}
              className="flex-1"
            />
            <Button
              variant="ghost"
              size="icon"
              onClick={() => removeColumn(index)}
            >
              <Trash2 className="h-4 w-4 text-destructive" />
            </Button>
          </div>
        ))}
        <div className="flex items-center gap-2">
          <Input
            value={newColumn}
            onChange={(e) => setNewColumn(e.target.value)}
            placeholder="添加列..."
            onKeyDown={(e) => e.key === "Enter" && addColumn()}
          />
          <Button variant="outline" size="icon" onClick={addColumn}>
            <Plus className="h-4 w-4" />
          </Button>
        </div>
      </div>
    </div>
  )
}

function ContainerProperties({ item }: { item: { id: string; config?: Record<string, unknown> } }) {
  const { updateItem } = useDesignStore()
  const config = item.config || {}

  const updateConfig = (key: string, value: unknown) => {
    updateItem(item.id, { config: { ...config, [key]: value } })
  }

  return (
    <div className="space-y-2">
      <Label htmlFor="columns">列数</Label>
      <select
        id="columns"
        className="w-full h-9 rounded-md border border-input bg-transparent px-3 text-sm"
        value={(config.columns as number) || 2}
        onChange={(e) => updateConfig("columns", parseInt(e.target.value))}
      >
        <option value={2}>2 列</option>
        <option value={3}>3 列</option>
        <option value={4}>4 列</option>
      </select>
    </div>
  )
}
