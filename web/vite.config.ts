import { defineConfig } from 'vite'
import { resolve } from 'path'

export default defineConfig({
  define: {
    'process.env.NODE_ENV': JSON.stringify('production'),
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    lib: {
      entry: {
        app: resolve(__dirname, 'client/app.ts'),
        scalar: resolve(__dirname, 'scalar/app.ts'),
      },
      formats: ['es'],
      fileName: (_, entryName) => `${entryName}.js`,
    },
    cssCodeSplit: true,
    sourcemap: true,
    minify: true,
  },
  resolve: {
    alias: {
      '@core': resolve(__dirname, 'client/core'),
      '@design': resolve(__dirname, 'client/design'),
      '@components': resolve(__dirname, 'client/components'),
    },
  },
})
