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
