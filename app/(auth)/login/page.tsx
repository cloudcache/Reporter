"use client"

import { useState, useEffect } from "react"
import { useRouter, useSearchParams } from "next/navigation"
import { signIn } from "next-auth/react"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { User, Lock, Loader2 } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Checkbox } from "@/components/ui/checkbox"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"

const loginSchema = z.object({
  username: z.string().min(1, "请输入用户名"),
  password: z.string().min(1, "请输入密码"),
  remember: z.boolean().default(true),
})

type LoginFormData = z.infer<typeof loginSchema>

const backgroundColors = [
  "bg-amber-100",
  "bg-orange-200",
  "bg-rose-300",
  "bg-amber-200",
]

export default function LoginPage() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const callbackUrl = searchParams.get("callbackUrl") || "/"
  
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState("")
  const [currentBgIndex, setCurrentBgIndex] = useState(0)

  const {
    register,
    handleSubmit,
    formState: { errors },
    setValue,
    watch,
  } = useForm<LoginFormData>({
    resolver: zodResolver(loginSchema),
    defaultValues: {
      username: "",
      password: "",
      remember: true,
    },
  })

  const rememberValue = watch("remember")

  // 背景轮播效果
  useEffect(() => {
    const interval = setInterval(() => {
      setCurrentBgIndex((prev) => (prev + 1) % backgroundColors.length)
    }, 10000)
    return () => clearInterval(interval)
  }, [])

  const onSubmit = async (data: LoginFormData) => {
    setIsLoading(true)
    setError("")

    try {
      const result = await signIn("credentials", {
        username: data.username,
        password: data.password,
        redirect: false,
      })

      if (result?.error) {
        setError("用户名或密码错误")
      } else {
        router.push(callbackUrl)
        router.refresh()
      }
    } catch {
      setError("登录失败，请稍后重试")
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="min-h-screen relative">
      {/* 背景轮播 */}
      <div className="absolute inset-0 transition-colors duration-1000 ease-in-out">
        {backgroundColors.map((color, index) => (
          <div
            key={color}
            className={`absolute inset-0 transition-opacity duration-1000 ${color} ${
              index === currentBgIndex ? "opacity-100" : "opacity-0"
            }`}
          />
        ))}
      </div>

      {/* 登录表单 */}
      <Card className="fixed right-8 top-24 w-full max-w-sm shadow-lg">
        <CardHeader>
          <CardTitle className="text-xl text-center">登录系统</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            {error && (
              <div className="p-3 text-sm text-destructive bg-destructive/10 rounded-md">
                {error}
              </div>
            )}

            <div className="space-y-2">
              <Label htmlFor="username">用户名</Label>
              <div className="relative">
                <User className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  id="username"
                  placeholder="用户名/电话号码/工号"
                  className="pl-10"
                  {...register("username")}
                />
              </div>
              {errors.username && (
                <p className="text-sm text-destructive">{errors.username.message}</p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="password">密码</Label>
              <div className="relative">
                <Lock className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  id="password"
                  type="password"
                  placeholder="密码"
                  className="pl-10"
                  autoComplete="current-password"
                  {...register("password")}
                />
              </div>
              {errors.password && (
                <p className="text-sm text-destructive">{errors.password.message}</p>
              )}
            </div>

            <div className="flex items-center justify-between">
              <div className="flex items-center space-x-2">
                <Checkbox
                  id="remember"
                  checked={rememberValue}
                  onCheckedChange={(checked) => setValue("remember", checked as boolean)}
                />
                <Label htmlFor="remember" className="text-sm font-normal cursor-pointer">
                  记住登录状态
                </Label>
              </div>
              <Button variant="link" className="px-0 text-sm h-auto" type="button">
                忘记密码
              </Button>
            </div>

            <Button type="submit" className="w-full" disabled={isLoading}>
              {isLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              登录
            </Button>

            <div className="text-center">
              <Button variant="link" className="text-sm" type="button">
                注册账号
              </Button>
            </div>
          </form>

          {/* 测试账号提示 */}
          <div className="mt-6 p-3 bg-muted rounded-md text-sm">
            <p className="font-medium mb-1">测试账号:</p>
            <p className="text-muted-foreground">管理员: admin / admin123</p>
            <p className="text-muted-foreground">普通用户: user / user123</p>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
