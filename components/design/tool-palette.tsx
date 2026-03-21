"use client"

import { useDraggable } from "@dnd-kit/core"
import { CSS } from "@dnd-kit/utilities"
import {
  TextCursor,
  CircleDot,
  CheckSquare,
  Table,
  Columns2,
  Columns3,
  LayoutGrid,
} from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { useDesignStore } from "@/hooks/use-design-store"
import type { DesignComponentItem } from "@/types"
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

interface DraggableToolProps {
  item: DesignComponentItem
}

function DraggableTool({ item }: DraggableToolProps) {
  const { setDraggedItem } = useDesignStore()
  const { attributes, listeners, setNodeRef, transform, isDragging } = useDraggable({
    id: item.id,
    data: { type: "tool", item },
  })

  const style = transform
    ? {
        transform: CSS.Translate.toString(transform),
      }
    : undefined

  const Icon = item.icon ? iconMap[item.icon] : null

  return (
    <div
      ref={setNodeRef}
      style={style}
      {...listeners}
      {...attributes}
      className={cn(
        "flex items-center gap-2 p-3 border rounded-lg bg-card cursor-grab transition-colors hover:border-primary hover:bg-primary/5",
        isDragging && "opacity-50 cursor-grabbing"
      )}
      onMouseDown={() => setDraggedItem(item)}
      onMouseUp={() => setDraggedItem(null)}
    >
      {Icon && <Icon className="h-4 w-4 text-muted-foreground" />}
      <span className="text-sm">{item.name}</span>
    </div>
  )
}

export function ToolPalette() {
  const { tools, layouts } = useDesignStore()

  return (
    <div className="w-64 border-r bg-muted/30 p-4 space-y-4 overflow-y-auto">
      <Card>
        <CardHeader className="py-3">
          <CardTitle className="text-sm font-medium">布局容器</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2 pt-0">
          {layouts.map((layout) => (
            <DraggableTool key={layout.id} item={layout} />
          ))}
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="py-3">
          <CardTitle className="text-sm font-medium">表单组件</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2 pt-0">
          {tools.map((tool) => (
            <DraggableTool key={tool.id} item={tool} />
          ))}
        </CardContent>
      </Card>

      <div className="text-xs text-muted-foreground p-2">
        <p>提示：拖拽组件到右侧画布区域</p>
      </div>
    </div>
  )
}
