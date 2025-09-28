import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import { resolve } from "path";
import AutoImport from "unplugin-auto-import/vite";
import Components from "unplugin-vue-components/vite";
import { ElementPlusResolver } from "unplugin-vue-components/resolvers";

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => {
  const isProduction = mode === "production";

  return {
    plugins: [
      vue({
        script: {
          defineModel: true,
          propsDestructure: true,
        },
      }),
      AutoImport({
        resolvers: [ElementPlusResolver({ importStyle: false })],
        imports: [
          "vue",
          "vue-router",
          "pinia",
          {
            "element-plus": [
              "ElMessage",
              "ElMessageBox",
              "ElNotification",
              "ElLoading",
            ],
          },
        ],
        dts: true,
        eslintrc: {
          enabled: true,
        },
      }),
      Components({
        resolvers: [ElementPlusResolver({ importStyle: false })],
        dts: true,
      }),
    ],
    resolve: {
      alias: {
        "@": resolve(__dirname, "src"),
      },
    },
    css: {
      preprocessorOptions: {
        scss: {
          additionalData: '@use "@/styles/variables.scss" as *;',
          // Silence deprecation warnings
          quietDeps: true,
        },
      },
    },
    server: {
      host: "0.0.0.0",
      port: 5173,
      open: false,
      proxy: {
        "/api": {
          target: "http://localhost:8080",
          changeOrigin: true,
          secure: false,
        },
        "/ws": {
          target: "ws://localhost:8080",
          ws: true,
          changeOrigin: true,
        },
      },
    },
    build: {
      target: "es2020",
      outDir: "dist",
      assetsDir: "assets",
      sourcemap: false,
      minify: "esbuild", // Faster minification
      reportCompressedSize: false,
      chunkSizeWarningLimit: 600,
      rollupOptions: {
        output: {
          manualChunks: {
            vue: ['vue'],
            'vue-router': ['vue-router'],
            pinia: ['pinia'],
            'element-plus': ['element-plus'],
          },
          chunkFileNames: "js/[name]-[hash].js",
          entryFileNames: "js/[name]-[hash].js",
          assetFileNames: "assets/[name]-[hash][extname]",
        },
      },
    },
    optimizeDeps: {
      include: [
        "vue",
        "vue-router",
        "pinia",
        "element-plus",
        "@element-plus/icons-vue",
        "axios",
        "dayjs",
        "nprogress",
        "js-cookie",
        "lodash-es",
      ],
      // Exclude heavy dependencies that should be loaded on demand
      exclude: ["@vueuse/core"],
    },
    // Performance optimizations
    esbuild: {
      // Remove console.log in production
      drop: isProduction ? ["console", "debugger"] : [],
    },
    // Removed experimental features to fix runtime errors
  };
});
