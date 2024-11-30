import { defineConfig, loadEnv } from "vite";
import vue from "@vitejs/plugin-vue";

import * as Types from "./src/types";

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "");
  const backendUrl = env.VITE_BACKEND_URL || "http://127.0.0.1:8090";

  return {
    server: {
      port: 4000,
      host: true,
      proxy: {
        "/api": {
          target: backendUrl,
          changeOrigin: true,
        },
      },
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
  };
});
