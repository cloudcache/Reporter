import { defineConfig } from "astro/config"
import react from "@astrojs/react"

export default defineConfig({
  srcDir: "./frontend",
  integrations: [react()],
  output: "static",
  vite: {
    resolve: {
      alias: {
        "@": new URL("./", import.meta.url).pathname,
      },
    },
  },
})
