# Session 05b: CSS and Design Tokens

## Overview

Establish minimal, native-first CSS design system using cascade layers (`@layer`). Tokens enable consistency and extensibility while preserving native browser styling.

**Guiding Principle**: Native web defaults are the design. Tokens are opt-in tools, not imposed styles.

## Architecture

```
@layer reset, theme, layout, components;
```

| Layer | Purpose |
|-------|---------|
| reset | Minimal box-sizing and margin reset |
| theme | Color and font tokens, body styling only |
| layout | Spacing and typography scale tokens |
| components | Reserved for Session 05c |

## Implementation Steps

### Step 1: Create reset.css

Create `web/src/design/reset.css`:

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
        min-height: 100svh;
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

### Step 2: Create theme.css

Create `web/src/design/theme.css`:

```css
@layer theme {
    :root {
        color-scheme: dark light;

        --font-sans: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
        --font-mono: ui-monospace, "Cascadia Code", "Source Code Pro", Menlo, Consolas, "DejaVu Sans Mono", monospace;
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

### Step 3: Create layout.css

Create `web/src/design/layout.css`:

```css
@layer layout {
    :root {
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
}
```

### Step 4: Update styles.css

Replace contents of `web/src/design/styles.css`:

```css
@layer reset, theme, layout, components;

@import url(./reset.css);
@import url(./theme.css);
@import url(./layout.css);

.app-nav {
    display: flex;
    gap: var(--space-4);
    padding: var(--space-4);
    background-color: var(--bg-1);
    border-bottom: 1px solid var(--divider);
}

.app-nav a {
    color: var(--color-1);
    text-decoration: none;
}

.app-nav a:hover {
    color: var(--color);
}

.app-content {
    padding: var(--space-4);
}
```

### Step 5: Add web client handler

Update `web/web.go` to add the Handler struct and Routes method:

```go
type Handler struct {
    tmpl *template.Template
}

func NewHandler() (*Handler, error) {
    tmpl, err := Templates()
    if err != nil {
        return nil, err
    }
    return &Handler{tmpl: tmpl}, nil
}

func (h *Handler) Routes() routes.Group {
    return routes.Group{
        Prefix: "/app",
        Routes: []routes.Route{
            {Method: "GET", Pattern: "", Handler: h.serveApp},
        },
    }
}

func (h *Handler) serveApp(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    h.tmpl.ExecuteTemplate(w, "app.html", map[string]string{
        "Title":  "Home",
        "Bundle": "shared",
    })
}
```

Add the routes import:

```go
import (
    "github.com/JaimeStill/agent-lab/internal/routes"
)
```

### Step 6: Register web routes

In `cmd/server/routes.go`, register the web handler after the docs handler:

```go
webHandler, err := web.NewHandler()
if err != nil {
    return err
}
r.RegisterGroup(webHandler.Routes())
```

## Verification

1. Build: `cd web && bun run build`
2. Vet: `go vet ./...`
3. Start server: `go run ./cmd/server`
4. Visit http://localhost:8080/app
5. Verify nav bar displays with proper styling (bg-1 background, divider border)
6. Toggle system dark/light mode - background and text should switch
7. Nav links should show color-1, hover to color

## Token Reference

**Colors**: `--bg`, `--bg-1`, `--bg-2`, `--color`, `--color-1`, `--color-2`, `--divider`, `--blue`, `--green`, `--red`, `--yellow`, `--orange` (each accent has `-bg` variant)

**Spacing**: `--space-{1,2,3,4,5,6,8,10,12,16}`

**Typography**: `--text-{xs,sm,base,lg,xl,2xl,3xl,4xl}`, `--font-sans`, `--font-mono`
