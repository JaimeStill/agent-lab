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
        shared: resolve(__dirname, 'src/entries/shared.ts'),
        docs: resolve(__dirname, 'src/entries/docs.ts'),
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
      '@core': resolve(__dirname, 'src/core'),
      '@design': resolve(__dirname, 'src/design'),
      '@components': resolve(__dirname, 'src/components'),
    },
  },
})
