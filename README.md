# RJCFU

a data-driven design sample

## Frontend API endpoint

The Astro frontend reads the API endpoint from `PUBLIC_API_BASE` at build/dev time. When it is empty or omitted, the built pages call same-origin `/api`, which is the recommended Caddy file-server deployment mode.

```bash
# Production: serve dist and reverse-proxy /api on the same domain.
PUBLIC_API_BASE= npm run build

# Local development: point Astro at the local API explicitly.
PUBLIC_API_BASE=http://127.0.0.1:8080 npm run dev
```

Do not rely on a hidden localhost fallback in production builds; there is none.

![](screenshot/screenshot.gif)
