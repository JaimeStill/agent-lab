---
name: web-development
description: >
  REQUIRED for web client development with Lit. Use when creating views,
  components, elements, services, or styling with CSS layers.
  Triggers: web/app/client/, LitElement, @customElement, @provide, @consume,
  SignalWatcher, design/tokens, "create component", "add view".
  File patterns: web/**/*.ts, web/**/*.css, web/**/*.go, pkg/web/*.go
---

# Web Development with Lit

## When This Skill Applies

- Creating or modifying web client code in `web/app/client/`
- Implementing Lit components (views, stateful components, elements)
- Working with services and context-based dependency injection
- Styling with CSS cascade layers and design tokens
- Integrating Go server with Lit client

## Architecture Overview

### Hard Boundary Principle

**Go owns data/routing, Lit owns presentation entirely.**

- Go serves a single HTML shell for all `/app/*` routes
- Client-side router handles view mounting
- No server-side view awareness for client routes

### Three-Tier Component Hierarchy

| Tier | Role | Tools | Example |
|------|------|-------|---------|
| Views | Provide services, route-level | `@provide`, `SignalWatcher` | `al-provider-list-view` |
| Stateful Components | Consume services, coordinate UI | `@consume`, event handlers | `al-provider-list` |
| Pure Elements | Props in, events out | `@property`, `CustomEvent` | `al-provider-card` |

## Component Patterns

### View Component (provides services)

```typescript
import { LitElement, html } from 'lit';
import { customElement } from 'lit/decorators.js';
import { provide } from '@lit/context';
import { SignalWatcher } from '@lit-labs/signals';
import { configServiceContext, createConfigService, ConfigService } from './service';

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

declare global {
  interface HTMLElementTagNameMap {
    'al-config-list-view': ConfigListView;
  }
}
```

### Stateful Component (consumes services)

```typescript
import { LitElement, html, css, unsafeCSS } from 'lit';
import { customElement } from 'lit/decorators.js';
import { consume } from '@lit/context';
import { SignalWatcher } from '@lit-labs/signals';
import { configServiceContext, ConfigService } from './service';
import styles from './config-list.css?inline';

@customElement('al-config-list')
export class ConfigList extends SignalWatcher(LitElement) {
  static styles = unsafeCSS(styles);

  @consume({ context: configServiceContext })
  private configService!: ConfigService;

  private handleDelete(e: CustomEvent<{ id: string }>) {
    this.configService.delete(e.detail.id);
  }

  private renderConfigs() {
    return this.configService.configs.get().map(
      (config) => html`
        <al-config-card
          .config=${config}
          @delete=${this.handleDelete}
        ></al-config-card>
      `
    );
  }

  render() {
    return html`<div class="grid">${this.renderConfigs()}</div>`;
  }
}
```

### Pure Element (stateless)

```typescript
import { LitElement, html, unsafeCSS } from 'lit';
import { customElement, property } from 'lit/decorators.js';
import type { AgentConfig } from './types';
import styles from './config-card.css?inline';

@customElement('al-config-card')
export class ConfigCard extends LitElement {
  static styles = unsafeCSS(styles);

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
        <p>${this.config.provider.name} / ${this.config.model.name}</p>
        <button @click=${this.handleDelete}>Delete</button>
      </div>
    `;
  }
}
```

## Service Infrastructure

### Consolidated Service File

Each domain has a single `service.ts` exporting context, interface, and factory:

```typescript
// <domain>/service.ts
import { createContext } from '@lit/context';
import { Signal } from '@lit-labs/signals';

export interface ConfigService {
  configs: Signal.State<AgentConfig[]>;
  loading: Signal.State<boolean>;
  error: Signal.State<string | null>;

  list(): void;
  find(id: string): AgentConfig | undefined;
  save(config: AgentConfig): void;
  delete(id: string): void;
}

export const configServiceContext = createContext<ConfigService>('config-service');

export function createConfigService(): ConfigService {
  const configs = new Signal.State<AgentConfig[]>([]);
  const loading = new Signal.State<boolean>(false);
  const error = new Signal.State<string | null>(null);

  return {
    configs,
    loading,
    error,

    list() {
      loading.set(true);
      api.get<AgentConfig[]>('/providers')
        .then((result) => {
          if (result.ok) {
            configs.set(result.data);
          } else {
            error.set(result.error);
          }
        })
        .finally(() => loading.set(false));
    },

    find(id: string) {
      return configs.get().find((c) => c.id === id);
    },

    save(config: AgentConfig) {
      // API call + update local state
    },

    delete(id: string) {
      // API call + update local state
    },
  };
}
```

## CSS Architecture

### Cascade Layers

```css
/* design/index.css */
@layer reset, theme, layout, components;

@import './core/reset.css' layer(reset);
@import './core/theme.css' layer(theme);
@import './core/tokens.css' layer(theme);
@import './core/layout.css' layer(layout);
@import './app/app.css' layer(components);
```

### Design Tokens

```css
/* design/core/tokens.css */
:root {
  /* Spacing scale */
  --space-1: 0.25rem;
  --space-2: 0.5rem;
  --space-3: 0.75rem;
  --space-4: 1rem;
  --space-6: 1.5rem;
  --space-8: 2rem;

  /* Typography */
  --text-xs: 0.75rem;
  --text-sm: 0.875rem;
  --text-base: 1rem;
  --text-lg: 1.125rem;
  --text-xl: 1.25rem;

  /* Colors (dark mode default) */
  --bg: #1a1a1a;
  --bg-1: #242424;
  --bg-2: #2a2a2a;
  --color: #e0e0e0;
  --color-muted: #888;
  --divider: #333;
  --blue: #4a9eff;
  --green: #4ade80;
  --red: #f87171;
}

@media (prefers-color-scheme: light) {
  :root {
    --bg: #ffffff;
    --bg-1: #f5f5f5;
    --bg-2: #e5e5e5;
    --color: #1a1a1a;
    --color-muted: #666;
    --divider: #ddd;
  }
}
```

### Layout Utilities

```css
/* design/core/layout.css */
.stack { display: flex; flex-direction: column; gap: var(--space-4); }
.stack-sm { display: flex; flex-direction: column; gap: var(--space-2); }
.cluster { display: flex; flex-wrap: wrap; gap: var(--space-4); align-items: center; }
.constrain { max-width: 80ch; margin-inline: auto; }
```

### External Component Styles

Co-locate CSS with components, import with `?inline`:

```typescript
import styles from './component.css?inline';
import { unsafeCSS } from 'lit';

static styles = unsafeCSS(styles);
```

Component CSS imports shared element styles:

```css
/* component.css */
@import '@app/design/app/elements.css';

:host {
  display: block;
  background: var(--bg-1);
  border: 1px solid var(--divider);
  padding: var(--space-4);
}
```

### App-Shell Scroll Architecture

Body fills viewport, never scrolls; views manage own scroll regions:

```css
body {
  display: flex;
  flex-direction: column;
  height: 100dvh;
  margin: 0;
  overflow: hidden;
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

View pattern for scroll regions:

```css
:host {
  display: flex;
  flex-direction: column;
}

.scrollable-content {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}
```

## Router Pattern

### Route Definition

```typescript
// router/routes.ts
export interface RouteConfig {
  component: string;
  title: string;
}

export const routes: Record<string, RouteConfig> = {
  '': { component: 'al-home-view', title: 'Home' },
  'providers': { component: 'al-provider-list-view', title: 'Providers' },
  'providers/:id': { component: 'al-provider-edit-view', title: 'Edit Provider' },
  '*': { component: 'al-not-found-view', title: 'Not Found' },
};
```

### Navigation

```typescript
import { navigate } from '@app/router';

// Programmatic navigation
navigate('providers');
navigate(`providers/${id}`);

// Template links (router intercepts clicks)
html`<a href="providers">View Providers</a>`
```

## Template Patterns

### Render Methods

Extract complex template logic into private `renderXxx()` methods:

```typescript
import { nothing } from 'lit';

private renderError() {
  const error = this.service.error.get();
  if (!error) return nothing;
  return html`<div class="error">${error}</div>`;
}

private renderLoading() {
  if (!this.service.loading.get()) return nothing;
  return html`<div class="loading">Loading...</div>`;
}

render() {
  return html`
    ${this.renderError()}
    ${this.renderLoading()}
    ${this.renderContent()}
  `;
}
```

### Form Handling

Extract values on submit via FormData:

```typescript
function buildConfigFromForm(form: HTMLFormElement, id: string): AgentConfig {
  const data = new FormData(form);
  return {
    id,
    name: data.get('name') as string,
    // ...
  };
}

private handleSubmit(e: Event) {
  e.preventDefault();
  const form = e.target as HTMLFormElement;
  const config = buildConfigFromForm(form, this.config.id);
  this.configService.save(config);
  navigate('config');
}

render() {
  return html`
    <form @submit=${this.handleSubmit}>
      <input name="name" .value=${this.config.name} required />
      <button type="submit">Save</button>
    </form>
  `;
}
```

### Host Attribute Reflection

Reflect state to host for CSS-driven layout changes:

```typescript
@state() private expanded = false;

updated(changed: Map<string, unknown>) {
  if (changed.has('expanded')) {
    this.toggleAttribute('expanded', this.expanded);
  }
}
```

```css
:host { grid-template-rows: auto auto 1fr; }
:host([expanded]) { grid-template-rows: auto 1fr 1fr; }
```

### Object URL Lifecycle

Manage blob URLs to prevent memory leaks:

```typescript
private imageUrls = new Map<File, string>();

disconnectedCallback() {
  super.disconnectedCallback();
  this.imageUrls.forEach((url) => URL.revokeObjectURL(url));
  this.imageUrls.clear();
}

private getImageUrl(file: File): string {
  let url = this.imageUrls.get(file);
  if (!url) {
    url = URL.createObjectURL(file);
    this.imageUrls.set(file, url);
  }
  return url;
}
```

## Go Integration

### Single Shell Pattern

Go serves one template for all `/app/*` routes:

```go
var views = []web.ViewDef{
    {Route: "/{path...}", Template: "shell.html", Title: "agent-lab", Bundle: "app"},
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
  <header class="app-header"><!-- nav --></header>
  <main id="app-content">{{ block "content" . }}{{ end }}</main>
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

//go:embed public/*
var publicFS embed.FS
```

## Naming Conventions

- **Component prefix**: `al-` (agent-lab)
- **Views**: `al-<domain>-<action>-view` (e.g., `al-provider-list-view`)
- **Components**: `al-<domain>-<name>` (e.g., `al-provider-list`)
- **Elements**: `al-<name>` (e.g., `al-config-card`)
- **Avoid HTMLElement conflicts**: Use `configId` not `id`, `heading` not `title`

## Anti-Patterns

### Do Not

- Create components for native HTML (buttons, inputs, badges)
- Use shadow DOM when global styles should apply
- Store service references in component state
- Skip `SignalWatcher` mixin when using signals
- Use `height: 100%` in flex contexts (use `flex: 1` instead)
- Forget `min-height: 0` for scroll boundaries
- Use inline `style` attributes (use CSS classes)
- Access `this.id` or `this.title` (conflicts with HTMLElement)

### Prefer

- Native HTML with CSS classes for simple elements
- `@provide`/`@consume` over prop drilling
- `nothing` from Lit for conditional non-rendering
- FormData extraction over controlled inputs
- Event delegation over individual handlers
- `disconnectedCallback` cleanup for resources
