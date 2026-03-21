// 菜单项类型
export interface MenuItem {
  menuName: string
  iconName: string
  tooltip?: string
  path: string
}

// 用户类型
export interface User {
  id: string
  username: string
  realName: string
  email?: string
  roles: Role[]
}

export interface Role {
  roleId: string
  roleName: string
}

// 课题/研究类型
export interface Research {
  researchId: string
  researchName: string
}

// 用户令牌类型
export interface UserToken {
  userToken: string
  currentResearch: Research
}

// 设计组件类型
export enum DesignComponentType {
  Container = -1,
  SingleSelection = 1,
  MultiSelection = 2,
  TextInput = 3,
  Table = 4,
}

export interface DesignComponentItem {
  id: string
  type: DesignComponentType
  name: string
  tooltip: string
  icon?: string
  config?: Record<string, unknown>
  children?: DesignComponentItem[]
}

// 列容器配置
export interface ContainerConfig {
  columns: number
}

// 文本输入配置
export interface TextInputConfig {
  placeholder?: string
  maxLength?: number
  required?: boolean
}

// 选择项配置
export interface SelectionConfig {
  options: { label: string; value: string }[]
  required?: boolean
}

// 表格配置
export interface TableConfig {
  columns: { key: string; title: string; type: DesignComponentType }[]
}
