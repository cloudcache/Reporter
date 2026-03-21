"use client"

import { useState } from "react"
import {
  DndContext,
  DragOverlay,
  closestCenter,
  PointerSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
  type DragStartEvent,
} from "@dnd-kit/core"
import { arrayMove } from "@dnd-kit/sortable"
import {
  TextCursor,
  CircleDot,
  CheckSquare,
  Table,
  Columns2,
  Columns3,
  LayoutGrid,
  Save,
  Eye,
  Undo,
  Redo,
} from "lucide-react"
import { Button } from "@/components/ui/button"
import { ToolPalette } from "@/components/design/tool-palette"
import { DesignCanvas } from "@/components/design/design-canvas"
import { PropertyPanel } from "@/components/design/property-panel"
import { useDesignStore } from "@/hooks/use-design-store"
import { DesignComponentType, type DesignComponentItem } from "@/types"
import { cn } from "@/lib/utils"

const iconMap: Record<string, React.ComponentType<{ className?: string }>> = {
  "text-cursor": TextCursor,
  "circle-dot": CircleDot,
  "check-square": CheckSquare,
  "table": Table,
  "columns-2": Columns2,
  "columns-3": Columns3,
  "layout-grid": LayoutGrid,
}

export default function DesignPage() {
  const [activeItem, setActiveItem] = useState<DesignComponentItem | null>(null)
  const { canvasItems, addToCanvas, addToContainer, moveItem } = useDesignStore()

  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: {
        distance: 8,
      },
    })
  )

  const handleDragStart = (event: DragStartEvent) => {
    const { active } = event
    const data = active.data.current

    if (data?.type === "tool") {
      setActiveItem(data.item)
    } else {
      // 画布内拖拽
      const item = canvasItems.find((i) => i.id === active.id)
      if (item) {
        setActiveItem(item)
      }
    }
  }

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event
    setActiveItem(null)

    if (!over) return

    const activeData = active.data.current
    const overData = over.data.current

    // 从工具面板拖入画布
    if (activeData?.type === "tool") {
      const item = activeData.item as DesignComponentItem

      // 拖入容器插槽
      if (overData?.type === "slot") {
        // 容器只能放在画布上，不能放在插槽里
        if (item.type === DesignComponentType.Container) {
          return
        }
        addToContainer(overData.containerId, item, overData.slotIndex)
        return
      }

      // 拖入画布
      if (over.id === "canvas" || overData?.type === "canvas") {
        addToCanvas(item)
        return
      }
    }

    // 画布内排序
    if (active.id !== over.id && canvasItems.some((i) => i.id === active.id)) {
      const oldIndex = canvasItems.findIndex((i) => i.id === active.id)
      const newIndex = canvasItems.findIndex((i) => i.id === over.id)
      if (oldIndex !== -1 && newIndex !== -1) {
        moveItem(oldIndex, newIndex)
      }
    }
  }

  return (
    <div className="h-[calc(100vh-4rem)] flex flex-col">
      {/* 工具栏 */}
      <div className="flex items-center justify-between px-6 py-3 border-b bg-card">
        <div>
          <h1 className="text-lg font-semibold">表单设计器</h1>
          <p className="text-sm text-muted-foreground">拖拽组件创建数据采集表单</p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" disabled>
            <Undo className="h-4 w-4 mr-1" />
            撤销
          </Button>
          <Button variant="outline" size="sm" disabled>
            <Redo className="h-4 w-4 mr-1" />
            重做
          </Button>
          <div className="w-px h-6 bg-border mx-2" />
          <Button variant="outline" size="sm">
            <Eye className="h-4 w-4 mr-1" />
            预览
          </Button>
          <Button size="sm">
            <Save className="h-4 w-4 mr-1" />
            保存
          </Button>
        </div>
      </div>

      {/* 设计器主体 */}
      <div className="flex-1 flex overflow-hidden">
        <DndContext
          sensors={sensors}
          collisionDetection={closestCenter}
          onDragStart={handleDragStart}
          onDragEnd={handleDragEnd}
        >
          {/* 左侧工具面板 */}
          <ToolPalette />

          {/* 中间画布 */}
          <DesignCanvas />

          {/* 右侧属性面板 */}
          <PropertyPanel />

          {/* 拖拽预览 */}
          <DragOverlay>
            {activeItem && (
              <div className="flex items-center gap-2 p-3 border rounded-lg bg-card shadow-lg">
                {activeItem.icon && iconMap[activeItem.icon] && (
                  (() => {
                    const Icon = iconMap[activeItem.icon]
                    return <Icon className="h-4 w-4 text-primary" />
                  })()
                )}
                <span className="text-sm font-medium">{activeItem.name}</span>
              </div>
            )}
          </DragOverlay>
        </DndContext>
      </div>
    </div>
  )
}
