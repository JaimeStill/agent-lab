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
