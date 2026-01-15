---
name: web-development
description: >
  Web development patterns for Milestone 5. Use when implementing frontend
  templates, CSS architecture, or client-side functionality.
  Triggers: Web Components, al-* components, templates, Vite, TypeScript,
  custom elements, shadow DOM, frontend, client-side, CSS classes.
  File patterns: web/**/*.ts, web/**/*.html, web/**/*.css
---

# Web Development Patterns

## When This Skill Applies

- Deciding whether to create a component or use native HTML
- Implementing frontend components in `web/`
- Working with TypeScript custom elements
- Styling native elements with semantic classes

## Native-First Principle

**Goal**: Keep the frontend as native as possible so a designer can extend it rather than fight an opinionated system.

**Architecture**:
- Server-side rendering with traditional form submissions
- Semantic CSS classes for styling native elements
- Web components only for functionality HTML cannot provide

### Don't Create Components For

Use native HTML with CSS classes instead:

| Need | Use |
|------|-----|
| Buttons | `<button class="btn btn-primary">` |
| Inputs | `<input class="input">`, `<textarea>`, `<select>` |
| Badges | `<span class="badge badge-success">` |
| Cards | `<article>` or `<section>` |
| Lists | `<ul>`, `<ol>`, `<li>` |
| Dialogs | `<dialog>` |
| Forms | `<form>` |
| Tables | `<table class="table">` |

### Create Components When

1. **Native HTML lacks the functionality**
   - SSE streaming connections
   - D3.js or canvas-based visualizations
   - Complex nested data editors

2. **Client-side state management is required**
   - Reactive updates during live execution
   - Real-time data synchronization

## Component Candidates

| Component | Justification | Session |
|-----------|---------------|---------|
| `al-workflow-monitor` | SSE + reactive state for live execution | 05h |
| `al-confidence-chart` | D3.js visualization | 05i |
| `al-stage-editor` | Complex nested data editing (evaluate need) | 05f |

## CSS Classes Reference

### Buttons

```html
<button class="btn">Default</button>
<button class="btn btn-primary">Primary action</button>
<button class="btn btn-danger">Destructive action</button>
```

### Form Elements

```html
<div class="form-group">
  <label class="form-label">Field name</label>
  <input class="input" type="text">
  <span class="form-error">Error message</span>
</div>

<input class="input input-error" type="text">
```

### Tables

```html
<table class="table">
  <thead>...</thead>
  <tbody>...</tbody>
</table>

<table class="table table-striped">...</table>
```

### Badges

```html
<span class="badge">Default</span>
<span class="badge badge-success">Active</span>
<span class="badge badge-warning">Pending</span>
<span class="badge badge-error">Failed</span>
```

## Component Implementation Pattern

When a component IS needed, follow this pattern:

```typescript
class AlWorkflowMonitor extends HTMLElement {
  static observedAttributes = ['workflow-id'];

  connectedCallback() {
    this.render();
    this.connect();
  }

  disconnectedCallback() {
    this.disconnect();
  }

  attributeChangedCallback() {
    this.render();
  }

  private render() {
    // Update DOM based on state
  }

  private connect() {
    // SSE or other async connection
  }

  private disconnect() {
    // Cleanup
  }
}

customElements.define('al-workflow-monitor', AlWorkflowMonitor);
```

**Conventions**:
- `al-` prefix for all custom elements
- Light DOM (no shadow DOM) for global CSS access
- Cleanup in `disconnectedCallback`
- Minimal internal state

## Directory Structure

```
web/
├── client/                      # Client-side TypeScript/CSS
│   ├── app.ts                   # Main entry point → dist/app.js
│   ├── core/                    # Foundation (created when needed)
│   │   ├── api.ts               # Fetch wrapper
│   │   ├── sse.ts               # SSE client
│   │   └── signals.ts           # TC39 Signals wrapper
│   ├── design/                  # Global CSS architecture
│   │   ├── reset.css            # Box-sizing, a11y defaults
│   │   ├── theme.css            # Color tokens, dark/light
│   │   ├── layout.css           # Spacing, typography, layout utilities
│   │   ├── components.css       # Semantic element classes
│   │   └── styles.css           # Layer orchestration
│   └── components/              # Custom elements (when needed)
│       └── al-*.ts
├── scalar/                      # Scalar OpenAPI UI
│   ├── app.ts                   # Entry point → dist/scalar.js
│   ├── index.html               # Scalar mount point
│   └── scalar.go                # Handler and routes
├── server/                      # Go templates (SSR)
│   ├── layouts/
│   │   └── app.html             # Main layout template
│   └── pages/
│       ├── home.html
│       └── [page].html
├── public/                      # Static assets (favicons, manifest)
├── dist/                        # Vite build output (gitignored)
├── web.go                       # Embedding and route definitions
├── vite.config.ts
└── package.json
```

**URL Routing:**
- `/app/*` - SSR pages (Go templates)
- `/dist/*` - Vite-built bundles (JS/CSS)
- `/scalar` - OpenAPI documentation UI
- `/*` - Public files (favicon.ico, etc.)

## Asset Co-location

Global styles live in `client/design/`. Page-specific styles can be co-located with templates when needed:

```
web/server/
├── layouts/
│   └── app.html
└── pages/
    └── workflows/
        ├── list.html
        └── list.css         # Page-scoped styles (optional)
```

**Loading scoped assets**: Entry files import the assets they need:

```typescript
// client/app.ts
import '@design/styles.css';
import '../server/pages/workflows/list.css';  // If page-specific styles exist
```

**When to co-locate**: Only create scoped CSS when styles are unique to that template. Prefer global utilities in `client/design/` when patterns are reusable.

## Core Principles

**Separation of Concerns**:
- Styles belong in `.css` files
- Markup belongs in `.html` files
- Code belongs in `.ts` files
- Never use inline `style` attributes in templates

**Exception**: Third-party library overrides (e.g., Scalar font variables in `web/scalar/index.html`) may use `<style>` in `<head>` when the library doesn't expose CSS custom properties.

## Anti-Patterns

- Creating `al-button` when `<button class="btn">` works
- Adding client-side validation when server validation suffices
- Building component wrappers for native elements
- Using shadow DOM when global styles should apply
- Using inline `style` attributes instead of CSS classes
