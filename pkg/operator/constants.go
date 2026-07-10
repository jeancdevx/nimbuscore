package operator

const (
	GroupName  = "nimbuscore.io"
	APIVersion = "v1alpha1"
	Kind       = "Workspace"
	Plural     = "workspaces"

	PrebuildKind  = "Prebuild"
	PrebuildPlural = "prebuilds"

	LabelWorkspaceID = "nimbuscore.io/workspace-id"
	LabelUserID      = "nimbuscore.io/user-id"
	LabelTeamID      = "nimbuscore.io/team-id"
	LabelPrebuildID  = "nimbuscore.io/prebuild-id"

	AnnotationRepoURL        = "nimbuscore.io/repo-url"
	AnnotationBranch         = "nimbuscore.io/branch"
	AnnotationPorts          = "nimbuscore.io/ports"
	AnnotationIdleTimeout    = "nimbuscore.io/idle-timeout"
	AnnotationPrebuildOutput = "nimbuscore.io/prebuild-output"
)
