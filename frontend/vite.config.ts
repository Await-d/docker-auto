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
        },
      },
    },
    server: {
      host: "0.0.0.0",
      port: 3000,
      open: false,
      proxy: {
        "/api": {
          target: "http://backend:8080",
          changeOrigin: true,
          secure: false,
        },
        "/ws": {
          target: "ws://backend:8080",
          ws: true,
          changeOrigin: true,
        },
      },
    },
    build: {
      target: "es2015",
      outDir: "dist",
      assetsDir: "assets",
      sourcemap: false,
      minify: "esbuild", // Faster minification
      reportCompressedSize: false, // Disable to speed up build
      chunkSizeWarningLimit: 600, // Lower threshold for better performance
      rollupOptions: {
        output: {
          manualChunks: (id) => {
            // Advanced chunking strategy for better performance
            if (id.includes("node_modules")) {
              if (id.includes("vue")) {
                return "vendor-vue";
              }
              if (id.includes("element-plus")) {
                return "vendor-ui";
              }
              if (id.includes("echarts")) {
                return "vendor-charts";
              }
              if (
                id.includes("axios") ||
                id.includes("dayjs") ||
                id.includes("lodash")
              ) {
                return "vendor-utils";
              }
              // Split large node_modules into smaller chunks
              return "vendor-misc";
            }
            // Split application code by feature
            if (id.includes("/views/")) {
              return "views";
            }
            if (id.includes("/components/dashboard/")) {
              return "dashboard";
            }
            if (id.includes("/components/container/")) {
              return "containers";
            }
            if (id.includes("/components/settings/")) {
              return "settings";
            }
          },
          // Optimize asset naming for better caching
          assetFileNames: (assetInfo) => {
            const info = assetInfo.name.split(".");
            const ext = info[info.length - 1];
            if (/png|jpe?g|svg|gif|tiff|bmp|ico/i.test(ext)) {
              return `img/[name]-[hash][extname]`;
            }
            if (/woff2?|eot|ttf|otf/i.test(ext)) {
              return `fonts/[name]-[hash][extname]`;
            }
            return `assets/[name]-[hash][extname]`;
          },
          chunkFileNames: "js/[name]-[hash].js",
          entryFileNames: "js/[name]-[hash].js",
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
        "echarts/core",
        "echarts/charts/LineChart",
        "echarts/charts/BarChart",
        "echarts/charts/PieChart",
        "echarts/components/GridComponent",
        "echarts/components/TooltipComponent",
        "echarts/components/LegendComponent",
        "echarts/renderers/CanvasRenderer",
        "vue-echarts",
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
    // Enable experimental features for better performance
    experimental: {
      renderBuiltUrl(filename, { hostType }) {
        if (hostType === "js") {
          return { js: `window.__staticBase + ${JSON.stringify(filename)}` };
        } else {
          return { relative: true };
        }
      },
    },
  };
});
