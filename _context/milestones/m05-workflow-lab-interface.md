# Milestone 5: Workflow Lab Interface - Architecture Document

## Overview

This document captures technical decisions and patterns for the Workflow Lab Interface milestone using Lit-based web architecture. The architecture was validated in the go-lit proof-of-concept project.

## Architecture Approach

### Lit Single-Page Application

The interface uses a client-side SPA architecture with Lit web components:

- **Single HTML Shell**: Go serves one `app.html` for all `/app/*` routes
- **Client-Side Routing**: Custom router handles view mounting via History API
- **Three-Tier Components**: Views → Stateful Components → Pure Elements
- **Service Injection**: `@lit/context` for dependency injection
- **Reactive State**: `@lit-labs/signals` for signal-based reactivity

**Rationale**:
- Hard boundary: Go owns data/routing, Lit owns presentation entirely
- Context-based services reduce prop drilling
- Signals provide fine-grained reactivity
- Patterns validated in go-lit POC

## Directory Structure

```
web/
├── package.json                 # Bun dependencies
├── vite.config.ts               # Root config (merges all clients)
├── vite.client.ts               # Shared config module
├── tsconfig.json                # TypeScript config (@app alias)
├── app/                         # Main app client
│   ├── client/                  # TypeScript source
│   │   ├── app.ts               # Entry point
│   │   ├── router/              # Client-side routing
│   │   │   ├── types.ts
│   │   │   ├── routes.ts
│   │   │   └── router.ts
│   │   ├── shared/              # Cross-domain utilities
│   │   │   ├── api.ts           # API client
│   │   │   └── types.ts         # Shared types
│   │   ├── design/              # CSS architecture
│   │   │   ├── core/            # Foundational system
│   │   │   │   ├── tokens.css   # Design tokens
│   │   │   │   ├── reset.css    # CSS reset
│   │   │   │   ├── theme.css    # Color scheme
│   │   │   │   └── layout.css   # Layout utilities
│   │   │   ├── app/             # Application-specific
│   │   │   │   ├── app.css      # Shell styles
│   │   │   │   └── elements.css # Shadow DOM base styles
│   │   │   └── index.css        # Entry point
│   │   ├── views/               # Route-level components
│   │   │   └── home-view.ts
│   │   └── <domain>/            # Domain modules
│   │       ├── types.ts         # Domain types
│   │       ├── service.ts       # Context + interface + factory
│   │       ├── views/           # Domain views
│   │       ├── components/      # Stateful components
│   │       └── elements/        # Pure elements
│   ├── dist/                    # Build output (gitignored)
│   ├── public/                  # Static assets
│   └── server/                  # Go templates
│       ├── layouts/
│       │   └── app.html         # Single shell template
│       └── views/
│           └── shell.html       # Empty content block
├── scalar/                      # OpenAPI UI (unchanged)
└── app.go                       # Module with catch-all route
```

## Technology Decisions

### Dependencies

```json
{
  "dependencies": {
    "lit": "^3.3.2",
    "@lit/context": "^1.1.6",
    "@lit-labs/signals": "^0.2.0"
  },
  "devDependencies": {
    "vite": "^7.3.1",
    "typescript": "^5.9.3",
    "@scalar/api-reference": "^1.43.10"
  }
}
```

### Build Pipeline

| Tool | Purpose |
|------|---------|
| Bun | Package management |
| Vite | Build tool with single `@app` alias |
| TypeScript | Type safety, decorators |

**Build Output**: ES modules in `web/app/dist/`, embedded via `go:embed`

**Development**: `bun run dev` (Vite watch) + restart Go server for shell changes

### CSS Architecture

**Cascade Layers** (`@layer`):
```css
@layer reset, theme, layout, components;
```

| Layer | Purpose |
|-------|---------|
| reset | Modern CSS reset |
| theme | Color tokens via `prefers-color-scheme` |
| layout | Spacing, sizing, typography tokens |
| components | Application component styles |

**Design Tokens** (CSS Custom Properties):
```css
:root {
  /* Spacing */
  --space-1: 0.25rem;
  --space-2: 0.5rem;
  --space-4: 1rem;
  --space-6: 1.5rem;

  /* Typography */
  --text-sm: 0.875rem;
  --text-base: 1rem;
  --text-lg: 1.125rem;

  /* Colors (dark mode default, light via media query) */
  --bg: #1a1a1a;
  --bg-1: #242424;
  --color: #e0e0e0;
  --divider: #333;
  --blue: #4a9eff;
}
```

**External Component Styles**:
```typescript
import styles from './component.css?inline';
import { unsafeCSS } from 'lit';

static styles = unsafeCSS(styles);
```

### Component Hierarchy

**Three-Tier Pattern**:

| Tier | Role | Tools |
|------|------|-------|
| Views | Provide services, route-level state | `@provide`, `SignalWatcher` |
| Stateful Components | Consume services, coordinate UI | `@consume`, event handlers |
| Pure Elements | Props in, events out | `@property`, `CustomEvent` |

**Example Structure**:
```
al-execute-view (provides: configService, executionService)
├── al-config-selector (consumes: configService)
│   └── al-config-card (stateless)
├── al-chat-panel (consumes: executionService)
│   ├── al-message-list (consumes: executionService)
│   │   └── al-message-bubble (stateless)
│   └── al-prompt-input (stateless)
```

### Service Pattern

**Consolidated Service Infrastructure** (single file per domain):

```typescript
// <domain>/service.ts
import { createContext } from '@lit/context';
import { Signal } from '@lit-labs/signals';

export interface ConfigService {
  configs: Signal.State<AgentConfig[]>;
  loading: Signal.State<boolean>;
  list(): void;
  find(id: string): AgentConfig | undefined;
  save(config: AgentConfig): void;
  delete(id: string): void;
}

export const configServiceContext = createContext<ConfigService>('config-service');

export function createConfigService(): ConfigService {
  const configs = new Signal.State<AgentConfig[]>([]);
  const loading = new Signal.State<boolean>(false);

  return {
    configs,
    loading,
    list() { /* ... */ },
    find(id) { /* ... */ },
    save(config) { /* ... */ },
    delete(id) { /* ... */ },
  };
}
```

### Router Pattern

**Static Route Mapping**:
```typescript
export const routes: Record<string, RouteConfig> = {
  '': { component: 'al-home-view', title: 'Home' },
  'providers': { component: 'al-provider-list-view', title: 'Providers' },
  'providers/:id': { component: 'al-provider-edit-view', title: 'Edit Provider' },
  'workflows': { component: 'al-workflow-list-view', title: 'Workflows' },
  'workflows/:id/run': { component: 'al-workflow-run-view', title: 'Run Workflow' },
  '*': { component: 'al-not-found-view', title: 'Not Found' },
};
```

**Router Features**:
- Path normalization relative to `<base href>`
- Parameter extraction (`:param` → attribute)
- History API with `popstate` listener
- Component mounting via `document.createElement()`

### App-Shell Scroll Architecture

```css
body {
  display: flex;
  flex-direction: column;
  height: 100dvh;
  overflow: hidden;
}

#app-content {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}
```

Views manage their own scroll regions with `overflow-y: auto` on leaf containers.

## API Surface for Interface

### Domains and Components

| Domain | Complexity | Key Components |
|--------|------------|----------------|
| Providers | Simple | List view, edit view |
| Agents | Medium | List view, edit view, execution panel |
| Documents | Medium | Upload form, page viewer |
| Images | Medium | Render controls, binary display |
| Profiles | Medium | Stage editor, nested forms |
| Workflows | High | Run monitor, confidence chart, comparison |

### Dependency Order

```
Providers → Agents → Documents → Images → Profiles → Workflows
```

Sessions follow this dependency order to establish patterns before complex UIs.

## Go Integration

### Single Shell Pattern

```go
// web/app/app.go
var views = []web.ViewDef{
    {Route: "/{path...}", Template: "shell.html", Title: "agent-lab", Bundle: "app"},
}

func NewModule(basePath string) (*module.Module, error) {
    ts, err := web.NewTemplateSet(layoutFS, viewFS, "server/layouts/*.html", "server/views", basePath, views)
    if err != nil {
        return nil, err
    }

    router := buildRouter(ts, basePath)
    return module.New(basePath, router), nil
}
```

### Shell Template

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <base href="{{ .BasePath }}/">
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{ .Title }}</title>
  <link rel="stylesheet" href="dist/{{ .Bundle }}.css">
</head>
<body>
  <header class="app-header">
    <a href="" class="brand">agent-lab</a>
    <nav>
      <a href="providers">Providers</a>
      <a href="agents">Agents</a>
      <a href="documents">Documents</a>
      <a href="workflows">Workflows</a>
    </nav>
  </header>
  <main id="app-content">
    {{ block "content" . }}{{ end }}
  </main>
  <script type="module" src="dist/{{ .Bundle }}.js"></script>
</body>
</html>
```

### Embedding

```go
//go:embed dist/*
var distFS embed.FS

//go:embed server/layouts/*
var layoutFS embed.FS

//go:embed server/views/*
var viewFS embed.FS

//go:embed public/*
var publicFS embed.FS
```

## Session Overview

| Session | Phase | Focus | Key Deliverable |
|---------|-------|-------|-----------------|
| 05a | Foundation | Lit Migration | Adapt existing: Lit deps, router, shell pattern |
| 05b | Foundation | Design System | Complete CSS layers, tokens, element styles |
| 05c | Foundation | Service Infrastructure | Context services, API client, signal patterns |
| 05d | Config UI | Provider/Agent Config | List/edit views, CRUD patterns |
| 05e | Config UI | Document Upload | Document management, storage integration |
| 05f | Config UI | Profile Management | Stage editor, nested forms |
| 05g | Workflow | Execution Trigger | Profile selection, workflow execute |
| 05h | Workflow | Run Monitoring | SSE integration, live updates |
| 05i | Workflow | Visualization | Confidence charts |
| 05j | Workflow | Comparison | Side-by-side, iteration |

### Session 05a Details

Adapts existing web infrastructure for Lit:

1. Add Lit dependencies to `package.json`
2. Create router from go-lit patterns
3. Update `app.ts` entry point
4. Convert Go routes to single catch-all shell
5. Create home view as baseline

### Existing Infrastructure (from previous 05a-05c)

The following exists and will be adapted:
- Vite build pipeline
- CSS layer structure (partial)
- Go `pkg/web` template infrastructure
- Module mounting pattern

## Success Criteria

From PROJECT.md:
- [ ] View document pages with enhancement filter controls
- [ ] Monitor execution in real-time with progress indicators
- [ ] Visualize confidence score evolution across pages
- [ ] Compare multiple runs side-by-side
- [ ] Adjust agent options and filter overrides, re-execute
- [ ] Complete iteration cycle in < 5 minutes

## Verification Approach

### Per-Session Verification

1. **Build verification**: `bun run build` succeeds, Go embeds correctly
2. **Visual verification**: Pages render with proper styling
3. **Functional verification**: Features work as expected
4. **Integration verification**: Services communicate correctly

### Final Milestone Verification

1. Upload a document
2. Render pages with filter adjustments
3. Execute classify-docs workflow
4. Monitor execution in real-time
5. View confidence charts
6. Compare with previous run
7. Re-execute with modified profile
8. Complete cycle < 5 minutes

## Component Patterns Reference

### View Component (provides services)

```typescript
@customElement('al-config-list-view')
export class ConfigListView extends SignalWatcher(LitElement) {
  @provide({ context: configServiceContext })
  private configService: ConfigService = createConfigService();

  connectedCallback() {
    super.connectedCallback();
    this.configService.list();
  }

  render() {
    return html`<al-config-list></al-config-list>`;
  }
}
```

### Stateful Component (consumes services)

```typescript
@customElement('al-config-list')
export class ConfigList extends SignalWatcher(LitElement) {
  @consume({ context: configServiceContext })
  private configService!: ConfigService;

  private handleDelete(e: CustomEvent<{ id: string }>) {
    this.configService.delete(e.detail.id);
  }

  render() {
    return html`
      ${this.configService.configs.get().map(config => html`
        <al-config-card
          .config=${config}
          @delete=${this.handleDelete}
        ></al-config-card>
      `)}
    `;
  }
}
```

### Pure Element (stateless)

```typescript
@customElement('al-config-card')
export class ConfigCard extends LitElement {
  @property({ type: Object }) config!: AgentConfig;

  private handleDelete() {
    this.dispatchEvent(new CustomEvent('delete', {
      detail: { id: this.config.id },
      bubbles: true,
      composed: true,
    }));
  }

  render() {
    return html`
      <div class="card">
        <h3>${this.config.name}</h3>
        <button @click=${this.handleDelete}>Delete</button>
      </div>
    `;
  }
}
```

### Template Render Methods

```typescript
private renderError() {
  const error = this.service.error.get();
  if (!error) return nothing;
  return html`<div class="error">${error}</div>`;
}

render() {
  return html`
    ${this.renderError()}
    ${this.renderContent()}
  `;
}
```

### Form Handling

```typescript
private handleSubmit(e: Event) {
  e.preventDefault();
  const form = e.target as HTMLFormElement;
  const data = new FormData(form);
  const config = buildConfigFromForm(data, this.config.id);
  this.configService.save(config);
  navigate('config');
}
```
