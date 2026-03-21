import type { Metadata, Viewport } from "next"
import { Noto_Sans_SC } from "next/font/google"
import "./globals.css"

const notoSansSC = Noto_Sans_SC({
  subsets: ["latin"],
  variable: "--font-sans",
})

export const metadata: Metadata = {
  title: "Reporter - 医疗研究管理系统",
  description: "医疗研究患者管理和数据采集系统",
}

export const viewport: Viewport = {
  width: "device-width",
  initialScale: 1,
  themeColor: "#1d4ed8",
}

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode
}>) {
  return (
    <html lang="zh-CN">
      <body className={`${notoSansSC.variable} font-sans antialiased`}>
        {children}
      </body>
    </html>
  )
}
