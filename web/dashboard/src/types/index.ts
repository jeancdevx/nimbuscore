export interface User {
  id: string
  email: string
  name: string
  provider: string
  provider_id: string
  avatar_url?: string
  teams?: Team[]
  created_at: string
  updated_at: string
}

export interface Team {
  id: string
  name: string
  slug: string
  created_at: string
  updated_at: string
}

export type WorkspaceStatus =
  | 'pending'
  | 'building'
  | 'running'
  | 'stopping'
  | 'stopped'
  | 'error'
  | 'deleted'

export interface WorkspaceResources {
  cpu: string
  memory: string
  disk: string
  gpu?: number
}

export interface PortMapping {
  port: number
  protocol: string
  subdomain?: string
}

export interface Workspace {
  id: string
  name: string
  user_id: string
  team_id?: string
  image: string
  status: WorkspaceStatus
  resources: WorkspaceResources
  repo_url?: string
  branch?: string
  ports?: PortMapping[]
  url?: string
  last_activity?: string
  created_at: string
  updated_at: string
}

export interface AuthResponse {
  token: string
  user_id: string
}
