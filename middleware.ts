import { auth } from "@/lib/auth"
import { NextResponse } from "next/server"

export default auth((req) => {
  const { nextUrl } = req
  const isLoggedIn = !!req.auth

  // 公开路由
  const publicRoutes = ["/login"]
  const isPublicRoute = publicRoutes.includes(nextUrl.pathname)

  // API 路由和静态资源不做处理
  if (
    nextUrl.pathname.startsWith("/api") ||
    nextUrl.pathname.startsWith("/_next") ||
    nextUrl.pathname.includes(".")
  ) {
    return NextResponse.next()
  }

  // 已登录用户访问登录页，重定向到首页
  if (isLoggedIn && isPublicRoute) {
    return NextResponse.redirect(new URL("/", nextUrl))
  }

  // 未登录用户访问受保护页面，重定向到登录页
  if (!isLoggedIn && !isPublicRoute) {
    const loginUrl = new URL("/login", nextUrl)
    loginUrl.searchParams.set("callbackUrl", nextUrl.pathname)
    return NextResponse.redirect(loginUrl)
  }

  return NextResponse.next()
})

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico).*)"],
}
