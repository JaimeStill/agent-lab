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
