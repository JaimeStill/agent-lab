import { handleStreamResponse, handleStreamError, type StreamOptions } from './streaming';

const BASE = '/api';

export type Result<T> =
  | { ok: true; data: T }
  | { ok: false; error: string };

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
