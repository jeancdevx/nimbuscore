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
	Ports        []int              `json:"ports,omitempty"`
	LastActivity *time.Time         `json:"last_activity,omitempty"`
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
}
