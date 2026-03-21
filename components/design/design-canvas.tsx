"use client"

import { useDroppable } from "@dnd-kit/core"
import {
  SortableContext,
  verticalListSortingStrategy,
  useSortable,
} from "@dnd-kit/sortable"
import { CSS } from "@dnd-kit/utilities"
import {
  TextCursor,
  CircleDot,
  CheckSquare,
  Table,
  Columns2,
  Columns3,
  LayoutGrid,
  Trash2,
  GripVertical,
} from "lucide-react"
import { Button } from "@/components/ui/button"
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

interface CanvasItemProps {
  item: DesignComponentItem
  onRemove: () => void
}

function CanvasItem({ item, onRemove }: CanvasItemProps) {
  const { selectedItemId, selectItem } = useDesignStore()
  const isSelected = selectedItemId === item.id
  const Icon = item.icon ? iconMap[item.icon] : null

  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: item.id })

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
  }

  // 渲染容器布局
  if (item.type === DesignComponentType.Container) {
    const columns = (item.config?.columns as number) || 2
    const children = (item.children as DesignComponentItem[][]) || []

    return (
      <div
        ref={setNodeRef}
        style={style}
        className={cn(
          "border-2 border-dashed rounded-lg p-4 transition-colors",
          isSelected ? "border-primary bg-primary/5" : "border-muted-foreground/30",
          isDragging && "opacity-50"
        )}
        onClick={() => selectItem(item.id)}
      >
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-2">
            <button {...attributes} {...listeners} className="cursor-grab touch-none">
              <GripVertical className="h-4 w-4 text-muted-foreground" />
            </button>
            {Icon && <Icon className="h-4 w-4 text-muted-foreground" />}
            <span className="text-sm font-medium">{item.name}</span>
          </div>
          <Button variant="ghost" size="icon" onClick={onRemove}>
            <Trash2 className="h-4 w-4 text-destructive" />
          </Button>
        </div>
        <div
          className="grid gap-4"
          style={{ gridTemplateColumns: `repeat(${columns}, 1fr)` }}
        >
          {Array.from({ length: columns }).map((_, index) => (
            <ContainerSlot
              key={index}
              containerId={item.id}
              slotIndex={index}
              items={children[index] || []}
            />
          ))}
        </div>
      </div>
    )
  }

  // 渲染普通组件
  return (
    <div
      ref={setNodeRef}
      style={style}
      className={cn(
        "flex items-center justify-between p-4 border rounded-lg bg-card transition-colors",
        isSelected ? "border-primary ring-2 ring-primary/20" : "border-border",
        isDragging && "opacity-50"
      )}
      onClick={() => selectItem(item.id)}
    >
      <div className="flex items-center gap-3">
        <button {...attributes} {...listeners} className="cursor-grab touch-none">
          <GripVertical className="h-4 w-4 text-muted-foreground" />
        </button>
        {Icon && <Icon className="h-5 w-5 text-primary" />}
        <div>
          <p className="font-medium">{item.name}</p>
          <p className="text-xs text-muted-foreground">{item.tooltip}</p>
        </div>
      </div>
      <Button variant="ghost" size="icon" onClick={onRemove}>
        <Trash2 className="h-4 w-4 text-destructive" />
      </Button>
    </div>
  )
}

interface ContainerSlotProps {
  containerId: string
  slotIndex: number
  items: DesignComponentItem[]
}

function ContainerSlot({ containerId, slotIndex, items }: ContainerSlotProps) {
  const { removeFromContainer, selectItem, selectedItemId } = useDesignStore()
  const { setNodeRef, isOver } = useDroppable({
    id: `${containerId}-slot-${slotIndex}`,
    data: { type: "slot", containerId, slotIndex },
  })

  return (
    <div
      ref={setNodeRef}
      className={cn(
        "min-h-24 border-2 border-dashed rounded-lg p-2 transition-colors",
        isOver ? "border-primary bg-primary/10" : "border-muted-foreground/20",
        items.length === 0 && "flex items-center justify-center"
      )}
    >
      {items.length === 0 ? (
        <span className="text-xs text-muted-foreground">拖放组件到此处</span>
      ) : (
        <div className="space-y-2">
          {items.map((item) => {
            const Icon = item.icon ? iconMap[item.icon] : null
            const isSelected = selectedItemId === item.id
            return (
              <div
                key={item.id}
                className={cn(
                  "flex items-center justify-between p-2 border rounded bg-background",
                  isSelected && "border-primary ring-1 ring-primary/20"
                )}
                onClick={() => selectItem(item.id)}
              >
                <div className="flex items-center gap-2">
                  {Icon && <Icon className="h-4 w-4 text-primary" />}
                  <span className="text-sm">{item.name}</span>
                </div>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-6 w-6"
                  onClick={(e) => {
                    e.stopPropagation()
                    removeFromContainer(containerId, item.id)
                  }}
                >
                  <Trash2 className="h-3 w-3 text-destructive" />
                </Button>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}

export function DesignCanvas() {
  const { canvasItems, removeFromCanvas, clearCanvas } = useDesignStore()
  const { setNodeRef, isOver } = useDroppable({
    id: "canvas",
    data: { type: "canvas" },
  })

  return (
    <div className="flex-1 flex flex-col overflow-hidden">
      <div className="flex items-center justify-between p-4 border-b">
        <h2 className="font-semibold">设计画布</h2>
        {canvasItems.length > 0 && (
          <Button variant="outline" size="sm" onClick={clearCanvas}>
            清空画布
          </Button>
        )}
      </div>

      <div
        ref={setNodeRef}
        className={cn(
          "flex-1 p-6 overflow-y-auto",
          isOver && "bg-primary/5"
        )}
      >
        {canvasItems.length === 0 ? (
          <div className="h-full flex flex-col items-center justify-center text-muted-foreground border-2 border-dashed rounded-lg">
            <LayoutGrid className="h-12 w-12 mb-4 opacity-50" />
            <p>将组件从左侧工具面板拖放到此处</p>
            <p className="text-sm">或先添加布局容器来组织表单</p>
          </div>
        ) : (
          <SortableContext
            items={canvasItems.map((i) => i.id)}
            strategy={verticalListSortingStrategy}
          >
            <div className="space-y-4">
              {canvasItems.map((item) => (
                <CanvasItem
                  key={item.id}
                  item={item}
                  onRemove={() => removeFromCanvas(item.id)}
                />
              ))}
            </div>
          </SortableContext>
        )}
      </div>
    </div>
  )
}
