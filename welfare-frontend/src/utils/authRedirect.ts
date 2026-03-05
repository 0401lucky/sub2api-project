const AUTH_REDIRECT_WHITELIST = new Set(['/', '/admin'])

function normalizePathname(pathname: string): string {
  if (pathname === '/') {
    return pathname
  }
  return pathname.replace(/\/+$/, '')
}

export function sanitizeAuthRedirect(input: unknown): string {
  if (typeof input !== 'string') {
    return '/'
  }

  const raw = input.trim()
  if (!raw) {
    return '/'
  }

  try {
    const url = new URL(raw, window.location.origin)
    if (url.origin !== window.location.origin) {
      return '/'
    }

    const pathname = normalizePathname(url.pathname)
    if (!AUTH_REDIRECT_WHITELIST.has(pathname)) {
      return '/'
    }

    return `${pathname}${url.search}${url.hash}`
  } catch {
    return '/'
  }
}
