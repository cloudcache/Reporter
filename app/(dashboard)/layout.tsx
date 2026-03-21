import { Header } from "@/components/layout/header"
import { SessionProvider } from "@/components/providers/session-provider"

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <SessionProvider>
      <div className="min-h-screen flex flex-col">
        <Header />
        <main className="flex-1 bg-background">
          {children}
        </main>
      </div>
    </SessionProvider>
  )
}
