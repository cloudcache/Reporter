export const apiBase = "http://127.0.0.1:8080"
export const accessTokenKey = "reporter.accessToken"

const refreshLockKey = "reporter.refresh.lockUntil"
const lastRefreshKey = "reporter.refresh.lastAt"
const refreshDebounceMs = 10_000
const refreshLockMs = 8_000

export interface CurrentUser {
  id: string
  username: string
  displayName: string
  roles: string[]
}

let pendingAuthCheck: Promise<CurrentUser> | null = null
let redirectingToLogin = false
let pendingRefresh: Promise<string | null> | null = null

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
  const response = await sendAuthed(path, init)
  if (response.status !== 401) return response

  const nextToken = await refreshAccessToken()
  if (nextToken) {
    const retry = await sendAuthed(path, init, nextToken)
    if (retry.status !== 401) return retry
  }

  redirectToLogin()
  throw new Error("登录已过期")
}

async function sendAuthed(path: string, init?: RequestInit, overrideToken?: string) {
  const token = overrideToken || localStorage.getItem(accessTokenKey)
  return fetch(`${apiBase}${path}`, {
    ...init,
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...(init?.headers || {}),
    },
  })
}

export async function authedJson<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await authedFetch(path, init)
  if (!response.ok) throw new Error(await response.text())
  return response.json()
}

export function publicApiUrl(path: string) {
  return `${apiBase}${path}`
}

export async function publicFetch(path: string, init?: RequestInit) {
  return fetch(publicApiUrl(path), {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers || {}),
    },
  })
}

export async function publicJson<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await publicFetch(path, init)
  if (!response.ok) throw new Error(await response.text())
  return response.json()
}

export async function logout() {
  try {
    await fetch(`${apiBase}/api/v1/auth/logout`, { method: "POST", credentials: "include" })
  } finally {
    localStorage.removeItem(accessTokenKey)
    window.location.href = "/login"
  }
}

async function refreshAccessToken() {
  if (pendingRefresh) return pendingRefresh
  pendingRefresh = doRefreshAccessToken().finally(() => {
    pendingRefresh = null
  })
  return pendingRefresh
}

async function doRefreshAccessToken() {
  const now = Date.now()
  const lastRefreshAt = Number(localStorage.getItem(lastRefreshKey) || 0)
  const lockUntil = Number(localStorage.getItem(refreshLockKey) || 0)
  if (now - lastRefreshAt < refreshDebounceMs || now < lockUntil) {
    return localStorage.getItem(accessTokenKey)
  }
  localStorage.setItem(refreshLockKey, String(now + refreshLockMs))
  try {
    const response = await fetch(`${apiBase}/api/v1/auth/refresh`, {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
    })
    if (!response.ok) return null
    const data = await response.json()
    localStorage.setItem(lastRefreshKey, String(Date.now()))
    if (data.accessToken) {
      localStorage.setItem(accessTokenKey, data.accessToken)
      return data.accessToken as string
    }
    return localStorage.getItem(accessTokenKey)
  } finally {
    localStorage.removeItem(refreshLockKey)
  }
}

function redirectToLogin() {
  if (redirectingToLogin || window.location.pathname === "/login") return
  redirectingToLogin = true
  window.location.replace(`/login?next=${encodeURIComponent(window.location.pathname)}`)
}
