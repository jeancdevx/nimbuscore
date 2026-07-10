package workspace

import (
	"time"

	"github.com/google/uuid"
	"github.com/nimbuscore/pkg/api"
)

var validTransitions = map[api.WorkspaceStatus][]api.WorkspaceStatus{
	api.WorkspaceStatusPending:  {api.WorkspaceStatusBuilding, api.WorkspaceStatusError, api.WorkspaceStatusDeleted},
	api.WorkspaceStatusBuilding: {api.WorkspaceStatusRunning, api.WorkspaceStatusError, api.WorkspaceStatusStopped},
	api.WorkspaceStatusRunning:  {api.WorkspaceStatusStopping, api.WorkspaceStatusError},
	api.WorkspaceStatusStopping: {api.WorkspaceStatusStopped, api.WorkspaceStatusError},
	api.WorkspaceStatusStopped:  {api.WorkspaceStatusBuilding, api.WorkspaceStatusDeleted, api.WorkspaceStatusError},
	api.WorkspaceStatusError:    {api.WorkspaceStatusBuilding, api.WorkspaceStatusDeleted},
	api.WorkspaceStatusDeleted:  {},
}

func CanTransition(from, to api.WorkspaceStatus) bool {
	allowed, ok := validTransitions[from]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}

type CreateWorkspaceInput struct {
	Name      string
	UserID    uuid.UUID
	TeamID    *uuid.UUID
	Image     string
	Resources api.WorkspaceResources
	RepoURL   string
	Branch    string
	Ports     []api.PortMapping
}

func NewWorkspace(input CreateWorkspaceInput) *api.Workspace {
	now := time.Now().UTC()
	if input.Ports == nil {
		input.Ports = []api.PortMapping{}
	}
	return &api.Workspace{
		ID:        uuid.New(),
		Name:      input.Name,
		UserID:    input.UserID,
		TeamID:    input.TeamID,
		Image:     input.Image,
		Status:    api.WorkspaceStatusPending,
		Resources: input.Resources,
		RepoURL:   input.RepoURL,
		Branch:    input.Branch,
		Ports:     input.Ports,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func CheckQuota(quota api.Quota, wsResources api.WorkspaceResources) bool {
	if quota.MaxWorkspaces > 0 && quota.UsedWorkspaces >= quota.MaxWorkspaces {
		return false
	}
	return true
}
