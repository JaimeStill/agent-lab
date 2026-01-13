# Session 05b: CSS and Design Tokens

**Status**: Completed (2026-01-13)

## Summary

Established minimal, native-first CSS design system using cascade layers (`@layer`). Added web client handler for `/app` route.

## Implemented

**CSS Architecture**:
- `@layer reset, theme, layout, components;` cascade structure
- `web/src/design/reset.css` - minimal box-sizing, margin reset, reduced-motion
- `web/src/design/theme.css` - color tokens with `prefers-color-scheme` dark/light
- `web/src/design/layout.css` - spacing and typography scale tokens
- `web/src/design/styles.css` - layer orchestration, app shell styles

**Web Client Handler**:
- `web/web.go` - Handler struct with Routes() method
- `/app` route rendering app.html with shared bundle
- Template integration with `{{ .Bundle }}` and `{{ .Title }}`

## Design Tokens

**Colors** (semantic):
- `--bg`, `--bg-1`, `--bg-2` - background levels
- `--color`, `--color-1`, `--color-2` - text levels
- `--divider` - borders/separators

**Colors** (accents):
- `--blue`, `--green`, `--red`, `--yellow`, `--orange`
- Each has `-bg` variant for backgrounds

**Spacing**: `--space-{1,2,3,4,5,6,8,10,12,16}` (0.25rem base)

**Typography**: `--text-{xs,sm,base,lg,xl,2xl,3xl,4xl}`, `--font-sans`, `--font-mono`

## Key Decisions

1. **Native-first**: No opinionated styling. Tokens enable consistency, not custom design.
2. **Tokens defined, not applied**: Colors/spacing available but components opt-in.
3. **Dark/light via media query**: Automatic switching via `prefers-color-scheme`.
4. **No radius/shadow tokens**: Add when concrete need emerges.
5. **Web handler in web package**: Follows `web/docs/docs.go` pattern for consistency.

## Validation

- `bun run build` succeeds
- `go vet ./...` clean
- `/app` renders with proper dark/light theme switching
- Nav bar styled with `--bg-1` and `--divider`
- System fonts applied via `--font-sans`

## Files Changed

**Created**:
- `web/src/design/reset.css`
- `web/src/design/theme.css`
- `web/src/design/layout.css`

**Modified**:
- `web/src/design/styles.css` - layer orchestration
- `web/web.go` - Handler struct and Routes()
- `cmd/server/routes.go` - web handler registration
