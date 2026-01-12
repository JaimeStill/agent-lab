# Frontend Architecture Design Guide

## Objective

Migrate from isolated `scalar/` extraction workflow to unified `web/` directory with:

- Bun for package management
- Vite in library mode for JS/CSS bundling
- Go templates for SSR with embedded assets
- Global design token system with route-scoped styles

## Current State

```
project-root/
├── scalar/
│   └── build.sh          # npm install + manual extraction
├── web/
│   └── docs/
│       ├── scalar.js
│       └── scalar.css
```

## Target State

```
project-root/
├── web/
│   ├── package.json
│   ├── vite.config.ts
│   ├── tsconfig.json
│   ├── src/
│   │   ├── styles.css              # Global tokens + base styles
│   │   └── docs/
│   │       ├── main.ts             # Scalar initialization
│   │       └── styles.css          # Docs-specific overrides
│   ├── dist/                       # Build output (gitignored)
│   │   ├── docs.js
│   │   └── docs.css
│   ├── templates/
│   │   └── docs.html
│   └── web.go                      # embed.FS + handlers
```

---

## Step 1: Remove Old Infrastructure

```bash
rm -rf scalar/
rm -rf web/docs/
```

---

## Step 2: Initialize Web Directory

```bash
cd web
bun init -y
```

### web/package.json

```json
{
  "name": "web",
  "private": true,
  "type": "module",
  "scripts": {
    "dev": "vite build --watch",
    "build": "tsc && vite build"
  },
  "devDependencies": {
    "typescript": "^5.4.0",
    "vite": "^5.4.0"
  },
  "dependencies": {
    "@scalar/api-reference": "^1.25.0"
  }
}
```

Install dependencies:

```bash
bun install
```

---

## Step 3: Configure Vite (Library Mode)

### web/vite.config.ts

```typescript
import { defineConfig } from 'vite'
import { resolve } from 'path'

export default defineConfig({
  root: 'src',
  
  build: {
    outDir: '../dist',
    emptyOutDir: true,
    
    // Library mode: no HTML processing
    lib: {
      entry: {
        'docs': resolve(__dirname, 'src/docs/main.ts'),
      },
      formats: ['es'],
      fileName: (_, entryName) => `${entryName}.js`,
    },
    
    rollupOptions: {
      output: {
        assetFileNames: '[name][extname]',
      },
    },
    
    // Inline small assets as data URIs
    assetsInlineLimit: 100000,
    
    // Generate sourcemaps for debugging (optional: disable for prod)
    sourcemap: true,
  },
  
  // Prevent external fetches
  define: {
    'process.env.NODE_ENV': JSON.stringify('production'),
  },
})
```

---

## Step 4: Configure TypeScript

### web/tsconfig.json

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "ESNext",
    "moduleResolution": "bundler",
    "strict": true,
    "noEmit": true,
    "isolatedModules": true,
    "skipLibCheck": true,
    "types": ["vite/client"]
  },
  "include": ["src/**/*.ts"]
}
```

---

## Step 5: Create Global Styles

### web/src/styles.css

```css
:root {
  /* Color tokens */
  --color-primary: #2563eb;
  --color-primary-hover: #1d4ed8;
  --color-surface: #ffffff;
  --color-background: #f8fafc;
  --color-text: #1e293b;
  --color-text-muted: #64748b;
  --color-border: #e2e8f0;
  --color-error: #dc2626;
  --color-success: #16a34a;
  
  /* Spacing scale */
  --space-1: 0.25rem;
  --space-2: 0.5rem;
  --space-3: 0.75rem;
  --space-4: 1rem;
  --space-6: 1.5rem;
  --space-8: 2rem;
  
  /* Typography */
  --font-sans: system-ui, -apple-system, sans-serif;
  --font-mono: 'JetBrains Mono', 'Fira Code', monospace;
  --text-sm: 0.875rem;
  --text-base: 1rem;
  --text-lg: 1.125rem;
  --text-xl: 1.25rem;
  
  /* Borders */
  --radius-sm: 0.25rem;
  --radius-md: 0.375rem;
  --radius-lg: 0.5rem;
}

*,
*::before,
*::after {
  box-sizing: border-box;
}

body {
  margin: 0;
  font-family: var(--font-sans);
  font-size: var(--text-base);
  color: var(--color-text);
  background: var(--color-background);
  line-height: 1.5;
}

code, pre {
  font-family: var(--font-mono);
}
```

---

## Step 6: Create Docs Entry Point (Scalar)

### web/src/docs/styles.css

```css
@import '../styles.css';

/* Scalar-specific overrides if needed */
#scalar-api-reference {
  min-height: 100vh;
}
```

### web/src/docs/main.ts

```typescript
import './styles.css'

declare global {
  interface Window {
    Scalar: {
      createApiReference: (element: HTMLElement, config: object) => void
    }
  }
}

import '@scalar/api-reference'

const container = document.getElementById('scalar-api-reference')
if (container) {
  const specUrl = container.dataset.specUrl || '/api/openapi.json'
  
  // @ts-expect-error Scalar attaches to window
  window.Scalar.createApiReference(container, {
    spec: { url: specUrl },
    theme: 'default',
  })
}
```

Note: Verify Scalar's exact initialization API. The standalone browser bundle may use a different pattern than the npm package. Adjust as needed based on current Scalar documentation.

---

## Step 7: Create Go Template

### web/templates/docs.html

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{ .Title }}</title>
  <link rel="stylesheet" href="/static/docs.css">
</head>
<body>
  <div id="scalar-api-reference" data-spec-url="{{ .SpecURL }}"></div>
  <script type="module" src="/static/docs.js"></script>
</body>
</html>
```

---

## Step 8: Create web.go

### web/web.go

```go
package web

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"
)

//go:embed dist/*
var distFS embed.FS

//go:embed templates/*
var templateFS embed.FS

var templates *template.Template

func init() {
	templates = template.Must(template.ParseFS(templateFS, "templates/*.html", "templates/**/*.html"))
}

// Static returns a handler for serving built JS/CSS assets.
// Mount at "/static/" in your mux.
func Static() http.Handler {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic(err)
	}
	return http.FileServer(http.FS(sub))
}

// DocsData holds template data for the API docs page.
type DocsData struct {
	Title   string
	SpecURL string
}

// DocsHandler returns a handler that renders the Scalar API reference.
func DocsHandler(data DocsData) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := templates.ExecuteTemplate(w, "docs.html", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
```

---

## Step 9: Integrate in cmd/server

### cmd/server/main.go (relevant additions)

```go
package main

import (
	"net/http"
	"strings"

	"yourmodule/web"
)

func main() {
	mux := http.NewServeMux()
	
	// Static assets (JS/CSS bundles)
	mux.Handle("/static/", http.StripPrefix("/static/", web.Static()))
	
	// API documentation
	mux.HandleFunc("/docs", web.DocsHandler(web.DocsData{
		Title:   "API Documentation",
		SpecURL: "/api/openapi.json",
	}))
	
	// API routes
	mux.HandleFunc("/api/openapi.json", serveOpenAPISpec)
	// ... other API handlers
	
	// Apply trailing slash middleware
	handler := trimSlash(mux)
	
	http.ListenAndServe(":8080", handler)
}

// trimSlash redirects paths with trailing slashes to their canonical form.
func trimSlash(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(r.URL.Path) > 1 && r.URL.Path[len(r.URL.Path)-1] == '/' {
			target := strings.TrimSuffix(r.URL.Path, "/")
			if r.URL.RawQuery != "" {
				target += "?" + r.URL.RawQuery
			}
			http.Redirect(w, r, target, http.StatusMovedPermanently)
			return
		}
		next.ServeHTTP(w, r)
	})
}
```

---

## Step 10: Update .gitignore

Add to project root `.gitignore`:

```gitignore
# Web build artifacts
web/dist/
web/node_modules/
```

---

## Step 11: Build Commands

### Development (watch mode)

```bash
cd web && bun run dev
```

Vite rebuilds on source changes. Run your Go server separately.

### Production build

```bash
cd web && bun run build
```

### Full build (for CI or Makefile)

```bash
cd web && bun install && bun run build && cd .. && go build ./cmd/server
```

---

## Adding App Routes (Future Reference)

When building client UI features under `/app/*`, follow this pattern:

### 1. Create source files

```
web/src/app/{route-name}/
├── main.ts       # Entry point
└── styles.css    # Route-scoped styles (imports ../../styles.css)
```

**styles.css pattern:**

```css
@import '../../styles.css';

/* Route-specific styles using global tokens */
.container {
  padding: var(--space-6);
}
```

**main.ts pattern:**

```typescript
import './styles.css'

interface PageConfig {
  // Define expected config from Go template
}

function getConfig(): PageConfig {
  const el = document.getElementById('app')
  return el?.dataset.config ? JSON.parse(el.dataset.config) : {}
}

function init() {
  const config = getConfig()
  // Initialize page behavior
}

document.readyState === 'loading'
  ? document.addEventListener('DOMContentLoaded', init)
  : init()
```

### 2. Add Vite entry point

In `web/vite.config.ts`, add to the entry object:

```typescript
lib: {
  entry: {
    'docs': resolve(__dirname, 'src/docs/main.ts'),
    '{route-name}': resolve(__dirname, 'src/app/{route-name}/main.ts'),
  },
  // ...
},
```

### 3. Create Go template

**web/templates/app/{route-name}.html:**

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{ .Title }}</title>
  <link rel="stylesheet" href="/static/{route-name}.css">
</head>
<body>
  <div id="app" data-config='{{ .ConfigJSON }}'></div>
  <!-- Template content here -->
  <script type="module" src="/static/{route-name}.js"></script>
</body>
</html>
```

### 4. Add handler in web.go

```go
type {RouteName}Data struct {
	Title      string
	ConfigJSON template.JS
	// Additional fields as needed
}

func {RouteName}Handler(data {RouteName}Data) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := templates.ExecuteTemplate(w, "app/{route-name}.html", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
```

### 5. Register in cmd/server

```go
mux.HandleFunc("/app/{route-name}", web.{RouteName}Handler(web.{RouteName}Data{
	Title:      "Page Title",
	ConfigJSON: `{"key":"value"}`,
}))
```

### 6. Build

```bash
cd web && bun run build
```

---

## Air-Gap Verification

Before deploying to air-gapped environment:

```bash
# Build production assets
cd web && bun run build

# Check for external URLs
grep -r "http://" dist/ || echo "No http:// found"
grep -r "https://" dist/ || echo "No https:// found"

# Test with network disabled
# 1. Build Go binary
# 2. Disconnect network
# 3. Run binary and verify all routes function
```

---

## Scalar Notes

The Scalar library initialization may differ from the example. Verify:

1. Check `node_modules/@scalar/api-reference/dist/` for available bundles
2. Review Scalar documentation for standalone browser usage
3. Confirm no external font or analytics fetches occur

If Scalar fetches external resources, you may need to:

- Override CSS to use local fonts
- Configure Scalar options to disable telemetry
- Bundle required fonts into your dist/ output
