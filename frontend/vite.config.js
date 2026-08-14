import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  build: {
    rollupOptions: {
      output: {
        /**
         * Dependensi baseline dipisah dari kode aplikasi. Isinya jarang berubah, sehingga
         * rilis fitur tidak membatalkan cache browser untuk React, router, query client,
         * Axios, dan Zod sekaligus.
         */
        manualChunks: (id) => (id.includes("node_modules") ? "vendor" : undefined),
      },
    },
  },
  server: {
    host: "0.0.0.0",
    port: 5173,
    // Bind mount Docker Desktop di Windows kadang tidak meneruskan event perubahan
    // file. Polling memastikan localhost selalu memakai source terbaru tanpa restart.
    watch: {
      usePolling: true,
      interval: 250,
    },
    // Nginx memakai nama service Docker sebagai Host saat browser E2E mengakses stack
    // secara internal. Host publik localhost tetap diizinkan oleh default Vite.
    allowedHosts: ["nginx"],
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        changeOrigin: true,
      },
      "/health": {
        target: "http://localhost:8080",
        changeOrigin: true,
      },
    },
  },
  test: {
    environment: "jsdom",
    setupFiles: "./src/test/setup.js",
    css: true,
    include: ["src/**/*.{test,spec}.{js,jsx}"],
    // Test komponen absensi dan form pengajuan melakukan banyak interaksi userEvent
    // berurutan; 5 detik bawaan terlalu ketat saat seluruh berkas berjalan paralel.
    testTimeout: 20000,
  },
});
