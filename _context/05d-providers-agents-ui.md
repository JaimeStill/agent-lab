# Session 05d: Providers + Agents UI

## Overview

Build reusable client infrastructure in `web/app/client/core/` then create Providers and Agents list views with CRUD operations.

**Approach**: Minimal infrastructure, native APIs where sufficient, web components for reusable controls.

---

## Phase 0: Refactor Pages to Views

Rename existing infrastructure from "page" to "view" terminology for consistency.

### 0.1 Rename `pkg/web/pages.go` to `pkg/web/views.go`

```bash
mv pkg/web/pages.go pkg/web/views.go
```

### 0.2 Update `pkg/web/views.go`

Rename types and update package doc:

```go
// Package web provides infrastructure for serving web views with Go templates.
// It supports pre-parsed templates for zero per-request overhead and
// declarative view definitions for simplified route generation.
package web
```

```go
// ViewDef defines a view with its route, template file, title, and bundle name.
type ViewDef struct {
	Route    string
	Template string
	Title    string
	Bundle   string
}

// ViewData contains the data passed to view templates during rendering.
// BasePath enables portable URL generation in templates via {{ .BasePath }}.
type ViewData struct {
	Title    string
	Bundle   string
	BasePath string
	Data     any
}
```

Update `TemplateSet` internal field and constructor:

```go
type TemplateSet struct {
	views    map[string]*template.Template
	basePath string
}

func NewTemplateSet(layoutFS, viewFS embed.FS, layoutGlob, viewSubdir, basePath string, views []ViewDef) (*TemplateSet, error) {
	layouts, err := template.ParseFS(layoutFS, layoutGlob)
	if err != nil {
		return nil, err
	}

	viewSub, err := fs.Sub(viewFS, viewSubdir)
	if err != nil {
		return nil, err
	}

	viewTemplates := make(map[string]*template.Template, len(views))
	for _, v := range views {
		t, err := layouts.Clone()
		if err != nil {
			return nil, fmt.Errorf("clone layouts for %s: %w", v.Template, err)
		}
		_, err = t.ParseFS(viewSub, v.Template)
		if err != nil {
			return nil, fmt.Errorf("parse template: %s: %w", v.Template, err)
		}
		viewTemplates[v.Template] = t
	}

	return &TemplateSet{
		views:    viewTemplates,
		basePath: basePath,
	}, nil
}
```

Rename handler methods:

```go
func (ts *TemplateSet) ErrorHandler(layout string, view ViewDef, status int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		data := ViewData{
			Title:    view.Title,
			Bundle:   view.Bundle,
			BasePath: ts.basePath,
		}
		if err := ts.Render(w, layout, view.Template, data); err != nil {
			http.Error(w, http.StatusText(status), status)
		}
	}
}

func (ts *TemplateSet) ViewHandler(layout string, view ViewDef) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := ViewData{
			Title:    view.Title,
			Bundle:   view.Bundle,
			BasePath: ts.basePath,
		}
		if err := ts.Render(w, layout, view.Template, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func (ts *TemplateSet) Render(w http.ResponseWriter, layoutName, viewPath string, data ViewData) error {
	t, ok := ts.views[viewPath]
	if !ok {
		return fmt.Errorf("template not found: %s", viewPath)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return t.ExecuteTemplate(w, layoutName, data)
}
```

### 0.3 Rename `web/app/server/pages/` to `web/app/server/views/`

```bash
mv web/app/server/pages web/app/server/views
```

### 0.4 Update `web/app/app.go`

Update embed and slice declarations:

```go
//go:embed server/views/*
var viewFS embed.FS
```

```go
var views = []web.ViewDef{
	{Route: "/{$}", Template: "home.html", Title: "Home", Bundle: "app"},
	{Route: "/components", Template: "components.html", Title: "Components", Bundle: "app"},
}

var errorViews = []web.ViewDef{
	{Template: "404.html", Title: "Not Found", Bundle: "app"},
}
```

Update `NewModule` function:

```go
func NewModule(basePath string) (*module.Module, error) {
	allViews := append(views, errorViews...)
	ts, err := web.NewTemplateSet(
		layoutFS,
		viewFS,
		"server/layouts/*.html",
		"server/views",
		basePath,
		allViews,
	)
	if err != nil {
		return nil, err
	}

	return module.New(
		basePath,
		buildRouter(ts),
	), nil
}
```

Update `buildRouter` function:

```go
func buildRouter(ts *web.TemplateSet) http.Handler {
	r := web.NewRouter()
	r.SetFallback(ts.ErrorHandler(
		"app.html",
		errorViews[0],
		http.StatusNotFound,
	))

	for _, v := range views {
		r.HandleFunc("GET "+v.Route, ts.ViewHandler("app.html", v))
	}

	// ... rest unchanged
}
```

### 0.5 Validate Refactor

```bash
go vet ./...
go run ./cmd/server
```

Navigate to `/app/` and `/app/components` to verify existing views still work.

---

## Phase 1: API Module

### 1.1 Create `web/app/client/core/api/types.ts`

```typescript
export interface PageResult<T> {
  data: T[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

export interface PageRequest {
  page?: number;
  page_size?: number;
  search?: string;
  sort?: string;
}

export type Result<T> =
  | { ok: true; data: T }
  | { ok: false; error: string };
```

### 1.2 Create `web/app/client/core/api/client.ts`

```typescript
import type { Result, PageRequest } from './types';

class ApiClient {
  private baseUrl: string;

  constructor(baseUrl: string = '/api') {
    this.baseUrl = baseUrl;
  }

  private buildUrl(path: string, params?: PageRequest): string {
    const url = new URL(this.baseUrl + path, window.location.origin);
    if (params) {
      if (params.page !== undefined) url.searchParams.set('page', String(params.page));
      if (params.page_size !== undefined) url.searchParams.set('page_size', String(params.page_size));
      if (params.search) url.searchParams.set('search', params.search);
      if (params.sort) url.searchParams.set('sort', params.sort);
    }
    return url.toString();
  }

  async get<T>(path: string, params?: PageRequest): Promise<Result<T>> {
    try {
      const response = await fetch(this.buildUrl(path, params));
      if (!response.ok) {
        const text = await response.text();
        return { ok: false, error: text || response.statusText };
      }
      const data = await response.json() as T;
      return { ok: true, data };
    } catch (err) {
      return { ok: false, error: err instanceof Error ? err.message : 'Unknown error' };
    }
  }

  async post<T>(path: string, body?: unknown): Promise<Result<T>> {
    try {
      const response = await fetch(this.baseUrl + path, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: body ? JSON.stringify(body) : undefined,
      });
      if (!response.ok) {
        const text = await response.text();
        return { ok: false, error: text || response.statusText };
      }
      const data = await response.json() as T;
      return { ok: true, data };
    } catch (err) {
      return { ok: false, error: err instanceof Error ? err.message : 'Unknown error' };
    }
  }

  async put<T>(path: string, body: unknown): Promise<Result<T>> {
    try {
      const response = await fetch(this.baseUrl + path, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
      if (!response.ok) {
        const text = await response.text();
        return { ok: false, error: text || response.statusText };
      }
      const data = await response.json() as T;
      return { ok: true, data };
    } catch (err) {
      return { ok: false, error: err instanceof Error ? err.message : 'Unknown error' };
    }
  }

  async del(path: string): Promise<Result<void>> {
    try {
      const response = await fetch(this.baseUrl + path, {
        method: 'DELETE',
      });
      if (!response.ok) {
        const text = await response.text();
        return { ok: false, error: text || response.statusText };
      }
      return { ok: true, data: undefined };
    } catch (err) {
      return { ok: false, error: err instanceof Error ? err.message : 'Unknown error' };
    }
  }
}

export const api = new ApiClient();
```

### 1.3 Create `web/app/client/core/api/index.ts`

```typescript
export type { PageResult, PageRequest, Result } from './types';
export { api } from './client';
```

---

## Phase 2: HTML Utilities

### 2.1 Create `web/app/client/core/html.ts`

```typescript
const escapeMap: Record<string, string> = {
  '&': '&amp;',
  '<': '&lt;',
  '>': '&gt;',
  '"': '&quot;',
  "'": '&#39;',
};

export function escape(value: unknown): string {
  const str = String(value ?? '');
  return str.replace(/[&<>"']/g, (char) => escapeMap[char]);
}

export function html(
  strings: TemplateStringsArray,
  ...values: unknown[]
): string {
  return strings.reduce((result, str, i) => {
    const value = i < values.length ? escape(values[i]) : '';
    return result + str + value;
  }, '');
}
```

---

## Phase 3: View Lifecycle Module

### 3.1 Create `web/app/client/core/view/types.ts`

```typescript
export interface ViewOptions {
  onMount(): void;
  onDestroy?(): void;
}
```

### 3.2 Create `web/app/client/core/view/create.ts`

```typescript
import type { ViewOptions } from './types';

export function createView(selector: string, options: ViewOptions): void {
  const init = () => {
    const element = document.querySelector(selector);
    if (!element) return;

    options.onMount();

    if (options.onDestroy) {
      window.addEventListener('beforeunload', options.onDestroy);
    }
  };

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
}
```

### 3.3 Create `web/app/client/core/view/index.ts`

```typescript
export type { ViewOptions } from './types';
export { createView } from './create';
```

---

## Phase 4: Domain Types Module

### 4.1 Create `web/app/client/core/domains/providers.ts`

```typescript
export interface Provider {
  id: string;
  name: string;
  config: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface ProviderCommand {
  name: string;
  config: Record<string, unknown>;
}
```

### 4.2 Create `web/app/client/core/domains/agents.ts`

```typescript
export interface Agent {
  id: string;
  name: string;
  config: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface AgentCommand {
  name: string;
  config: Record<string, unknown>;
}
```

### 4.3 Create `web/app/client/core/domains/index.ts`

```typescript
export type { Provider, ProviderCommand } from './providers';
export type { Agent, AgentCommand } from './agents';
```

---

## Phase 5: Core Index

### 5.1 Update `web/app/client/core/index.ts`

```typescript
export * from './api';
export * from './html';
export * from './view';
export * from './domains';
```

---

## Phase 6: Pagination Component

### 6.1 Create `web/app/client/components/al-pagination.ts`

```typescript
export class AlPagination extends HTMLElement {
  static observedAttributes = ['page', 'total-pages'];

  private get page(): number {
    return parseInt(this.getAttribute('page') || '1', 10);
  }

  private get totalPages(): number {
    return parseInt(this.getAttribute('total-pages') || '1', 10);
  }

  connectedCallback() {
    this.render();
    this.addEventListener('click', this.handleClick);
  }

  disconnectedCallback() {
    this.removeEventListener('click', this.handleClick);
  }

  attributeChangedCallback() {
    this.render();
  }

  private handleClick = (e: Event) => {
    const target = e.target as HTMLElement;
    const button = target.closest('button[data-page]') as HTMLButtonElement;
    if (button && !button.disabled) {
      const page = parseInt(button.dataset.page!, 10);
      this.dispatchEvent(new CustomEvent('page-change', {
        detail: { page },
        bubbles: true,
      }));
    }
  };

  private render() {
    const { page, totalPages } = this;

    if (totalPages <= 1) {
      this.innerHTML = '';
      return;
    }

    this.innerHTML = `
      <div class="pagination">
        <button class="btn" data-page="${page - 1}" ${page <= 1 ? 'disabled' : ''}>
          Previous
        </button>
        <span class="pagination-info">Page ${page} of ${totalPages}</span>
        <button class="btn" data-page="${page + 1}" ${page >= totalPages ? 'disabled' : ''}>
          Next
        </button>
      </div>
    `;
  }
}

customElements.define('al-pagination', AlPagination);
```

### 6.2 Update `web/app/client/components/index.ts`

```typescript
export { AlPagination } from './al-pagination';
```

---

## Phase 7: CSS Additions

### 7.1 Update `web/app/client/design/components.css`

Add within `@layer components` (after existing styles):

```css
dialog {
  padding: var(--space-6);
  border: 1px solid var(--divider);
  border-radius: 8px;
  background-color: var(--bg);
  color: var(--color);
  max-width: 32rem;
  width: 100%;
}

dialog::backdrop {
  background-color: rgba(0, 0, 0, 0.5);
}

dialog h2 {
  margin-top: 0;
  margin-bottom: var(--space-4);
}

dialog form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

dialog .dialog-actions {
  display: flex;
  gap: var(--space-2);
  justify-content: flex-end;
  margin-top: var(--space-2);
}

dialog .dialog-actions-split {
  display: flex;
  justify-content: space-between;
  margin-top: var(--space-2);
}

textarea.input {
  resize: vertical;
  min-height: 8rem;
  font-family: var(--font-mono);
}

.table-clickable tbody tr {
  cursor: pointer;
}

.table-clickable tbody tr:hover {
  background-color: var(--bg-1);
}

.view-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-6);
}

.view-header h1 {
  margin: 0;
}

.search-bar {
  display: flex;
  gap: var(--space-3);
  align-items: center;
}

.pagination {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  margin-top: var(--space-4);
}

.pagination-info {
  color: var(--color-1);
  font-size: var(--text-sm);
}

.text-muted {
  color: var(--color-1);
}

.empty-state {
  padding: var(--space-8);
  text-align: center;
  color: var(--color-1);
}
```

---

## Phase 8: Providers View

### 8.1 Create `web/app/server/views/providers.html`

```html
<div data-view="providers">
  <header class="view-header">
    <h1>Providers</h1>
    <div class="search-bar">
      <input
        type="search"
        id="provider-search"
        class="input"
        placeholder="Search providers..."
      />
      <button id="add-provider-btn" class="btn btn-primary">Add Provider</button>
    </div>
  </header>

  <table class="table table-clickable">
    <thead>
      <tr>
        <th>Name</th>
        <th>Type</th>
        <th>Updated</th>
      </tr>
    </thead>
    <tbody id="provider-tbody"></tbody>
  </table>

  <div id="provider-empty" class="empty-state hidden">
    No providers found
  </div>

  <al-pagination id="provider-pagination" page="1" total-pages="1"></al-pagination>

  <dialog id="provider-dialog">
    <h2 id="provider-dialog-title">Add Provider</h2>
    <form id="provider-form">
      <input type="hidden" id="provider-id" />
      <div class="form-group">
        <label class="form-label" for="provider-name">Name</label>
        <input type="text" id="provider-name" class="input" required />
      </div>
      <div class="form-group">
        <label class="form-label" for="provider-config">Configuration (JSON)</label>
        <textarea id="provider-config" class="input" placeholder="{}"></textarea>
      </div>
      <div class="dialog-actions-split">
        <button type="button" id="delete-provider-btn" class="btn btn-danger hidden">
          Delete
        </button>
        <div class="dialog-actions">
          <button type="button" id="cancel-provider-btn" class="btn">Cancel</button>
          <button type="submit" class="btn btn-primary">Save</button>
        </div>
      </div>
    </form>
  </dialog>
</div>
```

### 8.2 Create `web/app/client/core/views/providers.ts`

```typescript
import { createView } from '../view';
import { api, type PageResult, type PageRequest } from '../api';
import { html, escape } from '../html';
import type { Provider, ProviderCommand } from '../domains';

createView('[data-view="providers"]', {
  onMount() {
    const tbody = document.querySelector<HTMLTableSectionElement>('#provider-tbody')!;
    const emptyState = document.querySelector<HTMLElement>('#provider-empty')!;
    const table = tbody.closest('table')!;
    const pagination = document.querySelector<HTMLElement>('#provider-pagination')!;
    const searchInput = document.querySelector<HTMLInputElement>('#provider-search')!;
    const addBtn = document.querySelector<HTMLButtonElement>('#add-provider-btn')!;
    const dialog = document.querySelector<HTMLDialogElement>('#provider-dialog')!;
    const dialogTitle = document.querySelector<HTMLElement>('#provider-dialog-title')!;
    const form = document.querySelector<HTMLFormElement>('#provider-form')!;
    const idInput = document.querySelector<HTMLInputElement>('#provider-id')!;
    const nameInput = document.querySelector<HTMLInputElement>('#provider-name')!;
    const configInput = document.querySelector<HTMLTextAreaElement>('#provider-config')!;
    const deleteBtn = document.querySelector<HTMLButtonElement>('#delete-provider-btn')!;
    const cancelBtn = document.querySelector<HTMLButtonElement>('#cancel-provider-btn')!;

    let currentPage = 1;
    let searchTimeout: number;

    function renderRows(providers: Provider[]): string {
      return providers.map(p => html`
        <tr data-id="${p.id}">
          <td>${p.name}</td>
          <td>${p.config?.type ?? '-'}</td>
          <td>${new Date(p.updated_at).toLocaleDateString()}</td>
        </tr>
      `).join('');
    }

    async function loadProviders(params: PageRequest = {}) {
      const result = await api.get<PageResult<Provider>>('/providers', {
        page: currentPage,
        search: searchInput.value || undefined,
        ...params,
      });

      if (!result.ok) {
        tbody.innerHTML = '';
        emptyState.textContent = `Error: ${result.error}`;
        emptyState.classList.remove('hidden');
        table.classList.add('hidden');
        return;
      }

      const { data, page, total_pages } = result.data;

      if (data.length === 0) {
        tbody.innerHTML = '';
        emptyState.classList.remove('hidden');
        table.classList.add('hidden');
      } else {
        tbody.innerHTML = renderRows(data);
        emptyState.classList.add('hidden');
        table.classList.remove('hidden');
      }

      pagination.setAttribute('page', String(page));
      pagination.setAttribute('total-pages', String(total_pages));
    }

    function openCreateDialog() {
      dialogTitle.textContent = 'Add Provider';
      idInput.value = '';
      nameInput.value = '';
      configInput.value = '{}';
      deleteBtn.classList.add('hidden');
      dialog.showModal();
    }

    function openEditDialog(provider: Provider) {
      dialogTitle.textContent = 'Edit Provider';
      idInput.value = provider.id;
      nameInput.value = provider.name;
      configInput.value = JSON.stringify(provider.config, null, 2);
      deleteBtn.classList.remove('hidden');
      dialog.showModal();
    }

    function closeDialog() {
      dialog.close();
      form.reset();
    }

    async function handleSubmit(e: Event) {
      e.preventDefault();

      let config: Record<string, unknown>;
      try {
        config = JSON.parse(configInput.value || '{}');
      } catch {
        alert('Invalid JSON in configuration');
        return;
      }

      const command: ProviderCommand = {
        name: nameInput.value,
        config,
      };

      const id = idInput.value;
      const result = id
        ? await api.put<Provider>(`/providers/${id}`, command)
        : await api.post<Provider>('/providers', command);

      if (result.ok) {
        closeDialog();
        loadProviders();
      } else {
        alert(`Error: ${result.error}`);
      }
    }

    async function handleDelete() {
      const id = idInput.value;
      if (!id) return;

      if (!confirm('Are you sure you want to delete this provider?')) return;

      const result = await api.del(`/providers/${id}`);
      if (result.ok) {
        closeDialog();
        loadProviders();
      } else {
        alert(`Error: ${result.error}`);
      }
    }

    let cachedProviders: Provider[] = [];

    tbody.addEventListener('click', (e) => {
      const row = (e.target as Element).closest('tr');
      if (row?.dataset.id) {
        const provider = cachedProviders.find(p => p.id === row.dataset.id);
        if (provider) openEditDialog(provider);
      }
    });

    async function loadProvidersWithCache(params: PageRequest = {}) {
      const result = await api.get<PageResult<Provider>>('/providers', {
        page: currentPage,
        search: searchInput.value || undefined,
        ...params,
      });

      if (!result.ok) {
        tbody.innerHTML = '';
        emptyState.textContent = `Error: ${result.error}`;
        emptyState.classList.remove('hidden');
        table.classList.add('hidden');
        return;
      }

      const { data, page, total_pages } = result.data;
      cachedProviders = data;

      if (data.length === 0) {
        tbody.innerHTML = '';
        emptyState.classList.remove('hidden');
        table.classList.add('hidden');
      } else {
        tbody.innerHTML = renderRows(data);
        emptyState.classList.add('hidden');
        table.classList.remove('hidden');
      }

      pagination.setAttribute('page', String(page));
      pagination.setAttribute('total-pages', String(total_pages));
    }

    searchInput.addEventListener('input', () => {
      clearTimeout(searchTimeout);
      searchTimeout = window.setTimeout(() => {
        currentPage = 1;
        loadProvidersWithCache();
      }, 300);
    });

    pagination.addEventListener('page-change', ((e: CustomEvent<{ page: number }>) => {
      currentPage = e.detail.page;
      loadProvidersWithCache();
    }) as EventListener);

    addBtn.addEventListener('click', openCreateDialog);
    cancelBtn.addEventListener('click', closeDialog);
    deleteBtn.addEventListener('click', handleDelete);
    form.addEventListener('submit', handleSubmit);

    loadProvidersWithCache();
  },
});
```

---

## Phase 9: Agents View

### 9.1 Create `web/app/server/views/agents.html`

```html
<div data-view="agents">
  <header class="view-header">
    <h1>Agents</h1>
    <div class="search-bar">
      <input
        type="search"
        id="agent-search"
        class="input"
        placeholder="Search agents..."
      />
      <button id="add-agent-btn" class="btn btn-primary">Add Agent</button>
    </div>
  </header>

  <table class="table table-clickable">
    <thead>
      <tr>
        <th>Name</th>
        <th>Type</th>
        <th>Updated</th>
      </tr>
    </thead>
    <tbody id="agent-tbody"></tbody>
  </table>

  <div id="agent-empty" class="empty-state hidden">
    No agents found
  </div>

  <al-pagination id="agent-pagination" page="1" total-pages="1"></al-pagination>

  <dialog id="agent-dialog">
    <h2 id="agent-dialog-title">Add Agent</h2>
    <form id="agent-form">
      <input type="hidden" id="agent-id" />
      <div class="form-group">
        <label class="form-label" for="agent-name">Name</label>
        <input type="text" id="agent-name" class="input" required />
      </div>
      <div class="form-group">
        <label class="form-label" for="agent-config">Configuration (JSON)</label>
        <textarea id="agent-config" class="input" placeholder="{}"></textarea>
      </div>
      <div class="dialog-actions-split">
        <button type="button" id="delete-agent-btn" class="btn btn-danger hidden">
          Delete
        </button>
        <div class="dialog-actions">
          <button type="button" id="cancel-agent-btn" class="btn">Cancel</button>
          <button type="submit" class="btn btn-primary">Save</button>
        </div>
      </div>
    </form>
  </dialog>
</div>
```

### 9.2 Create `web/app/client/core/views/agents.ts`

```typescript
import { createView } from '../view';
import { api, type PageResult, type PageRequest } from '../api';
import { html, escape } from '../html';
import type { Agent, AgentCommand } from '../domains';

createView('[data-view="agents"]', {
  onMount() {
    const tbody = document.querySelector<HTMLTableSectionElement>('#agent-tbody')!;
    const emptyState = document.querySelector<HTMLElement>('#agent-empty')!;
    const table = tbody.closest('table')!;
    const pagination = document.querySelector<HTMLElement>('#agent-pagination')!;
    const searchInput = document.querySelector<HTMLInputElement>('#agent-search')!;
    const addBtn = document.querySelector<HTMLButtonElement>('#add-agent-btn')!;
    const dialog = document.querySelector<HTMLDialogElement>('#agent-dialog')!;
    const dialogTitle = document.querySelector<HTMLElement>('#agent-dialog-title')!;
    const form = document.querySelector<HTMLFormElement>('#agent-form')!;
    const idInput = document.querySelector<HTMLInputElement>('#agent-id')!;
    const nameInput = document.querySelector<HTMLInputElement>('#agent-name')!;
    const configInput = document.querySelector<HTMLTextAreaElement>('#agent-config')!;
    const deleteBtn = document.querySelector<HTMLButtonElement>('#delete-agent-btn')!;
    const cancelBtn = document.querySelector<HTMLButtonElement>('#cancel-agent-btn')!;

    let currentPage = 1;
    let searchTimeout: number;
    let cachedAgents: Agent[] = [];

    function renderRows(agents: Agent[]): string {
      return agents.map(a => html`
        <tr data-id="${a.id}">
          <td>${a.name}</td>
          <td>${a.config?.type ?? '-'}</td>
          <td>${new Date(a.updated_at).toLocaleDateString()}</td>
        </tr>
      `).join('');
    }

    async function loadAgents(params: PageRequest = {}) {
      const result = await api.get<PageResult<Agent>>('/agents', {
        page: currentPage,
        search: searchInput.value || undefined,
        ...params,
      });

      if (!result.ok) {
        tbody.innerHTML = '';
        emptyState.textContent = `Error: ${result.error}`;
        emptyState.classList.remove('hidden');
        table.classList.add('hidden');
        return;
      }

      const { data, page, total_pages } = result.data;
      cachedAgents = data;

      if (data.length === 0) {
        tbody.innerHTML = '';
        emptyState.classList.remove('hidden');
        table.classList.add('hidden');
      } else {
        tbody.innerHTML = renderRows(data);
        emptyState.classList.add('hidden');
        table.classList.remove('hidden');
      }

      pagination.setAttribute('page', String(page));
      pagination.setAttribute('total-pages', String(total_pages));
    }

    function openCreateDialog() {
      dialogTitle.textContent = 'Add Agent';
      idInput.value = '';
      nameInput.value = '';
      configInput.value = '{}';
      deleteBtn.classList.add('hidden');
      dialog.showModal();
    }

    function openEditDialog(agent: Agent) {
      dialogTitle.textContent = 'Edit Agent';
      idInput.value = agent.id;
      nameInput.value = agent.name;
      configInput.value = JSON.stringify(agent.config, null, 2);
      deleteBtn.classList.remove('hidden');
      dialog.showModal();
    }

    function closeDialog() {
      dialog.close();
      form.reset();
    }

    async function handleSubmit(e: Event) {
      e.preventDefault();

      let config: Record<string, unknown>;
      try {
        config = JSON.parse(configInput.value || '{}');
      } catch {
        alert('Invalid JSON in configuration');
        return;
      }

      const command: AgentCommand = {
        name: nameInput.value,
        config,
      };

      const id = idInput.value;
      const result = id
        ? await api.put<Agent>(`/agents/${id}`, command)
        : await api.post<Agent>('/agents', command);

      if (result.ok) {
        closeDialog();
        loadAgents();
      } else {
        alert(`Error: ${result.error}`);
      }
    }

    async function handleDelete() {
      const id = idInput.value;
      if (!id) return;

      if (!confirm('Are you sure you want to delete this agent?')) return;

      const result = await api.del(`/agents/${id}`);
      if (result.ok) {
        closeDialog();
        loadAgents();
      } else {
        alert(`Error: ${result.error}`);
      }
    }

    tbody.addEventListener('click', (e) => {
      const row = (e.target as Element).closest('tr');
      if (row?.dataset.id) {
        const agent = cachedAgents.find(a => a.id === row.dataset.id);
        if (agent) openEditDialog(agent);
      }
    });

    searchInput.addEventListener('input', () => {
      clearTimeout(searchTimeout);
      searchTimeout = window.setTimeout(() => {
        currentPage = 1;
        loadAgents();
      }, 300);
    });

    pagination.addEventListener('page-change', ((e: CustomEvent<{ page: number }>) => {
      currentPage = e.detail.page;
      loadAgents();
    }) as EventListener);

    addBtn.addEventListener('click', openCreateDialog);
    cancelBtn.addEventListener('click', closeDialog);
    deleteBtn.addEventListener('click', handleDelete);
    form.addEventListener('submit', handleSubmit);

    loadAgents();
  },
});
```

---

## Phase 10: Integration

### 10.1 Update `web/app/app.go`

Add providers and agents to `views` slice:

```go
var views = []web.ViewDef{
	{Route: "/{$}", Template: "home.html", Title: "Home", Bundle: "app"},
	{Route: "/components", Template: "components.html", Title: "Components", Bundle: "app"},
	{Route: "/providers", Template: "providers.html", Title: "Providers", Bundle: "app"},
	{Route: "/agents", Template: "agents.html", Title: "Agents", Bundle: "app"},
}
```

### 10.2 Update `web/app/client/app.ts`

Add view imports at the end:

```typescript
import '@app/core/views/providers';
import '@app/core/views/agents';
```

---

## File Creation Summary

**New files:**

```
web/app/client/core/
├── api/
│   ├── types.ts
│   ├── client.ts
│   └── index.ts
├── html.ts
├── view/
│   ├── types.ts
│   ├── create.ts
│   └── index.ts
├── domains/
│   ├── providers.ts
│   ├── agents.ts
│   └── index.ts
└── views/
    ├── providers.ts
    └── agents.ts

web/app/client/components/
└── al-pagination.ts

web/app/server/views/
├── providers.html
└── agents.html
```

**Existing files/directories to update (Phase 0):**

- `pkg/web/pages.go` - Rename to `pkg/web/views.go`, refactor types/functions
- `web/app/server/pages/` - Rename to `web/app/server/views/`
- `web/app/app.go` - Update to use refactored web package

**Existing files to update (Phases 1-10):**

- `web/app/client/core/index.ts` - Replace with re-exports
- `web/app/client/components/index.ts` - Export AlPagination
- `web/app/client/design/components.css` - Add dialog and utility styles
- `web/app/app.go` - Add providers/agents view registrations
- `web/app/client/app.ts` - Add view imports

---

## Verification

1. **Build**: `cd web && bun run build` succeeds
2. **Server**: `go run ./cmd/server` starts without errors
3. **Providers view**: Navigate to `/app/providers`
   - Table loads (empty or with data)
   - Search filters results with debounce
   - Pagination shows "Page X of Y" with prev/next
   - Add button opens dialog
   - Create provider, see it in list
   - Click row to edit
   - Delete provider with confirmation
4. **Agents view**: Same verification at `/app/agents`
5. **Tests**: `go test ./tests/...` passes
