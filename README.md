# RJCFU

a data-driven design sample

## Frontend API endpoint

The Astro frontend reads the API endpoint from `PUBLIC_API_BASE` at build/dev time.

```bash
PUBLIC_API_BASE=http://127.0.0.1:8080 npm run dev
PUBLIC_API_BASE=https://api.example.com npm run build
```

If omitted, it defaults to `http://127.0.0.1:8080`.

![](screenshot/screenshot.gif)
