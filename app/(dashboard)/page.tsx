import { auth } from "@/lib/auth"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Users, Calendar, Database, FileText } from "lucide-react"

export default async function HomePage() {
  const session = await auth()

  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-bold">欢迎回来，{session?.user?.name || "用户"}</h1>
        <p className="text-muted-foreground">这是您的医疗研究管理系统概览</p>
      </div>

      {/* 统计卡片 */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4 mb-6">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">患者总数</CardTitle>
            <Users className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">1,234</div>
            <p className="text-xs text-muted-foreground">较上月增长 12%</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">本月随访</CardTitle>
            <Calendar className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">56</div>
            <p className="text-xs text-muted-foreground">待完成 23 例</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">数据集</CardTitle>
            <Database className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">8</div>
            <p className="text-xs text-muted-foreground">活跃数据集</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">表单模板</CardTitle>
            <FileText className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">15</div>
            <p className="text-xs text-muted-foreground">自定义表单</p>
          </CardContent>
        </Card>
      </div>

      {/* 快速操作和最近活动 */}
      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>快速操作</CardTitle>
            <CardDescription>常用功能快捷入口</CardDescription>
          </CardHeader>
          <CardContent className="grid gap-2">
            <div className="flex items-center p-3 rounded-lg bg-muted/50 hover:bg-muted cursor-pointer transition-colors">
              <Users className="h-5 w-5 mr-3 text-primary" />
              <div>
                <p className="font-medium">新增患者</p>
                <p className="text-sm text-muted-foreground">录入新的患者信息</p>
              </div>
            </div>
            <div className="flex items-center p-3 rounded-lg bg-muted/50 hover:bg-muted cursor-pointer transition-colors">
              <Calendar className="h-5 w-5 mr-3 text-primary" />
              <div>
                <p className="font-medium">创建随访</p>
                <p className="text-sm text-muted-foreground">安排新的随访计划</p>
              </div>
            </div>
            <div className="flex items-center p-3 rounded-lg bg-muted/50 hover:bg-muted cursor-pointer transition-colors">
              <FileText className="h-5 w-5 mr-3 text-primary" />
              <div>
                <p className="font-medium">设计表单</p>
                <p className="text-sm text-muted-foreground">创建自定义数据采集表单</p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>最近活动</CardTitle>
            <CardDescription>系统最近的操作记录</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <div className="flex items-start">
                <div className="w-2 h-2 mt-2 rounded-full bg-green-500 mr-3" />
                <div>
                  <p className="text-sm">新增患者 张三</p>
                  <p className="text-xs text-muted-foreground">10 分钟前</p>
                </div>
              </div>
              <div className="flex items-start">
                <div className="w-2 h-2 mt-2 rounded-full bg-blue-500 mr-3" />
                <div>
                  <p className="text-sm">完成随访 李四 - 第3次随访</p>
                  <p className="text-xs text-muted-foreground">1 小时前</p>
                </div>
              </div>
              <div className="flex items-start">
                <div className="w-2 h-2 mt-2 rounded-full bg-amber-500 mr-3" />
                <div>
                  <p className="text-sm">更新表单模板 基础信息采集表</p>
                  <p className="text-xs text-muted-foreground">2 小时前</p>
                </div>
              </div>
              <div className="flex items-start">
                <div className="w-2 h-2 mt-2 rounded-full bg-primary mr-3" />
                <div>
                  <p className="text-sm">导出数据集 2024年Q1数据</p>
                  <p className="text-xs text-muted-foreground">昨天</p>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
