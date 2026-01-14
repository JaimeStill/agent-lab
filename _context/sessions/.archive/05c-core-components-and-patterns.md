# Session 05c: Core Components and Patterns

## Problem Context

The milestone guide lists components like `al-button`, `al-input`, `al-badge` - but these duplicate native HTML elements. This session establishes native-first guidelines and provides semantic CSS classes for styling native elements.

## Design Philosophy

**Goal**: Keep the frontend as native as possible so a designer can extend it rather than fight an opinionated system.

**Key Decisions**:
- Server-side rendering with traditional form submissions and page reloads
- Semantic CSS classes for styling native elements
- Web components only for functionality HTML cannot provide (SSE, D3 charts)
- Go template partials for repeated compositions (deferred to later sessions)

## Implementation

### Step 1: Create components.css

Create `web/src/design/components.css`:

```css
@layer components {
  .btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: var(--space-2);
    padding: var(--space-2) var(--space-4);
    font-size: var(--text-sm);
    font-family: inherit;
    border: 1px solid var(--divider);
    border-radius: 4px;
    background-color: var(--bg-1);
    color: var(--color);
    cursor: pointer;
  }

  .btn:hover {
    background-color: var(--bg-2);
  }

  .btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-primary {
    background-color: var(--blue);
    border-color: var(--blue);
    color: var(--bg);
  }

  .btn-primary:hover {
    opacity: 0.9;
    background-color: var(--blue);
  }

  .btn-danger {
    background-color: var(--red);
    border-color: var(--red);
    color: var(--bg);
  }

  .btn-danger:hover {
    opacity: 0.9;
    background-color: var(--red);
  }

  .input {
    padding: var(--space-2) var(--space-3);
    font-size: var(--text-base);
    font-family: inherit;
    border: 1px solid var(--divider);
    border-radius: 4px;
    background-color: var(--bg);
    color: var(--color);
  }

  .input:focus {
    outline: 2px solid var(--blue);
    outline-offset: 1px;
  }

  .input-error {
    border-color: var(--red);
  }

  .input-error:focus {
    outline-color: var(--red);
  }

  .form-group {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }

  .form-label {
    font-size: var(--text-sm);
    color: var(--color-1);
  }

  .form-error {
    font-size: var(--text-sm);
    color: var(--red);
  }

  .table {
    width: 100%;
    border-collapse: collapse;
  }

  .table th,
  .table td {
    padding: var(--space-3) var(--space-4);
    text-align: left;
    border-bottom: 1px solid var(--divider);
  }

  .table th {
    font-weight: 600;
    color: var(--color-1);
    background-color: var(--bg-1);
  }

  .table-striped tbody tr:nth-child(odd) {
    background-color: var(--bg-1);
  }

  .badge {
    display: inline-flex;
    align-items: center;
    padding: var(--space-1) var(--space-2);
    font-size: var(--text-xs);
    font-weight: 500;
    border-radius: 4px;
    background-color: var(--bg-2);
    color: var(--color-1);
  }

  .badge-success {
    background-color: var(--green-bg);
    color: var(--green);
  }

  .badge-warning {
    background-color: var(--yellow-bg);
    color: var(--yellow);
  }

  .badge-error {
    background-color: var(--red-bg);
    color: var(--red);
  }
}
```

### Step 2: Add layout utilities to layout.css

Add the following to the end of `web/src/design/layout.css` (inside the `@layer layout` block):

```css
  .stack {
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
  }

  .stack-sm {
    gap: var(--space-2);
  }

  .cluster {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-4);
    align-items: center;
  }

  .cluster-sm {
    gap: var(--space-2);
  }

  .constrain {
    max-width: 24rem;
  }
```

### Step 3: Update styles.css

Add the components.css import to `web/src/design/styles.css` after the layout import:

```css
@import url(./components.css);
```

The file should have imports in this order:
1. reset.css
2. theme.css
3. layout.css
4. components.css

### Step 4: Create page templates

Create `web/templates/pages/home/home.html`:

```html
{{ template "layouts/app.html" . }}

{{ define "content" }}
<div class="stack">
  <h1>Agent Lab</h1>
  <p>Agentic workflow orchestration platform.</p>
</div>
{{ end }}
```

Create `web/templates/pages/components/components.html`:

```html
{{ template "layouts/app.html" . }}

{{ define "content" }}
<div class="stack">
  <h1>Component Styles</h1>

  <section class="stack stack-sm">
    <h2>Buttons</h2>
    <div class="cluster">
      <button class="btn">Default</button>
      <button class="btn btn-primary">Primary</button>
      <button class="btn btn-danger">Danger</button>
      <button class="btn" disabled>Disabled</button>
    </div>
  </section>

  <section class="stack stack-sm">
    <h2>Form Elements</h2>
    <div class="stack constrain">
      <div class="form-group">
        <label class="form-label">Text Input</label>
        <input class="input" type="text" placeholder="Enter text...">
      </div>
      <div class="form-group">
        <label class="form-label">Input with Error</label>
        <input class="input input-error" type="text" value="Invalid value">
        <span class="form-error">This field has an error</span>
      </div>
      <div class="form-group">
        <label class="form-label">Select</label>
        <select class="input">
          <option>Option 1</option>
          <option>Option 2</option>
          <option>Option 3</option>
        </select>
      </div>
      <div class="form-group">
        <label class="form-label">Textarea</label>
        <textarea class="input" rows="3" placeholder="Enter description..."></textarea>
      </div>
    </div>
  </section>

  <section class="stack stack-sm">
    <h2>Badges</h2>
    <div class="cluster cluster-sm">
      <span class="badge">Default</span>
      <span class="badge badge-success">Success</span>
      <span class="badge badge-warning">Warning</span>
      <span class="badge badge-error">Error</span>
    </div>
  </section>

  <section class="stack stack-sm">
    <h2>Table</h2>
    <table class="table">
      <thead>
        <tr>
          <th>Name</th>
          <th>Status</th>
          <th>Actions</th>
        </tr>
      </thead>
      <tbody>
        <tr>
          <td>Item One</td>
          <td><span class="badge badge-success">Active</span></td>
          <td><button class="btn">Edit</button></td>
        </tr>
        <tr>
          <td>Item Two</td>
          <td><span class="badge badge-warning">Pending</span></td>
          <td><button class="btn">Edit</button></td>
        </tr>
        <tr>
          <td>Item Three</td>
          <td><span class="badge badge-error">Failed</span></td>
          <td><button class="btn">Edit</button></td>
        </tr>
      </tbody>
    </table>
  </section>

  <section class="stack stack-sm">
    <h2>Striped Table</h2>
    <table class="table table-striped">
      <thead>
        <tr>
          <th>ID</th>
          <th>Description</th>
          <th>Value</th>
        </tr>
      </thead>
      <tbody>
        <tr>
          <td>1</td>
          <td>First row</td>
          <td>100</td>
        </tr>
        <tr>
          <td>2</td>
          <td>Second row</td>
          <td>200</td>
        </tr>
        <tr>
          <td>3</td>
          <td>Third row</td>
          <td>300</td>
        </tr>
        <tr>
          <td>4</td>
          <td>Fourth row</td>
          <td>400</td>
        </tr>
      </tbody>
    </table>
  </section>
</div>
{{ end }}
```

### Step 5: Update web.go template handling

The current approach parses all templates together, causing `{{ define "content" }}` blocks to collide. Each page needs its own cloned template set.

Replace `web/web.go` with:

```go
package web

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"

	"github.com/JaimeStill/agent-lab/internal/routes"
)

//go:embed dist/*
var distFS embed.FS

//go:embed all:templates
var templateFS embed.FS

func Static() http.HandlerFunc {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic("failed to create dist sub-filesystem: " + err.Error())
	}
	fileServer := http.StripPrefix("/static/", http.FileServer(http.FS(sub)))
	return func(w http.ResponseWriter, r *http.Request) {
		fileServer.ServeHTTP(w, r)
	}
}

type Handler struct {
	layouts *template.Template
}

func NewHandler() (*Handler, error) {
	layouts, err := template.ParseFS(templateFS, "templates/layouts/*.html")
	if err != nil {
		return nil, err
	}
	return &Handler{layouts: layouts}, nil
}

func (h *Handler) page(name string) (*template.Template, error) {
	t, err := h.layouts.Clone()
	if err != nil {
		return nil, err
	}
	return t.ParseFS(templateFS, "templates/pages/"+name)
}

func (h *Handler) Routes() routes.Group {
	return routes.Group{
		Prefix: "/app",
		Routes: []routes.Route{
			{Method: "GET", Pattern: "", Handler: h.serveHome},
			{Method: "GET", Pattern: "/components", Handler: h.serveComponents},
		},
	}
}

func (h *Handler) serveHome(w http.ResponseWriter, r *http.Request) {
	tmpl, err := h.page("home/home.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.ExecuteTemplate(w, "app.html", map[string]string{
		"Title":  "Home",
		"Bundle": "shared",
	})
}

func (h *Handler) serveComponents(w http.ResponseWriter, r *http.Request) {
	tmpl, err := h.page("components/components.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.ExecuteTemplate(w, "app.html", map[string]string{
		"Title":  "Components",
		"Bundle": "shared",
	})
}
```

Key changes:
- `layouts` holds only the base layout templates
- `page()` clones layouts and parses a specific page template, isolating its `{{ define }}` blocks

### Step 6: Add favicon infrastructure

Create `web/public/` directory and copy favicon files:

```bash
mkdir -p web/public
cp ~/Pictures/logos/agent/favicon.ico web/public/
cp ~/Pictures/logos/agent/favicon-16x16.png web/public/
cp ~/Pictures/logos/agent/favicon-32x32.png web/public/
cp ~/Pictures/logos/agent/apple-touch-icon.png web/public/
cp ~/Pictures/logos/agent/site.webmanifest web/public/
```

Update `web/web.go` to embed and serve public files. Add the embed directive alongside the others:

```go
//go:embed public/*
var publicFS embed.FS
```

Add a helper function to create a handler for a specific public file:

```go
func publicFile(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := publicFS.ReadFile("public/" + name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(data))
	}
}
```

Add imports for `bytes` and `time` packages.

Add public file routes to the existing `Routes()` method:

```go
func (h *Handler) Routes() routes.Group {
	return routes.Group{
		Prefix: "/app",
		Routes: []routes.Route{
			{Method: "GET", Pattern: "", Handler: h.serveHome},
			{Method: "GET", Pattern: "/components", Handler: h.serveComponents},
			{Method: "GET", Pattern: "/favicon.ico", Handler: publicFile("favicon.ico")},
			{Method: "GET", Pattern: "/favicon-16x16.png", Handler: publicFile("favicon-16x16.png")},
			{Method: "GET", Pattern: "/favicon-32x32.png", Handler: publicFile("favicon-32x32.png")},
			{Method: "GET", Pattern: "/apple-touch-icon.png", Handler: publicFile("apple-touch-icon.png")},
			{Method: "GET", Pattern: "/site.webmanifest", Handler: publicFile("site.webmanifest")},
		},
	}
}
```

Update `web/templates/layouts/app.html` to add favicon links in `<head>`:

```html
<link rel="icon" type="image/x-icon" href="/app/favicon.ico">
<link rel="apple-touch-icon" sizes="180x180" href="/app/apple-touch-icon.png">
<link rel="icon" type="image/png" sizes="32x32" href="/app/favicon-32x32.png">
<link rel="icon" type="image/png" sizes="16x16" href="/app/favicon-16x16.png">
<link rel="manifest" href="/app/site.webmanifest">
```

### Step 7: Add home heading to nav

Update `web/templates/layouts/app.html` nav section to include a home link heading:

```html
<nav class="app-nav">
  <a href="/app" class="app-nav-brand">Agent Lab</a>
  <div class="app-nav-links">
    <a href="/app/workflows">Workflows</a>
    <a href="/app/documents">Documents</a>
    <a href="/app/profiles">Profiles</a>
    <a href="/app/agents">Agents</a>
    <a href="/app/providers">Providers</a>
    <a href="/app/components">Components</a>
  </div>
</nav>
```

Add styles to `web/src/design/styles.css` for the nav layout:

```css
.app-nav {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-4);
  background-color: var(--bg-1);
  border-bottom: 1px solid var(--divider);
}

.app-nav-brand {
  font-size: var(--text-lg);
  font-weight: 600;
  color: var(--color);
  text-decoration: none;
}

.app-nav-brand:hover {
  color: var(--blue);
}

.app-nav-links {
  display: flex;
  gap: var(--space-4);
}

.app-nav-links a {
  color: var(--color-1);
  text-decoration: none;
}

.app-nav-links a:hover {
  color: var(--color);
}
```

## Validation

1. Run `bun run build` in `web/` directory
2. Run `go vet ./...` from project root
3. Start server with `go run ./cmd/server`
4. Verify favicon appears in browser tab
5. Visit `/app/components` to visually validate all component styles
6. Verify nav has "Agent Lab" heading linking to `/app`
7. Verify dark/light mode switching (toggle system preference)

## Files Changed

| File | Action |
|------|--------|
| `.claude/skills/web-components/SKILL.md` | Updated with native-first guidelines |
| `web/src/design/components.css` | Created |
| `web/src/design/layout.css` | Modified (add layout utilities) |
| `web/src/design/styles.css` | Modified (add import, nav styles) |
| `web/templates/layouts/app.html` | Modified (favicon links, nav layout) |
| `web/templates/pages/home/home.html` | Created |
| `web/templates/pages/components/components.html` | Created |
| `web/public/*` | Created (favicon files) |
| `web/web.go` | Replaced (template isolation, public file routes) |
