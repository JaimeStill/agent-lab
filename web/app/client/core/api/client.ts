import type { Result, PageRequest } from './types';

class ApiClient {
  private baseUrl: string;

  constructor(baseUrl: string = '/api') {
    this.baseUrl = baseUrl;
  }

  private buildUrl(path: string, params?: PageRequest): string {
    const url = new URL(this.baseUrl + path, window.location.origin)
    if (params) {
      if (params.page !== undefined) url.searchParams.set('page', String(params.page));
      if (params.page_size !== undefined) url.searchParams.set('page_size', String(params.page_size));
      if (params.search) url.searchParams.set('search', params.search);
      if (params.sort) url.searchParams.set('sort', params.sort)
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
