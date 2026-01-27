# Session 05a: Lit Migration

## Overview

Adapt existing web infrastructure for Lit-based SPA. Establish client-side routing, minimal CSS foundation, and baseline views.

**Reference**: `_context/milestones/m05-workflow-lab-interface.md`

## Implementation Steps

### Step 1: Add Lit Dependencies

**File:** `web/package.json`

Add to `dependencies`:

```json
"lit": "^3.3.2",
"@lit/context": "^1.1.6",
"@lit-labs/signals": "^0.2.0"
```

Run `cd web && bun install` after editing.

---

### Step 2: Update TypeScript Configuration

**File:** `web/tsconfig.json`

Add to `compilerOptions`:

```json
"experimentalDecorators": true,
"useDefineForClassFields": false
```

Add to `paths`:

```json
"@app/router/*": ["./app/client/router/*"]
```

---

### Step 3: Update Vite Client Config

**File:** `web/app/client.config.ts`

Add router alias to `aliases`:

```typescript
'@app/router': resolve(root, 'client/router'),
```

---

### Step 4: Reorganize CSS Infrastructure

#### 4.1 Create Directory Structure

```bash
mkdir -p web/app/client/design/core
mkdir -p web/app/client/design/app
```

#### 4.2 Create `design/index.css`

**New file:** `web/app/client/design/index.css`

```css
@layer tokens, reset, theme, layout;

@import url(./core/tokens.css);
@import url(./core/reset.css);
@import url(./core/theme.css);
@import url(./core/layout.css);

@import url(./app/app.css);
```

#### 4.3 Create `design/core/tokens.css`

**New file:** `web/app/client/design/core/tokens.css`

```css
@layer tokens {
  :root {
    color-scheme: dark light;

    --font-sans: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
    --font-mono: ui-monospace, "Cascadia Code", "Source Code Pro", Menlo, Consolas, "DejaVu Sans Mono", monospace;

    --space-1: 0.25rem;
    --space-2: 0.5rem;
    --space-3: 0.75rem;
    --space-4: 1rem;
    --space-5: 1.25rem;
    --space-6: 1.5rem;
    --space-8: 2rem;
    --space-10: 2.5rem;
    --space-12: 3rem;
    --space-16: 4rem;

    --text-xs: 0.75rem;
    --text-sm: 0.875rem;
    --text-base: 1rem;
    --text-lg: 1.125rem;
    --text-xl: 1.25rem;
    --text-2xl: 1.5rem;
    --text-3xl: 1.875rem;
    --text-4xl: 2.25rem;
  }

  @media (prefers-color-scheme: dark) {
    :root {
      --bg: hsl(0, 0%, 7%);
      --bg-1: hsl(0, 0%, 12%);
      --bg-2: hsl(0, 0%, 18%);
      --color: hsl(0, 0%, 93%);
      --color-1: hsl(0, 0%, 80%);
      --color-2: hsl(0, 0%, 65%);
      --divider: hsl(0, 0%, 25%);

      --blue: hsl(210, 100%, 70%);
      --blue-bg: hsl(210, 50%, 20%);
      --green: hsl(140, 70%, 55%);
      --green-bg: hsl(140, 40%, 18%);
      --red: hsl(0, 85%, 65%);
      --red-bg: hsl(0, 50%, 20%);
      --yellow: hsl(45, 90%, 60%);
      --yellow-bg: hsl(45, 50%, 18%);
      --orange: hsl(25, 95%, 65%);
      --orange-bg: hsl(25, 50%, 20%);
    }
  }

  @media (prefers-color-scheme: light) {
    :root {
      --bg: hsl(0, 0%, 100%);
      --bg-1: hsl(0, 0%, 96%);
      --bg-2: hsl(0, 0%, 92%);
      --color: hsl(0, 0%, 10%);
      --color-1: hsl(0, 0%, 30%);
      --color-2: hsl(0, 0%, 45%);
      --divider: hsl(0, 0%, 80%);

      --blue: hsl(210, 90%, 45%);
      --blue-bg: hsl(210, 80%, 92%);
      --green: hsl(140, 60%, 35%);
      --green-bg: hsl(140, 50%, 90%);
      --red: hsl(0, 70%, 50%);
      --red-bg: hsl(0, 70%, 93%);
      --yellow: hsl(45, 80%, 40%);
      --yellow-bg: hsl(45, 80%, 88%);
      --orange: hsl(25, 85%, 50%);
      --orange-bg: hsl(25, 75%, 90%);
    }
  }
}
```

#### 4.4 Create `design/core/reset.css`

**New file:** `web/app/client/design/core/reset.css`

```css
@layer reset {
  *,
  *::before,
  *::after {
    box-sizing: border-box;
  }

  * {
    margin: 0;
  }

  body {
    line-height: 1.5;
  }

  img,
  picture,
  video,
  canvas,
  svg {
    display: block;
    max-width: 100%;
  }

  @media (prefers-reduced-motion: no-preference) {
    :has(:target) {
      scroll-behavior: smooth;
    }
  }
}
```

#### 4.5 Create `design/core/theme.css`

**New file:** `web/app/client/design/core/theme.css`

```css
@layer theme {
  body {
    font-family: var(--font-sans);
    background-color: var(--bg);
    color: var(--color);
  }

  pre,
  code {
    font-family: var(--font-mono);
  }
}
```

#### 4.6 Create `design/core/layout.css`

**New file:** `web/app/client/design/core/layout.css`

```css
@layer layout {
}
```

#### 4.7 Create `design/app/app.css`

**New file:** `web/app/client/design/app/app.css`

```css
body {
  display: flex;
  flex-direction: column;
  height: 100dvh;
  margin: 0;
  overflow: hidden;
}

.app-header {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-3) var(--space-6);
  background: var(--bg-1);
  border-bottom: 1px solid var(--divider);
}

.app-header .brand {
  font-size: var(--text-lg);
  font-weight: 600;
  color: var(--color);
  text-decoration: none;
}

.app-header .brand:hover {
  color: var(--blue);
}

.app-header nav {
  display: flex;
  gap: var(--space-4);
}

.app-header nav a {
  color: var(--color-1);
  text-decoration: none;
  font-size: var(--text-sm);
}

.app-header nav a:hover {
  color: var(--blue);
}

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

#### 4.8 Create `design/app/elements.css`

**New file:** `web/app/client/design/app/elements.css`

```css
/*
 * Base styles for Shadow DOM components.
 * Import in component CSS: @import '@app/design/app/elements.css';
 */
```

#### 4.9 Delete Old CSS Files

```bash
rm web/app/client/design/components.css
rm web/app/client/design/styles.css
rm web/app/client/design/layout.css
rm web/app/client/design/theme.css
rm web/app/client/design/reset.css
```

---

### Step 5: Create Router

#### 5.1 Create Router Directory

```bash
mkdir -p web/app/client/router
```

#### 5.2 Create `router/types.ts`

**New file:** `web/app/client/router/types.ts`

```typescript
export interface RouteConfig {
  component: string;
  title: string;
}

export interface RouteMatch {
  config: RouteConfig;
  params: Record<string, string>;
  query: Record<string, string>;
}
```

#### 5.3 Create `router/routes.ts`

**New file:** `web/app/client/router/routes.ts`

```typescript
import type { RouteConfig } from './types';

export const routes: Record<string, RouteConfig> = {
  '': { component: 'al-home-view', title: 'Home' },
  '*': { component: 'al-not-found-view', title: 'Not Found' },
};
```

#### 5.4 Create `router/router.ts`

**New file:** `web/app/client/router/router.ts`

```typescript
import { routes } from './routes';
import type { RouteMatch } from './types';

let routerInstance: Router | null = null;

export function navigate(path: string): void {
  routerInstance?.navigate(path);
}

export class Router {
  private container: HTMLElement;
  private basePath: string;

  constructor(containerId: string) {
    const el = document.getElementById(containerId);
    if (!el) throw new Error(`Container #${containerId} not found`);
    this.container = el;
    this.basePath =
      document.querySelector('base')?.getAttribute('href')?.replace(/\/$/, '') ??
      '/app';
    routerInstance = this;
  }

  navigate(path: string, pushState: boolean = true): void {
    const [pathPart, queryPart] = path.split('?');
    const normalized = this.normalizePath(pathPart);
    const query = this.parseQuery(queryPart);
    const match = this.match(normalized, query);

    if (pushState) {
      let fullPath = `${this.basePath}/${normalized}`.replace(/\/+/g, '/');
      if (queryPart) fullPath += `?${queryPart}`;
      history.pushState(null, '', fullPath);
    }

    document.title = `${match.config.title} - Agent Lab`;
    this.mount(match);
  }

  start(): void {
    this.navigate(this.currentPath(), false);

    window.addEventListener('popstate', () => {
      this.navigate(this.currentPath(), false);
    });
  }

  private currentPath(): string {
    const pathname = location.pathname;
    if (pathname.startsWith(this.basePath)) {
      return pathname.slice(this.basePath.length).replace(/^\//, '');
    }
    return pathname.replace(/^\//, '');
  }

  private match(path: string, query: Record<string, string>): RouteMatch {
    const segments = path.split('/').filter(Boolean);

    if (routes[path]) {
      return { config: routes[path], params: {}, query };
    }

    for (const [pattern, config] of Object.entries(routes)) {
      if (pattern === '*') continue;

      const patternSegments = pattern.split('/').filter(Boolean);

      if (patternSegments.length !== segments.length) continue;

      const params: Record<string, string> = {};
      let matched = true;

      for (let i = 0; i < patternSegments.length; i++) {
        const pat = patternSegments[i];
        const seg = segments[i];

        if (pat.startsWith(':')) {
          params[pat.slice(1)] = seg;
        } else if (pat !== seg) {
          matched = false;
          break;
        }
      }

      if (matched) {
        return { config, params, query };
      }
    }

    return { config: routes['*'], params: { path }, query };
  }

  private mount(match: RouteMatch): void {
    this.container.innerHTML = '';
    const el = document.createElement(match.config.component);

    for (const [key, value] of Object.entries(match.params)) {
      el.setAttribute(key, value);
    }

    for (const [key, value] of Object.entries(match.query)) {
      el.setAttribute(key, value);
    }

    this.container.appendChild(el);
  }

  private normalizePath(path: string): string {
    let normalized = path.replace(/^\//, '');
    const baseWithoutSlash = this.basePath.replace(/^\//, '');
    if (normalized.startsWith(baseWithoutSlash)) {
      normalized = normalized.slice(baseWithoutSlash.length).replace(/^\//, '');
    }
    return normalized;
  }

  private parseQuery(queryString?: string): Record<string, string> {
    if (!queryString) return {};

    const params = new URLSearchParams(queryString);
    const result: Record<string, string> = {};
    for (const [key, value] of params) {
      result[key] = value;
    }
    return result;
  }
}
```

#### 5.5 Create `router/index.ts`

**New file:** `web/app/client/router/index.ts`

```typescript
export { Router, navigate } from './router';
export type { RouteConfig, RouteMatch } from './types';
```

---

### Step 6: Create Baseline Views

#### 6.1 Create Views Directory

```bash
mkdir -p web/app/client/views
```

#### 6.2 Create `views/home-view.ts`

**New file:** `web/app/client/views/home-view.ts`

```typescript
import { LitElement, html, unsafeCSS } from 'lit';
import { customElement } from 'lit/decorators.js';
import styles from './home-view.css?inline';

@customElement('al-home-view')
export class HomeView extends LitElement {
  static styles = unsafeCSS(styles);

  render() {
    return html`
      <div class="container">
        <h1>Agent Lab</h1>
        <p>Workflow execution and monitoring interface.</p>
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    'al-home-view': HomeView;
  }
}
```

#### 6.3 Create `views/home-view.css`

**New file:** `web/app/client/views/home-view.css`

```css
:host {
  display: flex;
  align-items: center;
  justify-content: center;
}

.container {
  text-align: center;
}

h1 {
  margin-bottom: var(--space-4);
}

p {
  color: var(--color-1);
}
```

#### 6.4 Create `views/not-found-view.ts`

**New file:** `web/app/client/views/not-found-view.ts`

```typescript
import { LitElement, html, unsafeCSS } from 'lit';
import { customElement, property } from 'lit/decorators.js';
import styles from './not-found-view.css?inline';

@customElement('al-not-found-view')
export class NotFoundView extends LitElement {
  static styles = unsafeCSS(styles);

  @property({ type: String }) path?: string;

  render() {
    return html`
      <div class="container">
        <h1>404</h1>
        <p>Page not found${this.path ? `: /${this.path}` : ''}</p>
        <a href="">Return home</a>
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    'al-not-found-view': NotFoundView;
  }
}
```

#### 6.5 Create `views/not-found-view.css`

**New file:** `web/app/client/views/not-found-view.css`

```css
:host {
  display: flex;
  align-items: center;
  justify-content: center;
}

.container {
  text-align: center;
}

h1 {
  font-size: var(--text-4xl);
  margin-bottom: var(--space-2);
}

p {
  color: var(--color-1);
  margin-bottom: var(--space-4);
}

a {
  color: var(--blue);
}
```

---

### Step 7: Update App Entry Point

**File:** `web/app/client/app.ts`

Replace entire contents:

```typescript
import './design/index.css';

import { Router } from '@app/router';

import './views/home-view';
import './views/not-found-view';

const router = new Router('app-content');
router.start();
```

---

### Step 8: Convert Go to Single Shell

#### 8.1 Create Shell View Template

**New file:** `web/app/server/views/shell.html`

```html
{{ define "content" }}{{ end }}
```

#### 8.2 Update Go Module

**File:** `web/app/app.go`

Replace `views` variable:

```go
var views = []web.ViewDef{
	{Route: "/{path...}", Template: "shell.html", Title: "agent-lab", Bundle: "app"},
}
```

Remove `errorViews` variable.

Update `NewModule` function - remove `allViews` and use `views` directly:

```go
func NewModule(basePath string) (*module.Module, error) {
	ts, err := web.NewTemplateSet(
		layoutFS,
		viewFS,
		"server/layouts/*.html",
		"server/views",
		basePath,
		views,
	)
	if err != nil {
		return nil, err
	}

	router := buildRouter(ts, basePath)
	return module.New(basePath, router), nil
}
```

Update `buildRouter` to accept `basePath` and use single shell handler:

```go
func buildRouter(ts *web.TemplateSet, basePath string) http.Handler {
	r := web.NewRouter()

	r.HandleFunc("GET /{path...}", ts.PageHandler("app.html", views[0]))

	r.Handle("GET /dist/", http.FileServer(http.FS(distFS)))

	for _, route := range web.PublicFileRoutes(publicFS, "public", publicFiles...) {
		r.HandleFunc(route.Method+" "+route.Pattern, route.Handler)
	}

	return r
}
```

#### 8.3 Delete Old View Templates

```bash
rm web/app/server/views/home.html
rm web/app/server/views/components.html
rm web/app/server/views/404.html
```

---

### Step 9: Update Shell Template

**File:** `web/app/server/layouts/app.html`

Replace entire contents:

```html
<!DOCTYPE html>
<html lang="en">

<head>
  <base href="{{ .BasePath }}/">
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{ .Title }} - Agent Lab</title>
  <link rel="icon" type="image/x-icon" href="favicon.ico">
  <link rel="apple-touch-icon" sizes="180x180" href="apple-touch-icon.png">
  <link rel="icon" type="image/png" sizes="32x32" href="favicon-32x32.png">
  <link rel="icon" type="image/png" sizes="16x16" href="favicon-16x16.png">
  <link rel="stylesheet" href="dist/{{ .Bundle }}.css">
</head>

<body>
  <header class="app-header">
    <a href="" class="brand">Agent Lab</a>
    <nav>
      <a href="workflows">Workflows</a>
      <a href="documents">Documents</a>
      <a href="profiles">Profiles</a>
      <a href="agents">Agents</a>
      <a href="providers">Providers</a>
    </nav>
  </header>
  <main id="app-content">
    {{ block "content" . }}{{ end }}
  </main>

  <script type="module" src="dist/{{ .Bundle }}.js"></script>
</body>

</html>
```

---

## Verification

1. **Install dependencies:**
   ```bash
   cd web && bun install
   ```

2. **Build web assets:**
   ```bash
   bun run build
   ```

3. **Start server:**
   ```bash
   go run ./cmd/server
   ```

4. **Test navigation:**
   - Navigate to `/app` - Home view renders
   - Navigate to `/app/invalid` - Not found view renders with path
   - Use browser back/forward - Router handles history
   - Click nav links - Verify navigation (views not yet implemented show 404)
