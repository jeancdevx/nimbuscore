package api

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID         uuid.UUID `json:"id"`
	Email      string    `json:"email"`
	Name       string    `json:"name"`
	Provider   string    `json:"provider"`
	ProviderID string    `json:"provider_id"`
	AvatarURL  string    `json:"avatar_url,omitempty"`
	Teams      []Team    `json:"teams,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Team struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type WorkspaceStatus string

const (
	WorkspaceStatusPending  WorkspaceStatus = "pending"
	WorkspaceStatusBuilding WorkspaceStatus = "building"
	WorkspaceStatusRunning  WorkspaceStatus = "running"
	WorkspaceStatusStopping WorkspaceStatus = "stopping"
	WorkspaceStatusStopped  WorkspaceStatus = "stopped"
	WorkspaceStatusError    WorkspaceStatus = "error"
	WorkspaceStatusDeleted  WorkspaceStatus = "deleted"
)

type WorkspaceResources struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
	Disk   string `json:"disk"`
	GPU    int    `json:"gpu,omitempty"`
}

type PortMapping struct {
	Port      int    `json:"port"`
	Protocol  string `json:"protocol"`
	Subdomain string `json:"subdomain,omitempty"`
}

type Workspace struct {
	ID           uuid.UUID          `json:"id"`
	Name         string             `json:"name"`
	UserID       uuid.UUID          `json:"user_id"`
	TeamID       *uuid.UUID         `json:"team_id,omitempty"`
	Image        string             `json:"image"`
	Status       WorkspaceStatus    `json:"status"`
	Resources    WorkspaceResources `json:"resources"`
	RepoURL      string             `json:"repo_url,omitempty"`
	Branch       string             `json:"branch,omitempty"`
	Ports        []PortMapping      `json:"ports,omitempty"`
	LastActivity *time.Time         `json:"last_activity,omitempty"`
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
}

type SnapshotStatus string

const (
	SnapshotStatusPending   SnapshotStatus = "pending"
	SnapshotStatusRunning   SnapshotStatus = "running"
	SnapshotStatusCompleted SnapshotStatus = "completed"
	SnapshotStatusFailed    SnapshotStatus = "failed"
)

type Snapshot struct {
	ID          uuid.UUID      `json:"id"`
	WorkspaceID uuid.UUID      `json:"workspace_id"`
	Status      SnapshotStatus `json:"status"`
	Size        int64          `json:"size,omitempty"`
	Message     string         `json:"message,omitempty"`
	StartedAt   time.Time      `json:"started_at"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
}

type PrebuildStatus string

const (
	PrebuildStatusPending   PrebuildStatus = "pending"
	PrebuildStatusBuilding  PrebuildStatus = "building"
	PrebuildStatusAvailable PrebuildStatus = "available"
	PrebuildStatusFailed    PrebuildStatus = "failed"
)

type Prebuild struct {
	ID         uuid.UUID      `json:"id"`
	UserID     uuid.UUID      `json:"user_id"`
	TeamID     *uuid.UUID     `json:"team_id,omitempty"`
	Name       string         `json:"name"`
	Image      string         `json:"image"`
	RepoURL    string         `json:"repo_url"`
	Branch     string         `json:"branch"`
	DevcontainerPath string  `json:"devcontainer_path,omitempty"`
	Status     PrebuildStatus `json:"status"`
	Message    string         `json:"message,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

type Quota struct {
	TeamID       uuid.UUID `json:"team_id"`
	MaxWorkspaces int      `json:"max_workspaces"`
	MaxCPU       string    `json:"max_cpu"`
	MaxMemory    string    `json:"max_memory"`
	MaxDisk      string    `json:"max_disk"`
	MaxGPU       int       `json:"max_gpu,omitempty"`
	UsedWorkspaces int     `json:"used_workspaces,omitempty"`
	UsedCPU      string    `json:"used_cpu,omitempty"`
	UsedMemory   string    `json:"used_memory,omitempty"`
	UsedDisk     string    `json:"used_disk,omitempty"`
	UpdatedAt    time.Time `json:"updated_at"`
}
