# Session 05a: Lit Migration

**Status**: Completed (2026-01-28)

## Summary

Adapted existing web infrastructure for Lit-based SPA architecture. Established client-side routing, minimal CSS foundation with cascade layers, and baseline views.

## Implemented

### Lit Integration
- Added Lit 3.x dependencies (`lit`, `@lit/context`, `@lit-labs/signals`)
- TypeScript configuration for decorators (`experimentalDecorators`, `useDefineForClassFields: false`)
- Path alias `@app/*` → `./app/client/*` for clean imports
- CSS module declaration (`css.d.ts`) for `?inline` imports

### Client-Side Router
- Custom router using History API (`web/app/client/router/`)
- Route matching with parameter extraction (`:id` patterns)
- Query string parsing
- Programmatic navigation via `navigate()` export
- Component mounting based on route config

### CSS Architecture
- Cascade layers: `@layer tokens, reset, theme`
- Design tokens: fonts, spacing scale, typography scale, colors (dark/light)
- App-shell layout: 100dvh flex column, scroll containment
- External CSS pattern with `?inline` imports for Shadow DOM

### Baseline Views
- Home view (`lab-home-view`) - landing page
- Not found view (`lab-not-found-view`) - 404 with path display
- Component prefix: `lab-`

### Go Shell Pattern
- Single catch-all route `/{path...}` → `shell.html`
- Client-side router handles all view mounting
- Hard boundary: Go owns data/routing, Lit owns presentation

## Key Decisions

1. **Component prefix**: `lab-` (agent-lab) rather than `al-`
2. **Minimal CSS**: Removed unused opinionated styles (components.css, layout utilities)
3. **Path aliases**: Single wildcard pattern vs multiple specific aliases
4. **CSS layers**: Three foundational layers (tokens, reset, theme) plus app layer

## Files Created

```
web/app/client/
├── css.d.ts                    # TypeScript declaration for CSS imports
├── router/
│   ├── types.ts                # RouteConfig, RouteMatch interfaces
│   ├── routes.ts               # Route definitions
│   ├── router.ts               # Router class + navigate()
│   └── index.ts                # Re-exports
├── views/
│   ├── home-view.ts            # Landing page view
│   ├── home-view.css           # Home view styles
│   ├── not-found-view.ts       # 404 view
│   └── not-found-view.css      # Not found styles
└── design/
    ├── index.css               # Layer imports
    ├── core/
    │   ├── tokens.css          # Design tokens
    │   ├── reset.css           # Global reset
    │   └── theme.css           # Theme application
    └── app/
        ├── app.css             # App shell styles
        └── elements.css        # Shadow DOM base (empty)

web/app/server/views/
└── shell.html                  # Empty content block for SPA
```

## Files Modified

- `web/package.json` - Added Lit dependencies
- `web/tsconfig.json` - Decorator support, path alias
- `web/app/client.config.ts` - Vite alias
- `web/app/client/app.ts` - Router initialization
- `web/app/app.go` - Single shell route
- `web/app/server/layouts/app.html` - Updated shell template

## Files Deleted

- `web/app/client/design/components.css`
- `web/app/client/design/styles.css`
- `web/app/client/design/layout.css`
- `web/app/client/design/theme.css`
- `web/app/client/design/reset.css`
- `web/app/server/views/home.html`
- `web/app/server/views/components.html`
- `web/app/server/views/404.html`

## Skill Updates

Updated `.claude/skills/web-development/SKILL.md`:
- Component prefix changed from `al-` to `lab-`
- Added `<client>/` placeholder for generic paths
- Added `css.d.ts` requirement documentation
- Updated all code examples

## Validation

- Home view renders at `/app`
- Not found view renders at `/app/invalid` with path display
- Browser back/forward navigation works
- Nav link clicks trigger client-side navigation
- Dark/light theme responds to system preference
