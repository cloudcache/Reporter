import type { MenuItem } from "@/types"

export const menuItems: MenuItem[] = [
  { menuName: "首页", iconName: "home", path: "/" },
  { menuName: "搜索", iconName: "search", path: "/search" },
  { menuName: "患者管理", iconName: "users", path: "/patient" },
  { menuName: "随访管理", iconName: "calendar", path: "/followup" },
  { menuName: "数据集", iconName: "database", path: "/dataset" },
  { menuName: "表单设计", iconName: "layout", path: "/design" },
  { menuName: "系统设置", iconName: "settings", path: "/system" },
  { menuName: "数据导出", iconName: "download", path: "/export" },
]
