# Session 05a: Lit Migration

## Objective

Adapt existing web infrastructure for Lit-based SPA. Establish client-side routing, minimal CSS foundation, and baseline views.

## Changes Overview

### 1. Add Lit Dependencies

**File:** `web/package.json`

Add to dependencies:
- `lit`: `^3.3.2`
- `@lit/context`: `^1.1.6`
- `@lit-labs/signals`: `^0.2.0`

### 2. Update TypeScript Configuration

**File:** `web/tsconfig.json`

Add decorator support:
```json
"experimentalDecorators": true,
"useDefineForClassFields": false
```

Add path alias for router:
```json
"@app/router/*": ["./app/client/router/*"]
```

### 3. Update Vite Client Config

**File:** `web/app/client.config.ts`

Add router alias:
```typescript
'@app/router': resolve(root, 'client/router'),
```

### 4. Create Router

**New files in `web/app/client/router/`:**

| File | Purpose |
|------|---------|
| `types.ts` | RouteConfig, RouteMatch interfaces |
| `routes.ts` | Route definitions (home, not-found, wildcard) |
| `router.ts` | Router class + navigate() export |
| `index.ts` | Re-exports |

Adapted from go-lit patterns with `al-` prefix for components.

### 5. Create Baseline Views

**New files in `web/app/client/views/`:**

| File | Purpose |
|------|---------|
| `home-view.ts` | Landing page |
| `home-view.css` | Minimal styles |
| `not-found-view.ts` | 404 fallback |
| `not-found-view.css` | Minimal styles |

Simple Lit components extending `LitElement` with `@customElement` decorator.

### 6. Update App Entry Point

**File:** `web/app/client/app.ts`

```typescript
import './design/index.css';
import { Router } from '@app/router';
import './views/home-view';
import './views/not-found-view';

const router = new Router('app-content');
router.start();
```

### 7. Reorganize CSS Infrastructure

Align with go-lit file organization: `core/` for foundational infrastructure, `app/` for application-specific styles.

**New structure:**
```
design/
├── index.css           # Layer declaration + imports
├── core/
│   ├── tokens.css      # All design tokens (fonts, spacing, typography, colors)
│   ├── reset.css       # Global reset (moved from design/)
│   ├── theme.css       # Theme application (body font/bg, pre/code)
│   └── layout.css      # Empty placeholder (utilities added as needed)
└── app/
    ├── app.css         # App shell styles (body flex, nav, #app-content)
    └── elements.css    # Empty placeholder (shadow DOM base styles when needed)
```

**File contents:**

`index.css`:
```css
@layer tokens, reset, theme, layout;

@import url(./core/tokens.css);
@import url(./core/reset.css);
@import url(./core/theme.css);
@import url(./core/layout.css);

@import url(./app/app.css);
```

`core/tokens.css` - Consolidate all tokens:
- Font families (from current theme.css)
- Spacing scale (from current layout.css)
- Typography scale (from current layout.css)
- Color tokens with dark/light mode (from current theme.css)

`core/reset.css` - Move existing reset.css

`core/theme.css` - Simplified to just application:
```css
@layer theme {
  body {
    font-family: var(--font-sans);
    background-color: var(--bg);
    color: var(--color);
  }

  pre, code {
    font-family: var(--font-mono);
  }
}
```

`core/layout.css` - Empty layer placeholder:
```css
@layer layout {
  /* Layout utilities added as needed */
}
```

`app/app.css` - App shell layout + nav:
```css
body {
  display: flex;
  flex-direction: column;
  height: 100dvh;
  margin: 0;
  overflow: hidden;
}

.app-header { /* nav styles */ }

#app-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}

#app-content > * {
  flex: 1;
  min-height: 0;
}
```

`app/elements.css` - Empty placeholder with comment:
```css
/*
 * Base styles for Shadow DOM components.
 * Import in component CSS: @import '@app/design/app/elements.css';
 */
```

**Files to delete:**
- `design/components.css` - Unused opinionated styles
- `design/styles.css` - Replaced by index.css
- `design/layout.css` - Tokens moved to core/tokens.css
- `design/theme.css` - Split into core/tokens.css and core/theme.css
- `design/reset.css` - Moved to core/reset.css

### 8. Convert Go to Single Shell

**File:** `web/app/app.go`

Change from multiple view routes to single catch-all:
```go
var views = []web.ViewDef{
    {Route: "/{path...}", Template: "shell.html", Title: "agent-lab", Bundle: "app"},
}
```

**File:** `web/app/server/views/shell.html`

Create empty content block (router handles view mounting):
```html
{{ define "content" }}{{ end }}
```

**Delete:** `web/app/server/views/home.html`, `components.html`

**Keep:** `404.html` for server-level errors (not client routing)

### 9. Update Shell Template

**File:** `web/app/server/layouts/app.html`

- Change `<nav class="app-nav">` to `<header class="app-header">`
- Change `<main class="app-content">` to `<main id="app-content">`
- Update nav links to match client routes
- Remove "Components" link (was for SSR style guide)

## Files Summary

### New Files
- `web/app/client/router/types.ts`
- `web/app/client/router/routes.ts`
- `web/app/client/router/router.ts`
- `web/app/client/router/index.ts`
- `web/app/client/views/home-view.ts`
- `web/app/client/views/home-view.css`
- `web/app/client/views/not-found-view.ts`
- `web/app/client/views/not-found-view.css`
- `web/app/client/design/index.css`
- `web/app/client/design/core/tokens.css`
- `web/app/client/design/core/reset.css`
- `web/app/client/design/core/theme.css`
- `web/app/client/design/core/layout.css`
- `web/app/client/design/app/app.css`
- `web/app/client/design/app/elements.css`
- `web/app/server/views/shell.html`

### Modified Files
- `web/package.json`
- `web/tsconfig.json`
- `web/app/client.config.ts`
- `web/app/client/app.ts`
- `web/app/app.go`
- `web/app/server/layouts/app.html`

### Deleted Files
- `web/app/client/design/components.css`
- `web/app/client/design/styles.css`
- `web/app/client/design/layout.css`
- `web/app/client/design/theme.css`
- `web/app/client/design/reset.css`
- `web/app/server/views/home.html`
- `web/app/server/views/components.html`

## Verification

1. `cd web && bun install` - Install Lit dependencies
2. `bun run build` - Build succeeds without errors
3. `go run ./cmd/server` - Server starts
4. Navigate to `/app` - Home view renders
5. Navigate to `/app/invalid` - Not found view renders
6. Browser back/forward - Router handles history
7. Click nav links - Client-side navigation works
