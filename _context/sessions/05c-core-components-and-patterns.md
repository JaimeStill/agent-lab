# Session 05c: Core Components and Patterns

## Summary

Established native-first frontend guidelines and implemented foundational CSS component classes for styling native HTML elements. Added favicon infrastructure and home/components page templates.

## What Was Implemented

### Native-First Guidelines
- Updated `.claude/skills/web-development/SKILL.md` with comprehensive guidelines
- Documented when NOT to create components (buttons, inputs, tables, etc.)
- Defined criteria for when components ARE needed (SSE, D3 charts, complex editors)
- Established asset co-location convention for templates

### CSS Architecture
- Created `web/src/design/components.css` with semantic classes:
  - Buttons: `.btn`, `.btn-primary`, `.btn-danger`
  - Forms: `.input`, `.input-error`, `.form-group`, `.form-label`, `.form-error`
  - Tables: `.table`, `.table-striped`
  - Badges: `.badge`, `.badge-success`, `.badge-warning`, `.badge-error`
- Added layout utilities to `layout.css`: `.stack`, `.cluster`, `.constrain`

### Template Infrastructure
- Refactored `web/web.go` with template isolation pattern (clone + parse per page)
- Created page templates: `home/home.html`, `components/components.html`
- Added favicon infrastructure via embedded `web/public/` directory
- Updated nav with brand heading linking to `/app`

### Testing
- Extended `tests/web/web_test.go` with tests for:
  - Home page handler
  - Components page handler
  - Public file serving (favicon)
  - Complete route verification

## Key Decisions

| Decision | Rationale |
|----------|-----------|
| Native-first | Keep frontend extensible for designers, avoid opinionated component library |
| Server-side rendering | Traditional form submissions, page reloads - simpler architecture |
| Semantic CSS classes | Style native elements without creating wrapper components |
| Template isolation | Clone layouts before parsing pages to prevent `{{ define }}` block collisions |
| Public files under /app | Consolidate all web client assets under single route prefix |
| Absolute favicon paths | Relative paths resolve differently at `/app` vs `/app/components` |

## Patterns Established

### Template Structure
```
web/templates/
├── layouts/
│   └── app.html
└── pages/
    └── [page]/
        ├── [page].html
        ├── [page].css    # Optional co-located styles
        └── [page].ts     # Optional co-located scripts
```

### Template Isolation
```go
func (h *Handler) page(name string) (*template.Template, error) {
    t, err := h.layouts.Clone()
    if err != nil {
        return nil, err
    }
    return t.ParseFS(templateFS, "templates/pages/"+name)
}
```

### Public File Handler
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

## Files Changed

| File | Change |
|------|--------|
| `.claude/skills/web-development/SKILL.md` | Rewrote with native-first guidelines |
| `web/src/design/components.css` | Created |
| `web/src/design/layout.css` | Added layout utilities |
| `web/src/design/styles.css` | Added components import, nav styles |
| `web/templates/layouts/app.html` | Added favicon links, nav brand |
| `web/templates/pages/home/home.html` | Created |
| `web/templates/pages/components/components.html` | Created |
| `web/public/*` | Created favicon files |
| `web/web.go` | Refactored with template isolation, public routes |
| `tests/web/web_test.go` | Added handler and route tests |

## Deferred

- Responsive nav design - needs more thought on implementation approach
- `api.ts` fetch wrapper - create when needed
- `component.ts` base class - create when first component is needed
- Template partials - create when domain UIs need them
