const SSE_DATA_PREFIX = 'data: ';
const SSE_DONE_SIGNAL = '[DONE]';

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
      return
    }
    await parseSSE(res, options);
  }
}

export function handleStreamError(options: StreamOptions) {
  return (err: Error) => {
    if (err.name !== 'AbortError') {
      options.onError?.(err.message);
    }
  };
}
