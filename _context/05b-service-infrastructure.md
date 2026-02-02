# Session 05b: Service Infrastructure - Implementation Guide

## Overview

This session establishes the service infrastructure patterns for the Lit SPA:
- API client with type-safe fetch wrapper
- Pagination types matching Go backend
- SSE streaming via fetch body reader (POST support)
- Provider service as pattern validation
- Validation view demonstrating consumption

## Phase 1: Core Infrastructure

### 1.1 Create core/pagination.ts

**New file**: `web/app/client/core/pagination.ts`

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

export function toQueryString(params: PageRequest): string {
  const entries = Object.entries(params)
    .filter(([, v]) => v !== undefined && v !== null && v !== '')
    .map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(String(v))}`);
  return entries.length > 0 ? `?${entries.join('&')}` : '';
}
```

### 1.2 Create core/streaming.ts

**New file**: `web/app/client/core/streaming.ts`

Streaming chunk types and SSE parsing infrastructure.

```typescript
export interface StreamingChunk {
  id?: string;
  object?: string;
  created?: number;
  model: string;
  choices: Array<{
    index: number;
    delta: {
      role?: string;
      content?: string;
    };
    finish_reason: string | null;
  }>;
}

export type StreamCallback = (chunk: StreamingChunk) => void;
export type StreamErrorCallback = (error: string) => void;
export type StreamCompleteCallback = () => void;

export interface StreamOptions {
  onChunk: StreamCallback;
  onError?: StreamErrorCallback;
  onComplete?: StreamCompleteCallback;
  signal?: AbortSignal;
}

const SSE_DATA_PREFIX = 'data: ';
const SSE_DONE_SIGNAL = '[DONE]';

export async function parseSSE(
  response: Response,
  options: StreamOptions
): Promise<void> {
  const reader = response.body?.getReader();
  if (!reader) {
    options.onError?.('No response body');
    return;
  }

  const decoder = new TextDecoder();
  let buffer = '';

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;

    buffer += decoder.decode(value, { stream: true });
    const lines = buffer.split('\n');
    buffer = lines.pop() ?? '';

    for (const line of lines) {
      if (!line.startsWith(SSE_DATA_PREFIX)) continue;
      const data = line.slice(SSE_DATA_PREFIX.length).trim();

      if (data === SSE_DONE_SIGNAL) {
        options.onComplete?.();
        return;
      }

      try {
        const chunk = JSON.parse(data) as StreamingChunk;
        if ('error' in chunk) {
          options.onError?.((chunk as unknown as { error: string }).error);
          return;
        }
        options.onChunk(chunk);
      } catch {
        // Skip malformed chunks
      }
    }
  }

  options.onComplete?.();
}

export function handleStreamResponse(options: StreamOptions) {
  return async (res: Response) => {
    if (!res.ok) {
      const text = await res.text();
      options.onError?.(text || res.statusText);
      return;
    }
    await parseSSE(res, options);
  };
}

export function handleStreamError(options: StreamOptions) {
  return (err: Error) => {
    if (err.name !== 'AbortError') {
      options.onError?.(err.message);
    }
  };
}
```

### 1.3 Create core/api.ts

**New file**: `web/app/client/core/api.ts`

Minimal fetch wrapper using native `RequestInit`. Callers control request structure and response parsing explicitly.

```typescript
import type { StreamOptions } from './streaming';
import { handleStreamResponse, handleStreamError } from './streaming';

export type Result<T> = { ok: true; data: T } | { ok: false; error: string };

const BASE = '/api';

export async function request<T>(
  path: string,
  init?: RequestInit,
  parse: (res: Response) => Promise<T> = (res) => res.json()
): Promise<Result<T>> {
  try {
    const res = await fetch(`${BASE}${path}`, init);
    if (!res.ok) {
      const text = await res.text();
      return { ok: false, error: text || res.statusText };
    }
    if (res.status === 204) {
      return { ok: true, data: undefined as T };
    }
    return { ok: true, data: await parse(res) };
  } catch (e) {
    return { ok: false, error: e instanceof Error ? e.message : String(e) };
  }
}

export function stream(
  path: string,
  init: RequestInit,
  options: StreamOptions
): AbortController {
  const controller = new AbortController();
  const signal = options.signal ?? controller.signal;

  fetch(`${BASE}${path}`, { ...init, signal })
    .then(handleStreamResponse(options))
    .catch(handleStreamError(options));

  return controller;
}
```

**Usage patterns:**

```typescript
// GET JSON (default parsing)
const result = await request<Provider[]>('/providers');

// POST JSON
const created = await request<Provider>('/providers', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify(data),
});

// POST FormData (browser sets Content-Type with boundary automatically)
const doc = await request<Document>('/documents', {
  method: 'POST',
  body: formData,
});

// GET text response
const raw = await request<string>('/some/text', {}, (res) => res.text());

// GET blob response
const image = await request<Blob>(`/images/${id}/data`, {}, (res) => res.blob());

// DELETE (no response body)
const deleted = await request<void>(`/providers/${id}`, { method: 'DELETE' });

// Streaming POST
const controller = stream(
  '/agents/123/chat/stream',
  {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ prompt: 'Hello' }),
  },
  { onChunk, onError, onComplete }
);
```

### 1.4 Create core/index.ts

**New file**: `web/app/client/core/index.ts`

```typescript
export { request, stream, type Result } from './api';
export * from './pagination';
export * from './streaming';
```

---

## Phase 2: Provider Service

### 2.1 Create providers/types.ts

**New file**: `web/app/client/providers/types.ts`

```typescript
export interface Provider {
  id: string;
  name: string;
  config: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface CreateProvider {
  name: string;
  config: Record<string, unknown>;
}

export interface UpdateProvider {
  name: string;
  config: Record<string, unknown>;
}
```

### 2.2 Create providers/service.ts

**New file**: `web/app/client/providers/service.ts`

```typescript
import { createContext } from '@lit/context';
import { Signal } from '@lit-labs/signals';
import { request, toQueryString } from '@app/core';
import type { PageResult, PageRequest } from '@app/core';
import type { Provider, CreateProvider, UpdateProvider } from './types';

export interface ProviderService {
  providers: Signal.State<Provider[]>;
  loading: Signal.State<boolean>;
  error: Signal.State<string | null>;

  list(params?: PageRequest): Promise<void>;
  find(id: string): Promise<Provider | null>;
  create(data: CreateProvider): Promise<boolean>;
  update(id: string, data: UpdateProvider): Promise<boolean>;
  remove(id: string): Promise<boolean>;
}

export const providerServiceContext = createContext<ProviderService>('provider-service');

export function createProviderService(): ProviderService {
  const providers = new Signal.State<Provider[]>([]);
  const loading = new Signal.State<boolean>(false);
  const error = new Signal.State<string | null>(null);

  return {
    providers,
    loading,
    error,

    async list(params?: PageRequest) {
      loading.set(true);
      error.set(null);

      const query = params ? toQueryString(params) : '';
      const result = await request<PageResult<Provider>>(`/providers${query}`);

      if (result.ok) {
        providers.set(result.data.data);
      } else {
        error.set(result.error);
      }

      loading.set(false);
    },

    async find(id: string) {
      const result = await request<Provider>(`/providers/${id}`);
      if (result.ok) {
        return result.data;
      }
      error.set(result.error);
      return null;
    },

    async create(data: CreateProvider) {
      loading.set(true);
      error.set(null);

      const result = await request<Provider>('/providers', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      });

      loading.set(false);

      if (result.ok) {
        providers.set([...providers.get(), result.data]);
        return true;
      }

      error.set(result.error);
      return false;
    },

    async update(id: string, data: UpdateProvider) {
      loading.set(true);
      error.set(null);

      const result = await request<Provider>(`/providers/${id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      });

      loading.set(false);

      if (result.ok) {
        providers.set(
          providers.get().map((p) => (p.id === id ? result.data : p))
        );
        return true;
      }

      error.set(result.error);
      return false;
    },

    async remove(id: string) {
      loading.set(true);
      error.set(null);

      const result = await request<void>(`/providers/${id}`, {
        method: 'DELETE',
      });

      loading.set(false);

      if (result.ok) {
        providers.set(providers.get().filter((p) => p.id !== id));
        return true;
      }

      error.set(result.error);
      return false;
    },
  };
}
```

### 2.3 Create providers/index.ts

**New file**: `web/app/client/providers/index.ts`

```typescript
export * from './types';
export * from './service';
```

---

## Phase 3: Validation View

### 3.1 Create providers/views/providers-view.css

**New file**: `web/app/client/providers/views/providers-view.css`

```css
:host {
  display: block;
  padding: var(--space-6);
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-6);
}

h1 {
  margin: 0;
  font-size: var(--text-2xl);
  font-weight: 600;
}

.loading {
  color: var(--color-1);
}

.error {
  padding: var(--space-4);
  background: var(--red-bg);
  color: var(--red);
  border-radius: var(--radius-1);
}

.list {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.provider {
  padding: var(--space-4);
  background: var(--bg-1);
  border-radius: var(--radius-1);
}

.provider-name {
  font-weight: 500;
  margin-bottom: var(--space-1);
}

.provider-id {
  font-size: var(--text-sm);
  color: var(--color-1);
  font-family: var(--font-mono);
}

.empty {
  color: var(--color-1);
  text-align: center;
  padding: var(--space-8);
}
```

### 3.2 Create providers/views/providers-view.ts

**New file**: `web/app/client/providers/views/providers-view.ts`

```typescript
import { LitElement, html, nothing, unsafeCSS } from 'lit';
import { customElement } from 'lit/decorators.js';
import { provide } from '@lit/context';
import { SignalWatcher } from '@lit-labs/signals';
import {
  providerServiceContext,
  createProviderService,
  type ProviderService,
} from '../service';
import styles from './providers-view.css?inline';

@customElement('lab-providers-view')
export class ProvidersView extends SignalWatcher(LitElement) {
  static styles = unsafeCSS(styles);

  @provide({ context: providerServiceContext })
  private service: ProviderService = createProviderService();

  connectedCallback() {
    super.connectedCallback();
    this.service.list();
  }

  private renderError() {
    const error = this.service.error.get();
    if (!error) return nothing;
    return html`<div class="error">${error}</div>`;
  }

  private renderLoading() {
    if (!this.service.loading.get()) return nothing;
    return html`<p class="loading">Loading...</p>`;
  }

  private renderList() {
    if (this.service.loading.get()) return nothing;

    const providers = this.service.providers.get();
    if (providers.length === 0) {
      return html`<p class="empty">No providers configured.</p>`;
    }

    return html`
      <div class="list">
        ${providers.map(
          (p) => html`
            <div class="provider">
              <div class="provider-name">${p.name}</div>
              <div class="provider-id">${p.id}</div>
            </div>
          `
        )}
      </div>
    `;
  }

  render() {
    return html`
      <div class="header">
        <h1>Providers</h1>
      </div>
      ${this.renderError()}
      ${this.renderLoading()}
      ${this.renderList()}
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    'lab-providers-view': ProvidersView;
  }
}
```

---

## Phase 4: Route Registration

### 4.1 Update router/routes.ts

**Modify**: `web/app/client/router/routes.ts`

Add providers route:

```typescript
import type { RouteConfig } from './types';

export const routes: Record<string, RouteConfig> = {
  '': { component: 'lab-home-view', title: 'Home' },
  'providers': { component: 'lab-providers-view', title: 'Providers' },
  '*': { component: 'lab-not-found-view', title: 'Not Found' },
};
```

### 4.2 Update app.ts

**Modify**: `web/app/client/app.ts`

Add providers view import:

```typescript
import './design/index.css';

import { Router } from '@app/router';

import './views/home-view';
import './views/not-found-view';
import './providers/views/providers-view';

const router = new Router('app-content');
router.start();
```

### 4.3 Update tsconfig.json paths (if needed)

**Modify**: `web/tsconfig.json`

Ensure `@app/core` alias is configured:

```json
{
  "compilerOptions": {
    "paths": {
      "@app/*": ["./app/client/*"]
    }
  }
}
```

---

## Validation

1. **Build**: `cd web && bun run build`
2. **Run server**: `go run ./cmd/server`
3. **Test**: Navigate to `http://localhost:8080/app/providers`
   - Should see loading state briefly
   - Should see provider list (or empty message)
   - Check browser console for errors

---

## Files Summary

### New Files

| File | Purpose |
|------|---------|
| `web/app/client/core/pagination.ts` | Pagination types and query string builder |
| `web/app/client/core/streaming.ts` | SSE parsing and streaming types |
| `web/app/client/core/api.ts` | `request()` and `stream()` functions |
| `web/app/client/core/index.ts` | Re-exports |
| `web/app/client/providers/types.ts` | Provider domain types |
| `web/app/client/providers/service.ts` | Provider service |
| `web/app/client/providers/index.ts` | Re-exports |
| `web/app/client/providers/views/providers-view.ts` | Validation view |
| `web/app/client/providers/views/providers-view.css` | View styles |

### Modified Files

| File | Change |
|------|--------|
| `web/app/client/router/routes.ts` | Add providers route |
| `web/app/client/app.ts` | Import providers view |
