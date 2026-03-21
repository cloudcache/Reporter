import { create } from "zustand"
import { DesignComponentType, type DesignComponentItem } from "@/types"

// 工具面板组件
const toolComponents: DesignComponentItem[] = [
  {
    id: "tool-text-input",
    type: DesignComponentType.TextInput,
    name: "文本输入",
    tooltip: "单行文本输入框",
    icon: "text-cursor",
  },
  {
    id: "tool-single-select",
    type: DesignComponentType.SingleSelection,
    name: "单选",
    tooltip: "单项选择",
    icon: "circle-dot",
  },
  {
    id: "tool-multi-select",
    type: DesignComponentType.MultiSelection,
    name: "多选",
    tooltip: "多项选择",
    icon: "check-square",
  },
  {
    id: "tool-table",
    type: DesignComponentType.Table,
    name: "表格",
    tooltip: "数据表格",
    icon: "table",
  },
]

// 布局容器
const layoutComponents: DesignComponentItem[] = [
  {
    id: "layout-2-col",
    type: DesignComponentType.Container,
    name: "两列布局",
    tooltip: "两列容器布局",
    icon: "columns-2",
    config: { columns: 2 },
  },
  {
    id: "layout-3-col",
    type: DesignComponentType.Container,
    name: "三列布局",
    tooltip: "三列容器布局",
    icon: "columns-3",
    config: { columns: 3 },
  },
  {
    id: "layout-4-col",
    type: DesignComponentType.Container,
    name: "四列布局",
    tooltip: "四列容器布局",
    icon: "layout-grid",
    config: { columns: 4 },
  },
]

interface DesignStore {
  // 工具面板
  tools: DesignComponentItem[]
  layouts: DesignComponentItem[]
  
  // 画布状态
  canvasItems: DesignComponentItem[]
  selectedItemId: string | null
  
  // 拖拽状态
  draggedItem: DesignComponentItem | null
  
  // 操作方法
  setDraggedItem: (item: DesignComponentItem | null) => void
  addToCanvas: (item: DesignComponentItem, index?: number) => void
  removeFromCanvas: (id: string) => void
  updateItem: (id: string, data: Partial<DesignComponentItem>) => void
  selectItem: (id: string | null) => void
  moveItem: (fromIndex: number, toIndex: number) => void
  clearCanvas: () => void
  
  // 容器内操作
  addToContainer: (containerId: string, item: DesignComponentItem, slotIndex: number) => void
  removeFromContainer: (containerId: string, itemId: string) => void
}

// 生成唯一 ID
const generateId = () => `item-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`

export const useDesignStore = create<DesignStore>((set, get) => ({
  tools: toolComponents,
  layouts: layoutComponents,
  canvasItems: [],
  selectedItemId: null,
  draggedItem: null,

  setDraggedItem: (item) => set({ draggedItem: item }),

  addToCanvas: (item, index) => {
    const newItem: DesignComponentItem = {
      ...item,
      id: generateId(),
      children: item.type === DesignComponentType.Container 
        ? Array(item.config?.columns || 2).fill(null).map(() => []) 
        : undefined,
    }
    
    set((state) => {
      const items = [...state.canvasItems]
      if (index !== undefined) {
        items.splice(index, 0, newItem)
      } else {
        items.push(newItem)
      }
      return { canvasItems: items, selectedItemId: newItem.id }
    })
  },

  removeFromCanvas: (id) => {
    set((state) => ({
      canvasItems: state.canvasItems.filter((item) => item.id !== id),
      selectedItemId: state.selectedItemId === id ? null : state.selectedItemId,
    }))
  },

  updateItem: (id, data) => {
    set((state) => ({
      canvasItems: state.canvasItems.map((item) =>
        item.id === id ? { ...item, ...data } : item
      ),
    }))
  },

  selectItem: (id) => set({ selectedItemId: id }),

  moveItem: (fromIndex, toIndex) => {
    set((state) => {
      const items = [...state.canvasItems]
      const [removed] = items.splice(fromIndex, 1)
      items.splice(toIndex, 0, removed)
      return { canvasItems: items }
    })
  },

  clearCanvas: () => set({ canvasItems: [], selectedItemId: null }),

  addToContainer: (containerId, item, slotIndex) => {
    const newItem: DesignComponentItem = {
      ...item,
      id: generateId(),
    }
    
    set((state) => ({
      canvasItems: state.canvasItems.map((canvasItem) => {
        if (canvasItem.id === containerId && canvasItem.children) {
          const newChildren = [...canvasItem.children] as DesignComponentItem[][]
          if (!newChildren[slotIndex]) {
            newChildren[slotIndex] = []
          }
          newChildren[slotIndex] = [...newChildren[slotIndex], newItem]
          return { ...canvasItem, children: newChildren }
        }
        return canvasItem
      }),
      selectedItemId: newItem.id,
    }))
  },

  removeFromContainer: (containerId, itemId) => {
    set((state) => ({
      canvasItems: state.canvasItems.map((canvasItem) => {
        if (canvasItem.id === containerId && canvasItem.children) {
          const newChildren = (canvasItem.children as DesignComponentItem[][]).map(
            (slot) => slot.filter((item) => item.id !== itemId)
          )
          return { ...canvasItem, children: newChildren }
        }
        return canvasItem
      }),
      selectedItemId: state.selectedItemId === itemId ? null : state.selectedItemId,
    }))
  },
}))
