import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";

import * as Types from "./src/types";

export default defineConfig({
  server: {
    port: 4000,
    host: true,
  },
  base: "/",
  build: {
    chunkSizeWarningLimit: 1000,
    reportCompressedSize: false,
  },
  plugins: [vue()],
  define: {
    CRunStatus: JSON.stringify(Types.CRunStatus),
    CNodeStatus: JSON.stringify(Types.CNodeStatus),
    CCollectionName: JSON.stringify(Types.CCollectionName),
    CTerminalDefaults: JSON.stringify(Types.CTerminalDefaults),
  },
  resolve: {
    alias: {
      "@": __dirname + "/src",
    },
  },
});
