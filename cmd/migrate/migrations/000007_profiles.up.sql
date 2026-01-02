CREATE TABLE profiles (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workflow_name TEXT NOT NULL,
  name TEXT NOT NULL,
  description TEXT,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  UNIQUE(workflow_name, name)
);

CREATE INDEX idx_profiles_workflow_name ON profiles(workflow_name);

CREATE TABLE profile_stages (
  profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
  stage_name TEXT NOT NULL,
  agent_id UUID REFERENCES agents(id),
  system_prompt TEXT,
  options JSONB,
  PRIMARY KEY (profile_id, stage_name)
);
