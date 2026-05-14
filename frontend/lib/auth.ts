export const apiBase = "http://127.0.0.1:8080"

export interface CurrentUser {
  id: string
  username: string
  displayName: string
  roles: string[]
}

let pendingAuthCheck: Promise<CurrentUser> | null = null
let redirectingToLogin = false

export function requireSession() {
  return true
}

export async function currentUser() {
  if (!pendingAuthCheck) {
    pendingAuthCheck = authedJson<CurrentUser>("/api/v1/auth/me").finally(() => {
      pendingAuthCheck = null
    })
  }
  return pendingAuthCheck
}

export async function authedFetch(path: string, init?: RequestInit) {
  const response = await fetch(`${apiBase}${path}`, {
    ...init,
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers || {}),
    },
  })
  if (response.status === 401) {
    redirectToLogin()
    throw new Error("登录已过期")
  }
  return response
}

export async function authedJson<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await authedFetch(path, init)
  if (!response.ok) throw new Error(await response.text())
  return response.json()
}

export async function logout() {
  try {
    await fetch(`${apiBase}/api/v1/auth/logout`, { method: "POST", credentials: "include" })
  } finally {
    window.location.href = "/login"
  }
}

function redirectToLogin() {
  if (redirectingToLogin || window.location.pathname === "/login") return
  redirectingToLogin = true
  window.location.replace(`/login?next=${encodeURIComponent(window.location.pathname)}`)
}
