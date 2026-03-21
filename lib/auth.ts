import NextAuth from "next-auth"
import Credentials from "next-auth/providers/credentials"
import { z } from "zod"

const loginSchema = z.object({
  username: z.string().min(1, "用户名不能为空"),
  password: z.string().min(1, "密码不能为空"),
})

export const { handlers, signIn, signOut, auth } = NextAuth({
  pages: {
    signIn: "/login",
  },
  providers: [
    Credentials({
      name: "credentials",
      credentials: {
        username: { label: "用户名", type: "text" },
        password: { label: "密码", type: "password" },
      },
      async authorize(credentials) {
        const parsed = loginSchema.safeParse(credentials)
        
        if (!parsed.success) {
          return null
        }

        const { username, password } = parsed.data

        // 模拟用户验证 - 实际项目中应该查询数据库
        // TODO: 替换为真实的数据库验证
        if (username === "admin" && password === "admin123") {
          return {
            id: "1",
            name: "管理员",
            email: "admin@example.com",
            role: "admin",
          }
        }

        if (username === "user" && password === "user123") {
          return {
            id: "2",
            name: "普通用户",
            email: "user@example.com",
            role: "user",
          }
        }

        return null
      },
    }),
  ],
  callbacks: {
    async jwt({ token, user }) {
      if (user) {
        token.id = user.id
        token.role = user.role
      }
      return token
    },
    async session({ session, token }) {
      if (session.user) {
        session.user.id = token.id as string
        session.user.role = token.role as string
      }
      return session
    },
  },
  session: {
    strategy: "jwt",
  },
})
