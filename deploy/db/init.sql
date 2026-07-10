CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    email TEXT NOT NULL,
    name TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'user',
    provider TEXT NOT NULL,
    provider_id TEXT NOT NULL,
    avatar_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(provider, provider_id)
);

CREATE TABLE IF NOT EXISTS teams (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS workspaces (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id),
    team_id UUID REFERENCES teams(id),
    image TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    cpu TEXT NOT NULL,
    memory TEXT NOT NULL,
    disk TEXT NOT NULL,
    gpu INT NOT NULL DEFAULT 0,
    repo_url TEXT,
    branch TEXT DEFAULT 'main',
    ports JSONB NOT NULL DEFAULT '[]',
    last_activity TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS quotas (
    team_id UUID PRIMARY KEY REFERENCES teams(id),
    max_workspaces INT NOT NULL DEFAULT 10,
    max_cpu TEXT NOT NULL DEFAULT '8',
    max_memory TEXT NOT NULL DEFAULT '16Gi',
    max_disk TEXT NOT NULL DEFAULT '100Gi',
    max_gpu INT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS snapshots (
    id UUID PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    status TEXT NOT NULL DEFAULT 'pending',
    size BIGINT NOT NULL DEFAULT 0,
    message TEXT,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS prebuilds (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    team_id UUID REFERENCES teams(id),
    name TEXT NOT NULL,
    image TEXT NOT NULL,
    repo_url TEXT NOT NULL,
    branch TEXT NOT NULL DEFAULT 'main',
    devcontainer_path TEXT DEFAULT '.devcontainer/devcontainer.json',
    status TEXT NOT NULL DEFAULT 'pending',
    message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS audit_log (
    id UUID PRIMARY KEY,
    action TEXT NOT NULL,
    user_id UUID NOT NULL,
    target_id TEXT,
    target_type TEXT,
    metadata TEXT,
    ip_address TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS billing_records (
    id UUID PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    team_id UUID REFERENCES teams(id),
    user_id UUID NOT NULL,
    cpu_cores FLOAT NOT NULL,
    memory_gb FLOAT NOT NULL,
    disk_gb FLOAT NOT NULL,
    gpu_count INT NOT NULL DEFAULT 0,
    duration_min INT NOT NULL,
    cost FLOAT NOT NULL DEFAULT 0,
    started_at TIMESTAMPTZ NOT NULL,
    ended_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS clusters (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    api_endpoint TEXT NOT NULL,
    kubeconfig TEXT,
    region TEXT,
    provider TEXT,
    enabled BOOLEAN NOT NULL DEFAULT true,
    labels TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
