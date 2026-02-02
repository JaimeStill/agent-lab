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
