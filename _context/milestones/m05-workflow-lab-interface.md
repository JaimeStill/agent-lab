# Milestone 5: Workflow Lab Interface - Architecture Document

## Overview

This document captures technical decisions and patterns for the Workflow Lab Interface milestone. It provides context for implementation guides and preserves architectural decisions across sessions.

## Architecture Approach

### Hybrid (Go Templates + Web Component Islands)

The interface uses a hybrid architecture combining server-rendered HTML with client-side interactive components:

- **Go Templates**: Render page shell, navigation, lists, and initial state
- **Web Components**: Enhance interactive regions (monitoring, charts, forms)
- **Server-Driven Routing**: Full page reloads, no client-side router
- **Route-Scoped Bundles**: Each page loads only the JS/CSS it needs

**Rationale**:
- Fast initial render (server HTML)
- Smaller JavaScript bundles than pure SPA
- Simpler routing (server handles)
- Aligns with existing `go:embed` patterns

## Directory Structure

```
web/
├── package.json
├── vite.config.ts
├── tsconfig.json
├── src/
│   ├── core/                    # Foundation layer
│   │   ├── component.ts         # Base component class
│   │   ├── api.ts               # Fetch wrapper
│   │   ├── signals.ts           # TC39 Signals wrapper (Session 05h)
│   │   └── sse.ts               # SSE client (Session 05h)
│   ├── tokens/                  # Design tokens
│   │   ├── reset.css            # Modern CSS reset
│   │   ├── theme.css            # Color tokens + dark/light
│   │   ├── layout.css           # Spacing/sizing tokens
│   │   └── styles.css           # @layer imports + global
│   ├── components/              # Web Components (atomic design)
│   │   ├── atoms/               # al-button, al-input, al-badge
│   │   ├── molecules/           # al-form-field, al-card, al-list-item
│   │   └── organisms/           # al-data-table, al-workflow-monitor
│   └── entries/                 # Route-scoped bundles
│       ├── shared.ts            # Common components
│       ├── providers.ts
│       ├── agents.ts
│       ├── documents.ts
│       ├── profiles.ts
│       └── workflows.ts
├── templates/                   # Go templates (embedded)
│   ├── layouts/
│   │   └── app.html             # App shell with navigation
│   ├── partials/                # Reusable fragments
│   │   ├── pagination.html
│   │   ├── search-bar.html
│   │   └── status-badge.html
│   └── pages/                   # Route destinations
│       ├── providers/
│       ├── agents/
│       ├── documents/
│       ├── profiles/
│       └── workflows/
├── dist/                        # Build output (embedded via go:embed)
└── web.go                       # embed.FS + handlers + template rendering
```

## Technology Decisions

### Build Pipeline

| Tool | Purpose |
|------|---------|
| Bun | Package management (faster than npm) |
| Vite | Build tool with library mode |
| TypeScript | Type safety and IDE support |

**Build Output**: ES modules in `web/dist/`, embedded via `go:embed`

**Development**: `bun run dev` (Vite watch) + restart Go server for template changes

### CSS Architecture

**Cascade Layers** (`@layer`):
```css
@layer reset, theme, layout, components;
```

| Layer | Purpose |
|-------|---------|
| reset | Modern CSS reset with accessibility defaults |
| theme | Color tokens, dark/light via `prefers-color-scheme` |
| layout | Spacing, sizing, typography tokens |
| components | Base component styles |

**Design Tokens** (CSS Custom Properties):
```css
:root {
  /* Colors */
  --color-primary: #2563eb;
  --color-surface: #ffffff;
  --color-text: #1e293b;

  /* Spacing */
  --space-1: 0.25rem;
  --space-4: 1rem;

  /* Typography */
  --font-sans: system-ui, -apple-system, sans-serif;
  --font-mono: ui-monospace, monospace;
}
```

### Web Components

**Naming Convention**: `al-` prefix (agent-lab)
- `al-button`, `al-input`, `al-badge` (atoms)
- `al-form-field`, `al-card` (molecules)
- `al-data-table`, `al-workflow-monitor` (organisms)

**Shadow DOM**: Light DOM for most components (global styles apply)

**Component Pattern**:
```typescript
class AlButton extends HTMLElement {
  static observedAttributes = ['variant', 'disabled'];

  connectedCallback() {
    this.render();
  }

  attributeChangedCallback() {
    this.render();
  }

  private render() {
    // Update DOM based on attributes/properties
  }
}

customElements.define('al-button', AlButton);
```

### State Management

**TC39 Signals** (via signal-polyfill):
- Introduced in Session 05h when SSE monitoring is built
- Used for reactive state within interactive islands
- Not needed for server-rendered static content

```typescript
import { Signal } from 'signal-polyfill';

const events = new Signal.State<SSEEvent[]>([]);

// Reactive updates
events.set([...events.get(), newEvent]);
```

### Real-Time Communication

**SSE (Server-Sent Events)**:
- Workflow execution monitoring
- Agent chat/vision streaming

```typescript
class SSEClient {
  private source: EventSource | null = null;

  connect(url: string, onEvent: (event: SSEEvent) => void) {
    this.source = new EventSource(url);
    this.source.onmessage = (e) => onEvent(JSON.parse(e.data));
  }

  disconnect() {
    this.source?.close();
  }
}
```

## API Surface for Interface

### Domains and Components

| Domain | Complexity | Key Components |
|--------|------------|----------------|
| Providers | Simple | `al-data-table`, list/form templates |
| Agents | Medium | `al-data-table`, execution modals |
| Documents | Medium | Upload form, `al-image-viewer` |
| Images | Medium | Render form, binary display |
| Profiles | Medium | `al-stage-editor`, nested forms |
| Workflows | High | `al-workflow-monitor`, `al-confidence-chart` |

### Dependency Order

```
Providers → Agents → Documents → Images → Profiles → Workflows
```

Sessions follow this dependency order to ensure patterns are established before complex UIs.

## Template Patterns

### Layout Template

```html
{{/* templates/layouts/app.html */}}
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{ .Title }} - agent-lab</title>
  <link rel="stylesheet" href="/static/{{ .Bundle }}.css">
</head>
<body>
  <nav class="app-nav">
    <a href="/app/workflows">Workflows</a>
    <a href="/app/documents">Documents</a>
    <a href="/app/profiles">Profiles</a>
    <a href="/app/agents">Agents</a>
    <a href="/app/providers">Providers</a>
  </nav>

  <main class="app-content">
    {{ block "content" . }}{{ end }}
  </main>

  <script type="module" src="/static/{{ .Bundle }}.js"></script>
</body>
</html>
```

### Page Template

```html
{{/* templates/pages/workflows/list.html */}}
{{ template "layouts/app.html" . }}

{{ define "content" }}
<section class="page-header">
  <h1>Workflows</h1>
</section>

<ul class="workflow-list">
  {{ range .Workflows }}
  <li>
    <a href="/app/workflows/{{ .Name }}">{{ .Name }}</a>
    <span>{{ .Description }}</span>
  </li>
  {{ end }}
</ul>

{{/* Interactive island */}}
<al-execution-form
  data-workflows='{{ .WorkflowsJSON }}'
  data-api-base="/api">
</al-execution-form>
{{ end }}
```

## Go Integration

### Handler Pattern

```go
type Handler struct {
    templates *template.Template
    domain    *Domain
}

func NewHandler(templates *template.Template, domain *Domain) *Handler {
    return &Handler{templates: templates, domain: domain}
}

func (h *Handler) WorkflowsPage(w http.ResponseWriter, r *http.Request) {
    workflows := ListWorkflows()

    data := WorkflowsPageData{
        Title:     "Workflows",
        Bundle:    "workflows",
        Workflows: workflows,
    }

    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    if err := h.templates.ExecuteTemplate(w, "workflows/list.html", data); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}
```

### Embedding

```go
//go:embed dist/*
var distFS embed.FS

//go:embed templates/*
var templateFS embed.FS

func Static() http.Handler {
    sub, _ := fs.Sub(distFS, "dist")
    return http.FileServer(http.FS(sub))
}
```

## Session Overview

| Session | Phase | Focus | Key Deliverable |
|---------|-------|-------|-----------------|
| 05a | Foundation | Web Infrastructure | Vite + Go embedding pipeline |
| 05b | Foundation | Design Tokens | @layer CSS architecture |
| 05c | Foundation | Core Components | Atoms/molecules, base patterns |
| 05d | Config UI | Providers + Agents | Data table, CRUD patterns |
| 05e | Config UI | Documents + Images | Upload, image viewer |
| 05f | Config UI | Profiles | Nested resource editor |
| 05g | Workflow | Execution Trigger | Profile selection, execute |
| 05h | Workflow | Run Monitoring | SSE integration, signals |
| 05i | Workflow | Confidence Charts | D3.js visualization |
| 05j | Workflow | Comparison | Side-by-side, iteration |

## Success Criteria

From PROJECT.md:
- [ ] View document pages with enhancement filter controls
- [ ] Monitor execution in real-time with progress indicators
- [ ] Visualize confidence score evolution across pages
- [ ] Compare multiple runs side-by-side
- [ ] Adjust agent options and filter overrides, re-execute
- [ ] Complete iteration cycle in < 5 minutes

## Verification Approach

Each session includes:
1. **Build verification**: `bun run build` succeeds, Go embeds correctly
2. **Visual verification**: Pages render with proper styling
3. **Functional verification**: Features work as expected
4. **Integration verification**: Components communicate correctly

Final milestone verification:
1. Upload a document
2. Render pages with filter adjustments
3. Execute classify-docs workflow
4. Monitor execution in real-time
5. View confidence charts
6. Compare with previous run
7. Re-execute with modified profile
8. Complete cycle < 5 minutes
